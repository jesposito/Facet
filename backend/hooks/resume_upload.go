package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// UserError represents an error with both user-friendly and technical details.
type UserError struct {
	Message   string `json:"message"`
	Action    string `json:"action"`
	Technical string `json:"technical"`
}

// NewUserError builds a user-facing error with technical context.
func NewUserError(message, action, technical string) UserError {
	return UserError{Message: message, Action: action, Technical: technical}
}

// File-size and MIME-type limits for resume uploads.
const (
	resumeMaxBytes = 10 * 1024 * 1024 // 10MB hard ceiling
	mimePDF        = "application/pdf"
	mimeDOCX       = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
)

// importPayload mirrors services.ParsedResume but is decoded from the request body
// so the frontend can return only the items the user chose to import.
type importPayload struct {
	Filename   string                  `json:"filename"`
	Visibility string                  `json:"visibility"`
	Parsed     *services.ParsedResume `json:"parsed"`
}

// RegisterResumeUploadHooks wires up the BYO-key resume parsing + import endpoints.
// No credit gating, no managed tiers: the active AI provider configured by the user
// at /admin/settings/integrations is called directly.
func RegisterResumeUploadHooks(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	ai := services.NewAIService(crypto)
	parser := services.NewResumeParser(ai)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// POST /api/resume/upload
		// Extract text → call AI → return parsed JSON. NO records are created here.
		se.Router.POST("/api/resume/upload", func(e *core.RequestEvent) error {
			file, header, err := e.Request.FormFile("file")
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"We couldn't find a file to upload.",
						"Please select a PDF or DOCX resume file and try again.",
						fmt.Sprintf("File upload error: %v", err),
					),
				})
			}
			defer file.Close()

			if header.Size > resumeMaxBytes {
				fileSizeMB := float64(header.Size) / (1024 * 1024)
				return e.JSON(http.StatusRequestEntityTooLarge, map[string]interface{}{
					"error": NewUserError(
						"Your resume file is too large.",
						"Please use a file smaller than 10MB. Try compressing images or saving as a simpler format.",
						fmt.Sprintf("File size: %.2f MB (maximum: 10 MB)", fileSizeMB),
					),
				})
			}

			mimeType := header.Header.Get("Content-Type")
			if mimeType != mimePDF && mimeType != mimeDOCX {
				return e.JSON(http.StatusUnsupportedMediaType, map[string]interface{}{
					"error": NewUserError(
						"This file type isn't supported.",
						"Please upload your resume as a PDF (.pdf) or Word document (.docx).",
						fmt.Sprintf("Unsupported file type: %s (filename: %s)", mimeType, header.Filename),
					),
				})
			}

			providerID := e.Request.FormValue("provider_id")
			provider, err := getActiveProvider(app, crypto, providerID)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"No AI provider configured.",
						"Resume import requires AI to parse the file. Configure a provider in Admin → Settings → Integrations.",
						fmt.Sprintf("Provider error: %v", err),
					),
				})
			}

			fileBytes, err := services.ReadFileBytes(file)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": NewUserError(
						"We couldn't read your file.",
						"Try uploading again, or try a different file format.",
						fmt.Sprintf("File read error: %v (filename: %s)", err, header.Filename),
					),
				})
			}
			if len(fileBytes) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"Your resume file appears to be empty.",
						"Please check the file and try again with a valid PDF or DOCX document.",
						fmt.Sprintf("Empty file: %s (0 bytes)", header.Filename),
					),
				})
			}

			// Extract text.
			resumeText, err := parser.ExtractText(fileBytes, mimeType)
			// Drop the raw bytes immediately — we don't store the upload anywhere.
			fileBytes = nil
			if err != nil {
				status := http.StatusBadRequest
				action := "Try converting your resume to PDF format, or re-save your document and try again."
				if strings.Contains(err.Error(), "no text found") || strings.Contains(err.Error(), "scanned image") {
					status = http.StatusUnprocessableEntity
					action = "Your file appears to contain only images. Run it through OCR (or export as text-based PDF) and try again."
				} else if strings.Contains(err.Error(), "corrupted") {
					action = "Your file may be corrupted. Try opening it and saving a new copy."
				}
				return e.JSON(status, map[string]interface{}{
					"error": NewUserError(
						"We couldn't extract text from your resume.",
						action,
						fmt.Sprintf("Text extraction error: %v", err),
					),
				})
			}

			// Parse with the user's configured AI provider.
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			parsed, err := parser.ParseResume(ctx, provider, resumeText)
			if err != nil {
				// If the AI returned content but JSON parsing failed, surface raw text
				// so the user can copy/paste manually.
				if strings.Contains(err.Error(), "JSON") || strings.Contains(err.Error(), "json") {
					return e.JSON(http.StatusOK, map[string]interface{}{
						"status":   "partial",
						"filename": header.Filename,
						"raw_text": resumeText,
						"error": NewUserError(
							"We extracted your resume text but couldn't structure it.",
							"Copy/paste the relevant pieces into Experience, Projects, etc. manually.",
							fmt.Sprintf("AI parsing error: %v", err),
						),
					})
				}
				action := "Try uploading your resume again. If the problem persists, try a simpler resume format."
				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
					action = "The AI service took too long to respond. Try again in a moment, or try a shorter resume."
				}
				return e.JSON(http.StatusBadGateway, map[string]interface{}{
					"error": NewUserError(
						"We couldn't parse your resume with AI.",
						action,
						fmt.Sprintf("AI parsing error: %v", err),
					),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":   "success",
				"filename": header.Filename,
				"parsed":   parsed,
			})
		}).Bind(apis.RequireAuth())

		// POST /api/resume/import
		// Body: { filename, visibility, parsed: <ParsedResume from /upload, with items the user chose> }
		// Creates records with smart deduplication. The frontend trims `parsed.<section>`
		// arrays to only the items the user checked.
		se.Router.POST("/api/resume/import", func(e *core.RequestEvent) error {
			body, err := readRequestBody(e.Request.Body, 10*1024*1024)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"Couldn't read import payload.",
						"Try again — the request body was unreadable.",
						fmt.Sprintf("Body read error: %v", err),
					),
				})
			}

			var payload importPayload
			if err := json.Unmarshal(body, &payload); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"Invalid import payload.",
						"Try parsing the resume again and resubmit the selection.",
						fmt.Sprintf("JSON decode error: %v", err),
					),
				})
			}
			if payload.Parsed == nil {
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"No parsed resume data provided.",
						"Upload and parse the resume first, then choose items to import.",
						"payload.parsed is nil",
					),
				})
			}

			visibility := payload.Visibility
			switch visibility {
			case "public", "private", "unlisted":
				// ok
			default:
				visibility = "private"
			}
			filename := payload.Filename
			if filename == "" {
				filename = "resume-import"
			}

			imported, deduped, err := createResumeRecordsWithDeduplication(app, payload.Parsed, filename, visibility)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": NewUserError(
						"We couldn't save the selected items.",
						"This might be a temporary database issue. Try again in a moment.",
						fmt.Sprintf("Database error: %v", err),
					),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":       "success",
				"imported":     imported,
				"deduplicated": deduped,
				"counts": map[string]int{
					"experience":     len(imported["experience"]),
					"education":      len(imported["education"]),
					"skills":         len(imported["skills"]),
					"certifications": len(imported["certifications"]),
					"projects":       len(imported["projects"]),
					"awards":         len(imported["awards"]),
					"talks":          len(imported["talks"]),
				},
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// readRequestBody caps body size for the JSON import endpoint.
func readRequestBody(body io.ReadCloser, max int64) ([]byte, error) {
	defer body.Close()
	limited := io.LimitReader(body, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return data, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("body exceeds %d bytes", max)
	}
	return data, nil
}

