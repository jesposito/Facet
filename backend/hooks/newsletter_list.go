package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// slugRegex matches lowercase alphanumeric with single hyphens.
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// listRequest is the shared shape for create and update.
type listRequest struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	SenderName     string `json:"sender_name"`
	ReplyTo        string `json:"reply_to"`
	WelcomeSubject string `json:"welcome_subject"`
	WelcomeHTML    string `json:"welcome_html"`
	IsActive       *bool  `json:"is_active,omitempty"`
	// IsPublic lets visitors join the list from the public signup form.
	IsPublic *bool `json:"is_public,omitempty"`
}

// RegisterNewsletterListHooks wires up the admin CRUD for newsletter_lists.
// All admin endpoints are superuser-gated. Self-hosted has no plan cap —
// `cap` is always 0 (unlimited) in the list response.
func RegisterNewsletterListHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		gate := func(e *core.RequestEvent) error {
			return requireSuperuser(e)
		}

		// GET /api/admin/newsletter-lists — list all.
		se.Router.GET("/api/admin/newsletter-lists", func(e *core.RequestEvent) error {
			if err := gate(e); err != nil {
				return err
			}

			records, err := app.FindRecordsByFilter(
				"newsletter_lists", "id != ''", "-is_default,name", 0, 0,
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load lists"})
			}

			active := activeListMemberCounts(app)
			items := make([]map[string]any, 0, len(records))
			for _, r := range records {
				item := serializeList(r)
				item["active_subscriber_count"] = active[r.Id]
				items = append(items, item)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"data":     items,
				"cap":      0, // self-hosted: unlimited
				"count":    len(items),
				"at_limit": false,
			})
		})

		// POST /api/admin/newsletter-lists — create.
		se.Router.POST("/api/admin/newsletter-lists", func(e *core.RequestEvent) error {
			if err := gate(e); err != nil {
				return err
			}

			var req listRequest
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}
			if err := validateListRequest(&req, true); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Slug uniqueness.
			if existing, _ := app.FindFirstRecordByFilter(
				"newsletter_lists", "slug = {:slug}", dbx.Params{"slug": req.Slug},
			); existing != nil {
				return e.JSON(http.StatusConflict, map[string]string{"error": "slug already in use"})
			}

			collection, err := app.FindCollectionByNameOrId("newsletter_lists")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "lists collection not found"})
			}

			record := core.NewRecord(collection)
			applyListRequest(record, &req, false)
			record.Set("is_default", false)
			record.Set("is_active", reqBoolOrDefault(req.IsActive, true))
			record.Set("is_public", reqBoolOrDefault(req.IsPublic, false))
			record.Set("subscriber_count", 0)

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create list"})
			}

			return e.JSON(http.StatusOK, serializeList(record))
		})

		// GET /api/admin/newsletter-lists/{id} — view one.
		se.Router.GET("/api/admin/newsletter-lists/{id}", func(e *core.RequestEvent) error {
			if err := gate(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			record, err := findListByIdOrSlug(app, id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "list not found"})
			}
			return e.JSON(http.StatusOK, serializeList(record))
		})

		// PATCH /api/admin/newsletter-lists/{id} — update.
		se.Router.PATCH("/api/admin/newsletter-lists/{id}", func(e *core.RequestEvent) error {
			if err := gate(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			record, err := findListByIdOrSlug(app, id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "list not found"})
			}

			var req listRequest
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}
			if err := validateListRequest(&req, false); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Slug uniqueness (when slug is changing).
			if req.Slug != "" && req.Slug != record.GetString("slug") {
				if existing, _ := app.FindFirstRecordByFilter(
					"newsletter_lists", "slug = {:slug} && id != {:id}",
					dbx.Params{"slug": req.Slug, "id": record.Id},
				); existing != nil {
					return e.JSON(http.StatusConflict, map[string]string{"error": "slug already in use"})
				}
			}

			applyListRequest(record, &req, true)
			if req.IsActive != nil {
				record.Set("is_active", *req.IsActive)
			}
			if req.IsPublic != nil {
				record.Set("is_public", *req.IsPublic)
			}

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update list"})
			}

			return e.JSON(http.StatusOK, serializeList(record))
		})

		// POST /api/admin/newsletter-lists/{id}/set-default — atomically flip default.
		se.Router.POST("/api/admin/newsletter-lists/{id}/set-default", func(e *core.RequestEvent) error {
			if err := gate(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			target, err := findListByIdOrSlug(app, id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "list not found"})
			}
			if target.GetBool("is_default") {
				return e.JSON(http.StatusOK, serializeList(target))
			}

			// Atomic swap: clear all, set target. Partial-unique index on
			// is_default=1 forbids two defaults, so we must clear before set.
			err = app.RunInTransaction(func(txApp core.App) error {
				if _, err := txApp.DB().NewQuery(
					"UPDATE newsletter_lists SET is_default = 0 WHERE is_default = 1",
				).Execute(); err != nil {
					return err
				}
				target.Set("is_default", true)
				return txApp.Save(target)
			})
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to set default"})
			}

			return e.JSON(http.StatusOK, serializeList(target))
		})

		// DELETE /api/admin/newsletter-lists/{id} — delete.
		// Forbid delete if list is default OR the last list.
		se.Router.DELETE("/api/admin/newsletter-lists/{id}", func(e *core.RequestEvent) error {
			if err := gate(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			record, err := findListByIdOrSlug(app, id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "list not found"})
			}
			if record.GetBool("is_default") {
				return e.JSON(http.StatusConflict, map[string]string{
					"error": "cannot delete the default list; set another list as default first",
				})
			}
			total, _ := app.CountRecords("newsletter_lists")
			if total <= 1 {
				return e.JSON(http.StatusConflict, map[string]string{
					"error": "cannot delete the only remaining list",
				})
			}

			if err := app.Delete(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete list"})
			}
			// Cascade on memberships comes from the RelationField CascadeDelete
			// flag set in the migration; any subscribers now orphaned from
			// this list keep their other memberships.
			return e.JSON(http.StatusOK, map[string]any{"status": "deleted", "id": record.Id})
		})

		return se.Next()
	})

	// Keep list.subscriber_count in step with the membership table. These run
	// once the membership change has committed (after the surrounding
	// transaction, if there is one), and e.App is the app to write with then.
	app.OnRecordAfterCreateSuccess("subscriber_list_memberships").BindFunc(func(e *core.RecordEvent) error {
		if err := recountListMembers(e.App, e.Record.GetString("list_id")); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordAfterDeleteSuccess("subscriber_list_memberships").BindFunc(func(e *core.RecordEvent) error {
		if err := recountListMembers(e.App, e.Record.GetString("list_id")); err != nil {
			return err
		}
		return e.Next()
	})
}

// recountListMembers sets a list's subscriber_count from the membership table
// in one statement. Counting inside the UPDATE means two memberships changing
// at once can't each write a count the other has already made stale, and
// touching only that column means a list edit saved in between isn't
// overwritten. A list that has just been deleted matches no row, which is fine.
func recountListMembers(app core.App, listID string) error {
	if listID == "" {
		return nil
	}
	_, err := app.DB().NewQuery(`UPDATE newsletter_lists SET subscriber_count = (
		SELECT COUNT(*) FROM subscriber_list_memberships WHERE list_id = {:lid}
	) WHERE id = {:lid}`).Bind(dbx.Params{"lid": listID}).Execute()
	if err != nil {
		return fmt.Errorf("recount members of list %s: %w", listID, err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────
// helpers

func serializeList(r *core.Record) map[string]any {
	return map[string]any{
		"id":               r.Id,
		"name":             r.GetString("name"),
		"slug":             r.GetString("slug"),
		"description":      r.GetString("description"),
		"sender_name":      r.GetString("sender_name"),
		"reply_to":         r.GetString("reply_to"),
		"welcome_subject":  r.GetString("welcome_subject"),
		"welcome_html":     r.GetString("welcome_html"),
		"is_default":       r.GetBool("is_default"),
		"is_active":        r.GetBool("is_active"),
		"is_public":        r.GetBool("is_public"),
		"subscriber_count": r.GetInt("subscriber_count"),
		"created":          r.GetString("created"),
		"updated":          r.GetString("updated"),
	}
}

func validateListRequest(req *listRequest, isCreate bool) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.Description = strings.TrimSpace(req.Description)
	req.SenderName = strings.TrimSpace(req.SenderName)
	req.ReplyTo = strings.TrimSpace(req.ReplyTo)
	req.WelcomeSubject = strings.TrimSpace(req.WelcomeSubject)

	if isCreate {
		if req.Name == "" {
			return fmt.Errorf("name is required")
		}
		if req.Slug == "" {
			return fmt.Errorf("slug is required")
		}
	}
	if req.Slug != "" && !slugRegex.MatchString(req.Slug) {
		return fmt.Errorf("slug must be lowercase alphanumeric with hyphens")
	}
	if len(req.Name) > 100 {
		return fmt.Errorf("name too long (max 100 chars)")
	}
	if len(req.Slug) > 50 {
		return fmt.Errorf("slug too long (max 50 chars)")
	}
	if len(req.Description) > 500 {
		return fmt.Errorf("description too long (max 500 chars)")
	}
	if len(req.WelcomeSubject) > 200 {
		return fmt.Errorf("welcome subject too long (max 200 chars)")
	}
	if len(req.WelcomeHTML) > 100_000 {
		return fmt.Errorf("welcome HTML too long (max 100,000 chars)")
	}
	return nil
}

func applyListRequest(r *core.Record, req *listRequest, patch bool) {
	if !patch || req.Name != "" {
		r.Set("name", req.Name)
	}
	if !patch || req.Slug != "" {
		r.Set("slug", req.Slug)
	}
	// String fields with empty string meaning "clear it" — only set if the
	// caller actually sent a value. Admin UI always sends full payload.
	r.Set("description", req.Description)
	r.Set("sender_name", req.SenderName)
	r.Set("reply_to", req.ReplyTo)
	r.Set("welcome_subject", req.WelcomeSubject)
	r.Set("welcome_html", req.WelcomeHTML)
}

func reqBoolOrDefault(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

var (
	errNewsletterListNotFound = errors.New("newsletter list not found")
	errNewsletterListInactive = errors.New("newsletter list is inactive")
)

// newsletterAudience is who one send goes to. List is nil when the send
// targets every active subscriber.
type newsletterAudience struct {
	List          *core.Record
	SubscriberIDs []string
}

func (a *newsletterAudience) listName() string {
	if a == nil || a.List == nil {
		return ""
	}
	return a.List.GetString("name")
}

// selectNewsletterAudience returns the active, confirmed subscribers a send
// would reach. An empty listID means every active subscriber. A listID that
// doesn't match a list is an error, never a quiet fall back to everyone.
// Both the confirmation count and the send itself use this, so the number
// the owner approves is the number that gets mail.
func selectNewsletterAudience(app core.App, listID string) (*newsletterAudience, error) {
	listID = strings.TrimSpace(listID)
	audience := &newsletterAudience{}
	query := "SELECT id FROM subscribers WHERE status = 'active' ORDER BY id"
	params := dbx.Params{}
	if listID != "" {
		list, err := findListByIdOrSlug(app, listID)
		if err != nil || list == nil {
			return nil, errNewsletterListNotFound
		}
		if !list.GetBool("is_active") {
			return nil, errNewsletterListInactive
		}
		audience.List = list
		query = `SELECT s.id FROM subscribers s
			INNER JOIN subscriber_list_memberships m ON m.subscriber_id = s.id
			WHERE m.list_id = {:lid} AND s.status = 'active'
			GROUP BY s.id ORDER BY s.id`
		params["lid"] = list.Id
	}

	var rows []struct {
		ID string `db:"id"`
	}
	if err := app.DB().NewQuery(query).Bind(params).All(&rows); err != nil {
		return nil, err
	}
	audience.SubscriberIDs = make([]string, 0, len(rows))
	for _, r := range rows {
		audience.SubscriberIDs = append(audience.SubscriberIDs, r.ID)
	}
	return audience, nil
}

// newsletterPlaceholder matches the five documented placeholders, with or
// without spaces inside the braces. Anything else is left as written.
var newsletterPlaceholder = regexp.MustCompile(`\{\{\s*(subscriber\.name|subscriber\.email|list\.name|site\.name|site\.url)\s*\}\}`)

// newsletterPersonalization holds the values that are the same for every
// recipient of one send.
type newsletterPersonalization struct {
	Enabled  bool
	ListName string
	SiteName string
	SiteURL  string
}

func newNewsletterPersonalization(app core.App, enabled bool, audience *newsletterAudience) *newsletterPersonalization {
	siteName := app.Settings().Meta.AppName
	if profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil); err == nil && profile != nil {
		if name := profile.GetString("name"); name != "" {
			siteName = name
		}
	}
	return &newsletterPersonalization{
		Enabled:  enabled,
		ListName: audience.listName(),
		SiteName: siteName,
		SiteURL:  publicAppURL(app),
	}
}

func (p *newsletterPersonalization) value(key, name, email string) string {
	switch key {
	case "subscriber.name":
		return name
	case "subscriber.email":
		return email
	case "list.name":
		return p.ListName
	case "site.name":
		return p.SiteName
	case "site.url":
		return p.SiteURL
	}
	return ""
}

func (p *newsletterPersonalization) replace(text, name, email string, escape func(string) string) string {
	if p == nil || !p.Enabled {
		return text
	}
	return newsletterPlaceholder.ReplaceAllStringFunc(text, func(match string) string {
		key := newsletterPlaceholder.FindStringSubmatch(match)[1]
		return escape(p.value(key, name, email))
	})
}

// HTML fills placeholders in HTML, escaping each value so a subscriber's
// name can't add markup to the email.
func (p *newsletterPersonalization) HTML(body, name, email string) string {
	return p.replace(body, name, email, html.EscapeString)
}

// Subject fills placeholders in a subject line. Line breaks are dropped so a
// value can't add mail headers.
func (p *newsletterPersonalization) Subject(subject, name, email string) string {
	return p.replace(subject, name, email, func(v string) string {
		return strings.NewReplacer("\r", " ", "\n", " ").Replace(v)
	})
}

// key names the audience for duplicate detection: "all", or "list:<id>".
func (a *newsletterAudience) key() string {
	if a == nil || a.List == nil {
		return "all"
	}
	return "list:" + a.List.Id
}

func (a *newsletterAudience) listID() string {
	if a == nil || a.List == nil {
		return ""
	}
	return a.List.Id
}

// activeListMemberCounts returns, per list, how many members are active and
// confirmed: the number a send to that list would actually reach.
// subscriber_count on the list counts every membership, including pending
// and unsubscribed readers.
func activeListMemberCounts(app core.App) map[string]int {
	var rows []struct {
		ListID string `db:"list_id"`
		Count  int    `db:"n"`
	}
	_ = app.DB().NewQuery(`SELECT m.list_id AS list_id, COUNT(DISTINCT s.id) AS n
		FROM subscriber_list_memberships m
		INNER JOIN subscribers s ON s.id = m.subscriber_id
		WHERE s.status = 'active'
		GROUP BY m.list_id`).All(&rows)
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.ListID] = r.Count
	}
	return out
}

var (
	errSignupListUnavailable = errors.New("newsletter list not available")
	errNoDefaultList         = errors.New("no default newsletter list")
)

// publicSignupListOpen reports whether visitors may join list from the
// public form: it must be active, and either the default list or one the
// owner marked public.
func publicSignupListOpen(list *core.Record) bool {
	return list != nil && list.GetBool("is_active") && (list.GetBool("is_default") || list.GetBool("is_public"))
}

// resolveSignupList picks the list a public signup joins. No list means the
// default list. A named list must be open to the public; a private, inactive
// or missing list is refused the same way, so the form can't be used to
// discover private lists. It never quietly swaps in another list.
func resolveSignupList(app core.App, requested string) (*core.Record, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		list, err := app.FindFirstRecordByFilter("newsletter_lists", "is_default = true")
		if err != nil || list == nil {
			return nil, errNoDefaultList
		}
		if !list.GetBool("is_active") {
			return nil, errSignupListUnavailable
		}
		return list, nil
	}
	list, err := findListByIdOrSlug(app, requested)
	if err != nil || !publicSignupListOpen(list) {
		return nil, errSignupListUnavailable
	}
	return list, nil
}

