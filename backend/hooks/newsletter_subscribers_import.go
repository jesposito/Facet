package hooks

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	maxImportUploadBytes    int64 = 5 << 20
	maxImportRows                 = 50_000
	maxImportErrorsReturned       = 50
)

type importRowError struct {
	Row   int    `json:"row"`
	Email string `json:"email"`
	Error string `json:"error"`
}

type importResult struct {
	Total         int
	Created       int
	Updated       int
	SkippedCap    int
	Failed        int
	Errors        []importRowError
	CreatedTags   []string
	UnknownTags   []string
	createdTagSet map[string]bool
	unknownTagSet map[string]bool
}

func (r *importResult) addError(row int, email, msg string) {
	r.Failed++
	if len(r.Errors) < maxImportErrorsReturned {
		r.Errors = append(r.Errors, importRowError{Row: row, Email: email, Error: msg})
	}
}

func (r *importResult) noteCreatedTag(name string) {
	if r.createdTagSet == nil {
		r.createdTagSet = map[string]bool{}
	}
	key := strings.ToLower(name)
	if r.createdTagSet[key] {
		return
	}
	r.createdTagSet[key] = true
	r.CreatedTags = append(r.CreatedTags, name)
}

func (r *importResult) noteUnknownTag(name string) {
	if r.unknownTagSet == nil {
		r.unknownTagSet = map[string]bool{}
	}
	key := strings.ToLower(name)
	if r.unknownTagSet[key] {
		return
	}
	r.unknownTagSet[key] = true
	r.UnknownTags = append(r.UnknownTags, name)
}

type importColumns struct {
	email  int
	name   int
	status int
	tags   int
}

func parseImportHeader(header []string) (importColumns, error) {
	cols := importColumns{email: -1, name: -1, status: -1, tags: -1}
	for i, raw := range header {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "email", "e-mail", "email address", "email_address":
			if cols.email == -1 {
				cols.email = i
			}
		case "name", "first_name", "first name":
			if cols.name == -1 {
				cols.name = i
			}
		case "status":
			if cols.status == -1 {
				cols.status = i
			}
		case "tags":
			if cols.tags == -1 {
				cols.tags = i
			}
		}
	}
	if cols.email == -1 {
		return cols, fmt.Errorf("CSV must have an 'email' column (header row)")
	}
	return cols, nil
}

func importCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func importSubscribersCSV(app core.App, planConfig *services.PlanConfig, r io.Reader, listID string) (*importResult, error) {
	res := &importResult{}
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err == io.EOF {
		return res, fmt.Errorf("CSV is empty (no header row)")
	}
	if err != nil {
		return res, fmt.Errorf("failed to read CSV header: %w", err)
	}
	cols, err := parseImportHeader(header)
	if err != nil {
		return res, err
	}

	collection, err := app.FindCollectionByNameOrId("subscribers")
	if err != nil {
		return res, fmt.Errorf("subscribers collection not found")
	}

	capHit := false
	rowNum := 1
	for {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		rowNum++
		if readErr != nil {
			res.Total++
			res.addError(rowNum, "", "malformed CSV row")
			continue
		}

		res.Total++
		if res.Total > maxImportRows {
			return res, fmt.Errorf("CSV exceeds the maximum of %d data rows", maxImportRows)
		}

		email := strings.ToLower(importCell(row, cols.email))
		if email == "" || !emailRegex.MatchString(email) {
			res.addError(rowNum, email, "invalid or empty email")
			continue
		}
		name := importCell(row, cols.name)
		statusRaw := strings.ToLower(importCell(row, cols.status))
		tagRaw := importCell(row, cols.tags)

		existing, _ := app.FindRecordsByFilter(
			"subscribers",
			"email = {:email}",
			"",
			1,
			0,
			map[string]any{"email": email},
		)
		if len(existing) > 0 {
			sub := existing[0]
			if name != "" {
				sub.Set("name", name)
			}
			if tagIDs := resolveImportTagNames(app, tagRaw, res); len(tagIDs) > 0 {
				sub.Set("tags", mergeTagIDs(sub.GetStringSlice("tags"), tagIDs))
			}
			if err := app.Save(sub); err != nil {
				res.addError(rowNum, email, "failed to update subscriber")
				continue
			}
			if err := EnsureSubscriberOnList(app, sub.Id, listID); err != nil {
				app.Logger().Warn("csv import: ensure list membership failed", "email", maskEmail(email), "error", err)
			}
			res.Updated++
			continue
		}

		if capHit {
			res.SkippedCap++
			continue
		}
		if err := checkSubscriberCap(app, planConfig); err != nil {
			capHit = true
			res.SkippedCap++
			continue
		}

		status := "active"
		if statusRaw == "active" || statusRaw == "unsubscribed" {
			status = statusRaw
		}

		sub := core.NewRecord(collection)
		sub.Set("email", email)
		sub.Set("name", name)
		sub.Set("source", "import")
		sub.Set("status", status)
		if tagIDs := resolveImportTagNames(app, tagRaw, res); len(tagIDs) > 0 {
			sub.Set("tags", tagIDs)
		}
		if err := app.Save(sub); err != nil {
			res.addError(rowNum, email, "failed to create subscriber")
			continue
		}
		if err := EnsureSubscriberOnList(app, sub.Id, listID); err != nil {
			app.Logger().Warn("csv import: ensure list membership failed", "email", maskEmail(email), "error", err)
		}
		res.Created++
	}

	return res, nil
}