// normalizeDate coerces a variety of date string shapes into PocketBase date format.
func normalizeDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" || strings.EqualFold(dateStr, "null") || strings.EqualFold(dateStr, "present") {
		return ""
	}
	if idx := strings.IndexAny(dateStr, "T "); idx != -1 {
		dateStr = dateStr[:idx]
	}
	layouts := []struct {
		format string
		suffix string
	}{
		{"2006-01-02", ""},
		{"2006-01", "-01"},
		{"2006", "-01-01"},
	}
	for _, l := range layouts {
		if _, err := time.Parse(l.format, dateStr); err == nil {
			return dateStr + l.suffix
		}
	}
	return ""
}

// generateSlug creates a URL-friendly slug from a title.
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

// createResumeRecordsWithDeduplication writes records to the canonical collections,
// deduping with the same strategy the cloud uses (per-resume for facetable items,
// global for credentials).
func createResumeRecordsWithDeduplication(app *pocketbase.PocketBase, parsed *services.ParsedResume, filename string, visibility string) (map[string][]string, int, error) {
	imported := make(map[string][]string)
	duplicateCount := 0

	if len(parsed.Experience) > 0 {
		expCollection, err := app.FindCollectionByNameOrId("experience")
		if err != nil {
			return nil, 0, fmt.Errorf("experience collection not found: %w", err)
		}
		for _, exp := range parsed.Experience {
			filter := "company = {:company} && title = {:title} && start_date = {:startDate} && import_filename = {:filename}"
			existing, err := app.FindRecordsByFilter(expCollection.Name, filter, "", 1, 0,
				dbx.Params{
					"company":   exp.Company,
					"title":     exp.Title,
					"startDate": normalizeDate(exp.StartDate),
					"filename":  filename,
				})
			if err == nil && len(existing) > 0 {
				duplicateCount++
				continue
			}
			record := core.NewRecord(expCollection)
			record.Set("company", exp.Company)
			record.Set("company_group_id", NormalizeCompanyName(exp.Company))
			record.Set("title", exp.Title)
			record.Set("location", exp.Location)
			if n := normalizeDate(exp.StartDate); n != "" {
				record.Set("start_date", n)
			}
			if n := normalizeDate(exp.EndDate); n != "" {
				record.Set("end_date", n)
			}
			record.Set("description", exp.Description)
			if len(exp.Bullets) > 0 {
				record.Set("bullets", exp.Bullets)
			}
			if len(exp.Skills) > 0 {
				record.Set("skills", exp.Skills)
			}
			record.Set("import_filename", filename)
			record.Set("visibility", visibility)
			record.Set("is_draft", false)
			record.Set("sort_order", 0)
			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-IMPORT] failed to create experience: %v", err)
				continue
			}
			imported["experience"] = append(imported["experience"], record.Id)
		}
	}

	if len(parsed.Education) > 0 {
		eduCollection, err := app.FindCollectionByNameOrId("education")
		if err != nil {
			return nil, 0, fmt.Errorf("education collection not found: %w", err)
		}
		for _, edu := range parsed.Education {
			var filter string
			var params dbx.Params
			if edu.EndDate != "" && edu.EndDate != "null" {
				filter = "institution = {:institution} && degree = {:degree} && field = {:field} && end_date = {:endDate}"
				params = dbx.Params{"institution": edu.Institution, "degree": edu.Degree, "field": edu.Field, "endDate": normalizeDate(edu.EndDate)}
			} else {
				filter = "institution = {:institution} && degree = {:degree} && field = {:field}"
				params = dbx.Params{"institution": edu.Institution, "degree": edu.Degree, "field": edu.Field}
			}
			existing, err := app.FindRecordsByFilter(eduCollection.Name, filter, "", 1, 0, params)
			if err == nil && len(existing) > 0 {
				duplicateCount++
				continue
			}
			record := core.NewRecord(eduCollection)
			record.Set("institution", edu.Institution)
			record.Set("degree", edu.Degree)
			record.Set("field", edu.Field)
			if n := normalizeDate(edu.StartDate); n != "" {
				record.Set("start_date", n)
			}
			if n := normalizeDate(edu.EndDate); n != "" {
				record.Set("end_date", n)
			}
			record.Set("description", edu.Description)
			record.Set("import_filename", filename)
			record.Set("visibility", visibility)
			record.Set("is_draft", false)
			record.Set("sort_order", 0)
			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-IMPORT] failed to create education: %v", err)
				continue
			}
			imported["education"] = append(imported["education"], record.Id)
		}
	}

	if len(parsed.Skills) > 0 {
		skillsCollection, err := app.FindCollectionByNameOrId("skills")
		if err != nil {
			return nil, 0, fmt.Errorf("skills collection not found: %w", err)
		}
		existing := make(map[string]struct{})
		const pageSize = 200
		for offset := 0; ; offset += pageSize {
			page, err := app.FindRecordsByFilter(skillsCollection.Name, "", "", pageSize, offset, nil)
			if err != nil {
				break
			}
			for _, s := range page {
				existing[strings.ToLower(strings.TrimSpace(s.GetString("name")))] = struct{}{}
			}
			if len(page) < pageSize {
				break
			}
		}
		for _, skill := range parsed.Skills {
			norm := strings.ToLower(strings.TrimSpace(skill.Name))
			if _, ok := existing[norm]; ok {
				duplicateCount++
				continue
			}
			record := core.NewRecord(skillsCollection)
			record.Set("name", skill.Name)
			record.Set("category", skill.Category)
			record.Set("proficiency", skill.Proficiency)
			record.Set("import_filename", filename)
			record.Set("visibility", visibility)
			record.Set("sort_order", 0)
			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-IMPORT] failed to create skill: %v", err)
				continue
			}
			imported["skills"] = append(imported["skills"], record.Id)
			existing[norm] = struct{}{}
		}
	}

	if len(parsed.Certifications) > 0 {
		certsCollection, err := app.FindCollectionByNameOrId("certifications")
		if err != nil {
			return nil, 0, fmt.Errorf("certifications collection not found: %w", err)
		}
		for _, cert := range parsed.Certifications {
			existing, err := app.FindRecordsByFilter(certsCollection.Name,
				"name = {:name} && issuer = {:issuer}", "", 1, 0,
				dbx.Params{"name": cert.Name, "issuer": cert.Issuer})
			if err == nil && len(existing) > 0 {
				duplicateCount++
				continue
			}
			record := core.NewRecord(certsCollection)
			record.Set("name", cert.Name)
			record.Set("issuer", cert.Issuer)
			if n := normalizeDate(cert.IssueDate); n != "" {
				record.Set("issue_date", n)
			}
			if n := normalizeDate(cert.ExpiryDate); n != "" {
				record.Set("expiry_date", n)
			}
			record.Set("credential_id", cert.CredentialID)
			record.Set("credential_url", cert.CredentialURL)
			record.Set("import_filename", filename)
			record.Set("visibility", visibility)
			record.Set("is_draft", false)
			record.Set("sort_order", 0)
			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-IMPORT] failed to create certification: %v", err)
				continue
			}
			imported["certifications"] = append(imported["certifications"], record.Id)
		}
	}

	if len(parsed.Projects) > 0 {
		projectsCollection, err := app.FindCollectionByNameOrId("projects")
		if err != nil {
			return nil, 0, fmt.Errorf("projects collection not found: %w", err)
		}
		for _, proj := range parsed.Projects {
			existing, err := app.FindRecordsByFilter(projectsCollection.Name,
				"title = {:title} && import_filename = {:filename}", "", 1, 0,
				dbx.Params{"title": proj.Title, "filename": filename})
			if err == nil && len(existing) > 0 {
				duplicateCount++
				continue
			}
			record := core.NewRecord(projectsCollection)
			record.Set("title", proj.Title)
			record.Set("slug", generateSlug(proj.Title))
			record.Set("summary", proj.Summary)
			record.Set("description", proj.Description)
			if len(proj.TechStack) > 0 {
				record.Set("tech_stack", proj.TechStack)
			}
			if len(proj.Links) > 0 {
				linksJSON, _ := json.Marshal(proj.Links)
				record.Set("links", string(linksJSON))
			}
			record.Set("import_filename", filename)
			record.Set("visibility", visibility)
			record.Set("is_draft", false)
			record.Set("is_featured", false)
			record.Set("sort_order", 0)
			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-IMPORT] failed to create project: %v", err)
				continue
			}
			imported["projects"] = append(imported["projects"], record.Id)
		}
	}

	if len(parsed.Awards) > 0 {
		awardsCollection, err := app.FindCollectionByNameOrId("awards")
		if err == nil {
			for _, award := range parsed.Awards {
				var filter string
				var params dbx.Params
				if award.AwardedAt != "" && award.AwardedAt != "null" {
					filter = "title = {:title} && issuer = {:issuer} && awarded_at = {:awardedAt}"
					params = dbx.Params{"title": award.Title, "issuer": award.Issuer, "awardedAt": normalizeDate(award.AwardedAt)}
				} else {
					filter = "title = {:title} && issuer = {:issuer}"
					params = dbx.Params{"title": award.Title, "issuer": award.Issuer}
				}
				existing, err := app.FindRecordsByFilter(awardsCollection.Name, filter, "", 1, 0, params)
				if err == nil && len(existing) > 0 {
					duplicateCount++
					continue
				}
				record := core.NewRecord(awardsCollection)
				record.Set("title", award.Title)
				record.Set("issuer", award.Issuer)
				if n := normalizeDate(award.AwardedAt); n != "" {
					record.Set("awarded_at", n)
				}
				record.Set("description", award.Description)
				record.Set("import_filename", filename)
				record.Set("visibility", visibility)
				record.Set("is_draft", false)
				record.Set("sort_order", 0)
				if err := app.Save(record); err != nil {
					log.Printf("[RESUME-IMPORT] failed to create award: %v", err)
					continue
				}
				imported["awards"] = append(imported["awards"], record.Id)
			}
		}
	}

	if len(parsed.Talks) > 0 {
		talksCollection, err := app.FindCollectionByNameOrId("talks")
		if err == nil {
			for _, talk := range parsed.Talks {
				existing, err := app.FindRecordsByFilter(talksCollection.Name,
					"title = {:title} && event = {:event}", "", 1, 0,
					dbx.Params{"title": talk.Title, "event": talk.Event})
				if err == nil && len(existing) > 0 {
					duplicateCount++
					continue
				}
				record := core.NewRecord(talksCollection)
				record.Set("title", talk.Title)
				record.Set("slug", generateSlug(talk.Title))
				record.Set("event", talk.Event)
				if n := normalizeDate(talk.Date); n != "" {
					record.Set("date", n)
				}
				record.Set("location", talk.Location)
				record.Set("description", talk.Description)
				record.Set("import_filename", filename)
				record.Set("visibility", visibility)
				record.Set("is_draft", false)
				record.Set("sort_order", 0)
				if err := app.Save(record); err != nil {
					log.Printf("[RESUME-IMPORT] failed to create talk: %v", err)
					continue
				}
				imported["talks"] = append(imported["talks"], record.Id)
			}
		}
	}

	return imported, duplicateCount, nil
}