// isListMember reports whether a subscriber already belongs to a list.
func isListMember(app core.App, subscriberID, listID string) bool {
	m, err := app.FindFirstRecordByFilter(
		"subscriber_list_memberships",
		"subscriber_id = {:s} && list_id = {:l}",
		dbx.Params{"s": subscriberID, "l": listID},
	)
	return err == nil && m != nil
}

// ensureListMembership adds a subscriber to a list once. The unique index on
// (subscriber_id, list_id) backs this up if two confirmations race.
func ensureListMembership(app core.App, subscriberID, listID string) error {
	if isListMember(app, subscriberID, listID) {
		return nil
	}
	col, err := app.FindCollectionByNameOrId("subscriber_list_memberships")
	if err != nil {
		return err
	}
	m := core.NewRecord(col)
	m.Set("subscriber_id", subscriberID)
	m.Set("list_id", listID)
	if err := app.Save(m); err != nil {
		if isListMember(app, subscriberID, listID) {
			return nil
		}
		return err
	}
	return nil
}

// newsletterContentHash identifies what a send says, for duplicate checks.
func newsletterContentHash(subject, bodyHTML string) string {
	sum := sha256.Sum256([]byte(subject + "\x00" + bodyHTML))
	return hex.EncodeToString(sum[:])
}