func resolveImportTagNames(app core.App, raw string, res *importResult) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	ids := make([]string, 0)
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		tag := findSubscriberTagByNameCI(app, name)
		if tag == nil {
			tag = createImportSubscriberTag(app, name, res)
			if tag == nil {
				continue
			}
		}
		if seen[tag.Id] {
			continue
		}
		seen[tag.Id] = true
		ids = append(ids, tag.Id)
	}
	return ids
}

func findSubscriberTagByNameCI(app core.App, name string) *core.Record {
	recs, err := app.FindRecordsByFilter(
		"subscriber_tags",
		"name ~ {:name}",
		"",
		200,
		0,
		map[string]any{"name": name},
	)
	if err != nil {
		return nil
	}
	for _, r := range recs {
		if strings.EqualFold(r.GetString("name"), name) {
			return r
		}
	}
	return nil
}

func createImportSubscriberTag(app core.App, name string, res *importResult) *core.Record {
	col, err := app.FindCollectionByNameOrId("subscriber_tags")
	if err != nil {
		res.noteUnknownTag(name)
		return nil
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("subscriber_count", 0)
	if err := app.Save(rec); err != nil {
		if existing := findSubscriberTagByNameCI(app, name); existing != nil {
			return existing
		}
		app.Logger().Warn("csv import: could not create subscriber tag", "tag", name, "error", err)
		res.noteUnknownTag(name)
		return nil
	}
	res.noteCreatedTag(name)
	return rec
}

func mergeTagIDs(existing, add []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(add))
	for _, id := range existing {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range add {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func GetDefaultList(app core.App) (*core.Record, error) {
	if rec, err := app.FindFirstRecordByFilter("newsletter_lists", "is_default = true", nil); err == nil && rec != nil {
		return rec, nil
	}
	return app.FindFirstRecordByFilter("newsletter_lists", "slug = {:slug}", dbx.Params{"slug": "default"})
}

func EnsureSubscriberOnList(app core.App, subscriberID, listID string) error {
	if subscriberID == "" || listID == "" {
		return nil
	}
	existing, err := app.FindRecordsByFilter(
		"subscriber_list_memberships",
		"subscriber_id = {:sid} && list_id = {:lid}",
		"",
		1,
		0,
		map[string]any{"sid": subscriberID, "lid": listID},
	)
	if err == nil && len(existing) > 0 {
		return nil
	}
	col, err := app.FindCollectionByNameOrId("subscriber_list_memberships")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("subscriber_id", subscriberID)
	rec.Set("list_id", listID)
	if err := app.Save(rec); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil
		}
		return err
	}
	return nil
}

func checkSubscriberCap(app core.App, planConfig *services.PlanConfig) error {
	return nil
}

func registerNewsletterSubscriberImportRoute(se *core.ServeEvent, app *pocketbase.PocketBase, planConfig *services.PlanConfig) {
	se.Router.POST("/api/admin/subscribers/import", func(e *core.RequestEvent) error {
		if err := requireSuperuser(e); err != nil {
			return err
		}
		if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
			return respondForbidden(e, "newsletter not available on current plan")
		}

		e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxImportUploadBytes)
		if err := e.Request.ParseMultipartForm(maxImportUploadBytes); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("upload too large or malformed (max %dMB)", maxImportUploadBytes>>20),
			})
		}
		file, _, err := e.Request.FormFile("file")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing CSV file upload (form field 'file')"})
		}
		defer file.Close()

		listID := strings.TrimSpace(e.Request.FormValue("list_id"))
		if listID == "" {
			def, err := GetDefaultList(app)
			if err != nil || def == nil {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "no target list: provide list_id or create a default newsletter list",
				})
			}
			listID = def.Id
		} else if _, err := app.FindRecordById("newsletter_lists", listID); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "target list not found"})
		}

		res, err := importSubscribersCSV(app, planConfig, file, listID)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{"status": "error", "error": err.Error()})
		}
		if res.Errors == nil {
			res.Errors = []importRowError{}
		}
		if res.CreatedTags == nil {
			res.CreatedTags = []string{}
		}
		if res.UnknownTags == nil {
			res.UnknownTags = []string{}
		}
		return e.JSON(http.StatusOK, map[string]any{
			"status":       "ok",
			"total":        res.Total,
			"created":      res.Created,
			"updated":      res.Updated,
			"skipped_cap":  res.SkippedCap,
			"failed":       res.Failed,
			"errors":       res.Errors,
			"created_tags": res.CreatedTags,
			"unknown_tags": res.UnknownTags,
		})
	})
}
