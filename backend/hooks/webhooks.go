// Package hooks — admin REST + lifecycle wiring for webhooks.
//
// Endpoints (all superuser-only):
//   - GET    /api/admin/webhooks                    list
//   - POST   /api/admin/webhooks                    create (auto-generates HMAC secret)
//   - PATCH  /api/admin/webhooks/{id}               update
//   - DELETE /api/admin/webhooks/{id}               delete
//   - POST   /api/admin/webhooks/{id}/test          fire a synthetic event
//   - GET    /api/admin/webhooks/{id}/deliveries    paginated delivery log
//   - GET    /api/admin/webhooks/events             list registered event names
//
// Lifecycle:
//   - comments.go: dispatched on OnRecordAfterCreateSuccess for the comments collection.
//   - posts/talks/custom_content: dispatched when is_draft transitions from true to false.
//   - newsletter: dispatched on confirmed subscribe (create with status="active").
package hooks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterWebhookHooks wires admin REST endpoints + record lifecycle dispatchers.
// The returned service is stored on the app so other hook files can call Dispatch.
func RegisterWebhookHooks(app *pocketbase.PocketBase, rl *services.RateLimitService) *services.WebhookService {
	svc := services.NewWebhookService(app)
	app.Store().Set("facet_webhook_service", svc)

	// Validate URLs on direct PocketBase CRUD too, in case someone goes through
	// the SDK instead of the admin REST handler.
	app.OnRecordCreateRequest("webhooks").BindFunc(func(e *core.RecordRequestEvent) error {
		if u := e.Record.GetString("url"); u != "" {
			if err := services.ValidateWebhookURL(u); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("webhooks").BindFunc(func(e *core.RecordRequestEvent) error {
		if u := e.Record.GetString("url"); u != "" {
			if err := services.ValidateWebhookURL(u); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
		}
		return e.Next()
	})

	// ============= Lifecycle wiring =============
	// post.published / talks/custom_content treated as posts here for parity.
	publishables := []string{"posts", "talks", "custom_content"}
	for _, coll := range publishables {
		name := coll
		app.OnRecordAfterUpdateSuccess(name).BindFunc(func(e *core.RecordEvent) error {
			// Fire on transition draft -> published. We don't have the prior
			// record handy from PocketBase's after-update event, so we use the
			// conservative rule: "fire whenever is_draft=false on save".
			// Receivers are documented to deduplicate by id+published_at.
			if !e.Record.GetBool("is_draft") {
				svc.Dispatch(services.EventPostPublished, map[string]any{
					"id":         e.Record.Id,
					"collection": name,
					"title":      e.Record.GetString("title"),
					"slug":       e.Record.GetString("slug"),
				})
			}
			return e.Next()
		})
	}

	// newsletter.subscribed — fires when a subscriber row is created in "active" state.
	app.OnRecordAfterCreateSuccess("subscribers").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "active" {
			svc.Dispatch(services.EventNewsletterSubscribed, map[string]any{
				"id":     e.Record.Id,
				"email":  e.Record.GetString("email"),
				"name":   e.Record.GetString("name"),
				"source": e.Record.GetString("source"),
			})
		}
		return e.Next()
	})

	// comment.created — fires when a public comment row is created.
	app.OnRecordAfterCreateSuccess("comments").BindFunc(func(e *core.RecordEvent) error {
		svc.Dispatch(services.EventCommentCreated, map[string]any{
			"id":           e.Record.Id,
			"content_type": e.Record.GetString("content_type"),
			"content_id":   e.Record.GetString("content_id"),
			"author_name":  e.Record.GetString("author_name"),
			"author_email": e.Record.GetString("author_email"),
			"body":         e.Record.GetString("body"),
			"status":       e.Record.GetString("status"),
		})
		return e.Next()
	})

	// ============= Admin REST endpoints =============
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		svc.Start()

		// GET /api/admin/webhooks/events — registered event catalog for the UI picker
		se.Router.GET("/api/admin/webhooks/events", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			return e.JSON(http.StatusOK, map[string]any{"events": services.RegisteredEvents})
		}))

		// GET /api/admin/webhooks — list
		se.Router.GET("/api/admin/webhooks", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			records, err := app.FindRecordsByFilter("webhooks", "id != ''", "-created", 100, 0)
			if err != nil {
				return e.JSON(http.StatusOK, map[string]any{"data": []any{}})
			}
			out := make([]map[string]any, 0, len(records))
			for _, r := range records {
				var events []string
				if raw := r.Get("events"); raw != nil {
					b, _ := json.Marshal(raw)
					_ = json.Unmarshal(b, &events)
				}
				out = append(out, map[string]any{
					"id":                r.Id,
					"label":             r.GetString("label"),
					"url":               r.GetString("url"),
					"events":            events,
					"is_active":         r.GetBool("is_active"),
					"failure_count":     r.GetInt("failure_count"),
					"last_delivered_at": r.GetString("last_delivered_at"),
					"last_failure_at":   r.GetString("last_failure_at"),
					"last_error":        r.GetString("last_error"),
					"created":           r.GetString("created"),
				})
			}
			return e.JSON(http.StatusOK, map[string]any{"data": out})
		}))

		// POST /api/admin/webhooks — create (auto-generated secret returned ONCE)
		se.Router.POST("/api/admin/webhooks", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			var req struct {
				Label    string   `json:"label"`
				URL      string   `json:"url"`
				Events   []string `json:"events"`
				IsActive *bool    `json:"is_active"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}
			if req.Label == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "label is required"})
			}
			if req.URL == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
			}
			if err := services.ValidateWebhookURL(req.URL); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
			if len(req.Events) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "events is required"})
			}
			valid := map[string]bool{}
			for _, ev := range services.RegisteredEvents {
				valid[ev] = true
			}
			for _, ev := range req.Events {
				if !valid[ev] {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event: " + ev})
				}
			}

			secretBytes := make([]byte, 32)
			if _, err := rand.Read(secretBytes); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate secret"})
			}
			secret := hex.EncodeToString(secretBytes)

			collection, err := app.FindCollectionByNameOrId("webhooks")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "webhooks collection not found"})
			}
			rec := core.NewRecord(collection)
			rec.Set("label", req.Label)
			rec.Set("url", req.URL)
			rec.Set("events", req.Events)
			rec.Set("secret", secret)
			active := true
			if req.IsActive != nil {
				active = *req.IsActive
			}
			rec.Set("is_active", active)
			rec.Set("failure_count", 0)
			if err := app.Save(rec); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save webhook"})
			}
			return e.JSON(http.StatusCreated, map[string]any{
				"id":        rec.Id,
				"label":     req.Label,
				"url":       req.URL,
				"events":    req.Events,
				"is_active": active,
				"secret":    secret,
			})
		}))

		// PATCH /api/admin/webhooks/{id}
		se.Router.PATCH("/api/admin/webhooks/{id}", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			rec, err := app.FindRecordById("webhooks", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
			}
			var req struct {
				Label    *string  `json:"label"`
				URL      *string  `json:"url"`
				Events   []string `json:"events"`
				IsActive *bool    `json:"is_active"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}
			if req.Label != nil {
				rec.Set("label", *req.Label)
			}
			if req.URL != nil {
				if err := services.ValidateWebhookURL(*req.URL); err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
				}
				rec.Set("url", *req.URL)
			}
			if req.Events != nil {
				valid := map[string]bool{}
				for _, ev := range services.RegisteredEvents {
					valid[ev] = true
				}
				for _, ev := range req.Events {
					if !valid[ev] {
						return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event: " + ev})
					}
				}
				rec.Set("events", req.Events)
			}
			if req.IsActive != nil {
				rec.Set("is_active", *req.IsActive)
				if *req.IsActive {
					// Re-enabling resets the auto-disable counter.
					rec.Set("failure_count", 0)
					rec.Set("last_error", "")
				}
			}
			if err := app.Save(rec); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update"})
			}
			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		}))

		// DELETE /api/admin/webhooks/{id}
		se.Router.DELETE("/api/admin/webhooks/{id}", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			rec, err := app.FindRecordById("webhooks", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
			}
			if err := app.Delete(rec); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
			}
			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		}))

		// POST /api/admin/webhooks/dispatch-test — fire synthetic event to ALL
		// matching active webhooks via normal Dispatch pipeline. Body: {"event": "post.published"}.
		// Returns count of webhooks the event was enqueued to so the admin sees
		// at-a-glance how many endpoints will receive it.
		se.Router.POST("/api/admin/webhooks/dispatch-test", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			var req struct {
				Event string `json:"event"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}
			if req.Event == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "event is required"})
			}
			valid := false
			for _, ev := range services.RegisteredEvents {
				if ev == req.Event {
					valid = true
					break
				}
			}
			if !valid {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "unknown event"})
			}

			// Count how many active webhooks subscribe to this event so the UI
			// can report a meaningful "fired to N webhooks" message.
			hooks, _ := app.FindRecordsByFilter("webhooks", "is_active = true", "", 200, 0)
			matched := 0
			for _, h := range hooks {
				var events []string
				if raw := h.Get("events"); raw != nil {
					b, _ := json.Marshal(raw)
					_ = json.Unmarshal(b, &events)
				}
				for _, ev := range events {
					if ev == req.Event {
						matched++
						break
					}
				}
			}

			svc.Dispatch(req.Event, services.SamplePayload(req.Event))

			return e.JSON(http.StatusOK, map[string]any{
				"event":    req.Event,
				"enqueued": matched,
			})
		}))

		// POST /api/admin/webhooks/{id}/test — synchronous test fire
		se.Router.POST("/api/admin/webhooks/{id}/test", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			rec, err := app.FindRecordById("webhooks", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
			}
			status, body, derr := svc.SendNow(
				rec.GetString("url"),
				rec.GetString("secret"),
				"test",
				map[string]any{"message": "This is a test webhook delivery from Facet."},
			)
			result := map[string]any{
				"status_code":   status,
				"response_body": body,
				"succeeded":     derr == nil && status >= 200 && status < 300,
			}
			if derr != nil {
				result["error"] = derr.Error()
			}
			return e.JSON(http.StatusOK, result)
		}))

		// GET /api/admin/webhooks/{id}/deliveries — paginated log
		se.Router.GET("/api/admin/webhooks/{id}/deliveries", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			if _, err := app.FindRecordById("webhooks", id); err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
			}

			page := parsePositiveInt(e.Request.URL.Query().Get("page"), 1, 0)
			perPage := parsePositiveInt(e.Request.URL.Query().Get("per_page"), 25, 100)
			offset := (page - 1) * perPage

			recs, err := app.FindRecordsByFilter(
				"webhook_deliveries",
				"webhook_id = {:wh}",
				"-created",
				perPage, offset,
				map[string]any{"wh": id},
			)
			if err != nil {
				return e.JSON(http.StatusOK, map[string]any{"data": []any{}, "page": page, "per_page": perPage})
			}

			out := make([]map[string]any, 0, len(recs))
			for _, r := range recs {
				out = append(out, map[string]any{
					"id":              r.Id,
					"event_name":      r.GetString("event_name"),
					"request_body":    r.GetString("request_body"),
					"response_status": r.GetInt("response_status"),
					"response_body":   r.GetString("response_body"),
					"attempt_count":   r.GetInt("attempt_count"),
					"succeeded":       r.GetBool("succeeded"),
					"created":         r.GetString("created"),
				})
			}
			return e.JSON(http.StatusOK, map[string]any{
				"data":     out,
				"page":     page,
				"per_page": perPage,
			})
		}))

		return se.Next()
	})

	return svc
}
