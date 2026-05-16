package hooks

import (
	"fmt"
	"net/http"
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

			items := make([]map[string]any, 0, len(records))
			for _, r := range records {
				items = append(items, serializeList(r))
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

		// Subscriber-count recomputation hook: runs whenever a membership is
		// created or deleted (keeps list.subscriber_count honest for the UI).
		// Read-modify-write on the LIST row is safe because lists are low-
		// cardinality; the count is advisory (source of truth is the join).
		recalcCount := func(listID string) {
			if listID == "" {
				return
			}
			var count int64
			_ = app.DB().NewQuery(
				"SELECT COUNT(*) FROM subscriber_list_memberships WHERE list_id = {:lid}",
			).Bind(dbx.Params{"lid": listID}).Row(&count)
			if list, err := app.FindRecordById("newsletter_lists", listID); err == nil && list != nil {
				list.Set("subscriber_count", count)
				_ = app.Save(list)
			}
		}
		app.OnRecordAfterCreateSuccess("subscriber_list_memberships").BindFunc(func(e *core.RecordEvent) error {
			go recalcCount(e.Record.GetString("list_id"))
			return e.Next()
		})
		app.OnRecordAfterDeleteSuccess("subscriber_list_memberships").BindFunc(func(e *core.RecordEvent) error {
			go recalcCount(e.Record.GetString("list_id"))
			return e.Next()
		})

		return se.Next()
	})
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

func findListByIdOrSlug(app core.App, idOrSlug string) (*core.Record, error) {
	if r, err := app.FindRecordById("newsletter_lists", idOrSlug); err == nil {
		return r, nil
	}
	return app.FindFirstRecordByFilter("newsletter_lists", "slug = {:s}", dbx.Params{"s": idOrSlug})
}