// newsletterRequestKeyPattern is the shape of a compose page request key.
// crypto.randomUUID() values fit.
var newsletterRequestKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

var errPerRecipientLink = errors.New("links can't contain {{ subscriber.name }} or {{ subscriber.email }}; each link is tracked and checked against the newsletter as sent, so per-reader addresses can't be followed safely. Put the placeholder in the text instead")

var errLinkPlaceholderNeedsPersonalize = errors.New("a link contains a placeholder such as {{ site.url }}; turn on personalization so it's filled in, or write the address out")

var placeholderURLDecoder = strings.NewReplacer("%7B", "{", "%7b", "{", "%7D", "}", "%7d", "}", "%20", " ")

// subscriberPlaceholder matches the placeholders that differ per reader.
var subscriberPlaceholder = regexp.MustCompile(`\{\{\s*subscriber\.(name|email)\s*\}\}`)

// prepareNewsletterLinks fills placeholders inside href and src attributes
// before links are tracked, so the tracked address and the stored body agree.
//
// Placeholders that are the same for every reader ({{ list.name }},
// {{ site.name }}, {{ site.url }}) are filled here. site.url goes in as an
// address; the others are path-escaped so they can't change the URL's shape.
// Per-reader placeholders are refused: the click route only follows links
// found in the stored body, and putting a reader's address in a URL would
// also hand it to whichever site the link points at.
//
// With personalization off a placeholder in a link would go out as a broken
// address, so that is refused too, with a message saying how to fix it.
func prepareNewsletterLinks(bodyHTML string, p *newsletterPersonalization) (string, error) {
	enabled := p != nil && p.Enabled
	var refused, needsPersonalize bool
	out := hrefSrcRegex.ReplaceAllStringFunc(bodyHTML, func(match string) string {
		parts := hrefSrcRegex.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		// Markdown percent-encodes braces and spaces in link addresses, so
		// [read](https://x/{{ list.name }}) arrives as %7B%7B%20list.name...
		value := placeholderURLDecoder.Replace(html.UnescapeString(parts[2]))
		if !strings.Contains(value, "{{") {
			return match
		}
		if subscriberPlaceholder.MatchString(value) {
			refused = true
			return match
		}
		if !newsletterPlaceholder.MatchString(value) {
			return match
		}
		if !enabled {
			needsPersonalize = true
			return match
		}
		filled := newsletterPlaceholder.ReplaceAllStringFunc(value, func(ph string) string {
			key := newsletterPlaceholder.FindStringSubmatch(ph)[1]
			v := p.value(key, "", "")
			if key == "site.url" {
				return v
			}
			return url.PathEscape(v)
		})
		filled = strings.ReplaceAll(filled, " ", "%20")
		return parts[1] + html.EscapeString(filled) + parts[3]
	})
	if refused {
		return "", errPerRecipientLink
	}
	if needsPersonalize {
		return "", errLinkPlaceholderNeedsPersonalize
	}
	return out, nil
}

func findListByIdOrSlug(app core.App, idOrSlug string) (*core.Record, error) {
	if r, err := app.FindRecordById("newsletter_lists", idOrSlug); err == nil {
		return r, nil
	}
	return app.FindFirstRecordByFilter("newsletter_lists", "slug = {:s}", dbx.Params{"s": idOrSlug})
}
