package hooks

// system_alerts.go — operator-facing inbox for system events worth
// surfacing to the self-hosted admin (failed backups, SMTP errors,
// webhook delivery failures, etc.).
//
// This is the self-hosted analog of the cloud /admin/alerts page, which in
// the multi-tenant edition aggregates provisioning + capacity + tenant-down
// events from a control plane. Here we keep it simple: a small, acknowledgable
// inbox backed by the `system_alerts` PocketBase collection.
//
// Endpoints (all admin-only, require superuser auth):
//
//	GET    /api/admin/system-alerts                — list (filterable)
//	POST   /api/admin/system-alerts/{id}/acknowledge — ack one
//	POST   /api/admin/system-alerts/acknowledge-all  — ack all unacknowledged
//	DELETE /api/admin/system-alerts/{id}            — permanently remove
//
// CreateSystemAlert is the helper other hooks should call to emit alerts.
// It is intentionally lenient: it logs and returns silently on failure so
// emitting an alert never breaks the hot path that triggered it.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SeverityInfo / SeverityWarning / SeverityCritical are the canonical severity
// strings stored in system_alerts.severity. Frontend maps these to colours and
// icons; backend emitters should use these constants to stay in sync.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// SystemAlertInput captures the fields a backend emitter provides when
// creating a new alert. event + title + severity are required; the rest are
// optional context.
type SystemAlertInput struct {
	Severity string         // info | warning | critical (defaults to "info")
	Event    string         // short machine identifier, e.g. "backup_failed"
	Title    string         // human-readable summary line
	Details  string         // optional longer description
	Metadata map[string]any // optional structured payload
}

// CreateSystemAlert persists a new alert. Failures are logged but never
// propagated — the caller's primary operation must not fail just because the
// alert sink is unavailable.
func CreateSystemAlert(app core.App, input SystemAlertInput) {
	severity := strings.TrimSpace(input.Severity)
	if severity == "" {
		severity = SeverityInfo
	}
	if severity != SeverityInfo && severity != SeverityWarning && severity != SeverityCritical {
		severity = SeverityInfo
	}

	event := strings.TrimSpace(input.Event)
	title := strings.TrimSpace(input.Title)
	if event == "" || title == "" {
		app.Logger().Warn("system_alerts: refusing to create alert with empty event or title",
			"event", event, "title", title)
		return
	}

	collection, err := app.FindCollectionByNameOrId("system_alerts")
	if err != nil {
		app.Logger().Error("system_alerts: collection lookup failed", "error", err)
		return
	}

	record := core.NewRecord(collection)
	record.Set("severity", severity)
	record.Set("event", event)
	record.Set("title", title)
	if input.Details != "" {
		record.Set("details", input.Details)
	}
	if len(input.Metadata) > 0 {
		record.Set("metadata", input.Metadata)
	}

	if err := app.Save(record); err != nil {
		app.Logger().Error("system_alerts: save failed", "event", event, "error", err)
	}
}

// RegisterSystemAlertHooks wires the admin endpoints for listing,
// acknowledging, and clearing system alerts.
func RegisterSystemAlertHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// -------------------------------------------------------------
		// GET /api/admin/system-alerts
		//   ?acknowledged=true|false   (default: all)
		//   ?severity=info|warning|critical
		//   ?limit=N (1..200, default 100)
		// -------------------------------------------------------------
		se.Router.GET("/api/admin/system-alerts", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			limit := 100
			if l := e.Request.URL.Query().Get("limit"); l != "" {
				if n, err := strconv.Atoi(l); err == nil {
					if n < 1 {
						n = 1
					}
					if n > 200 {
						n = 200
					}
					limit = n
				}
			}

			// Build PocketBase filter DSL clauses.
			var clauses []string
			params := dbx.Params{}
			switch e.Request.URL.Query().Get("acknowledged") {
			case "true":
				clauses = append(clauses, "acknowledged_at != ''")
			case "false":
				clauses = append(clauses, "acknowledged_at = ''")
			}
			if sev := strings.TrimSpace(e.Request.URL.Query().Get("severity")); sev != "" {
				if sev == SeverityInfo || sev == SeverityWarning || sev == SeverityCritical {
					clauses = append(clauses, "severity = {:sev}")
					params["sev"] = sev
				}
			}
			filter := strings.Join(clauses, " && ")

			records, err := app.FindRecordsByFilter(
				"system_alerts",
				filter,
				"-created",
				limit,
				0,
				params,
			)
			if err != nil {
				app.Logger().Error("system_alerts: list failed", "error", err)
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to load alerts"})
			}

			out := make([]map[string]any, 0, len(records))
			for _, r := range records {
				out = append(out, serializeAlert(r))
			}

			// Unacknowledged count is cheap and useful for the sidebar badge.
			unacknowledged, _ := app.CountRecords("system_alerts",
				dbx.NewExp("acknowledged_at = ''"))

			return respondJSON(e, http.StatusOK, map[string]any{
				"alerts":         out,
				"total":          len(out),
				"unacknowledged": unacknowledged,
			})
		})

		// -------------------------------------------------------------
		// POST /api/admin/system-alerts/{id}/acknowledge
		// -------------------------------------------------------------
		se.Router.POST("/api/admin/system-alerts/{id}/acknowledge", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			if id == "" {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "id required"})
			}
			record, err := app.FindRecordById("system_alerts", id)
			if err != nil {
				return respondNotFound(e, "alert not found")
			}
			// Idempotent: setting ack on an already-ack'd alert is a no-op.
			if record.GetString("acknowledged_at") == "" {
				// PocketBase DateField format is space-separated with milliseconds and a Z suffix.
				record.Set("acknowledged_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
				if err := app.Save(record); err != nil {
					app.Logger().Error("system_alerts: acknowledge save failed", "id", id, "error", err)
					return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to acknowledge"})
				}
			}
			return respondJSON(e, http.StatusOK, map[string]any{
				"acknowledged":    true,
				"acknowledged_at": record.GetString("acknowledged_at"),
			})
		})

		// -------------------------------------------------------------
		// POST /api/admin/system-alerts/acknowledge-all
		// -------------------------------------------------------------
		se.Router.POST("/api/admin/system-alerts/acknowledge-all", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
			result, err := app.DB().NewQuery(
				"UPDATE system_alerts SET acknowledged_at = {:now} WHERE acknowledged_at = ''",
			).Bind(dbx.Params{"now": now}).Execute()
			if err != nil {
				app.Logger().Error("system_alerts: acknowledge-all failed", "error", err)
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to acknowledge"})
			}
			affected, _ := result.RowsAffected()
			return respondJSON(e, http.StatusOK, map[string]any{
				"acknowledged": affected,
			})
		})

		// -------------------------------------------------------------
		// DELETE /api/admin/system-alerts/{id}
		// -------------------------------------------------------------
		se.Router.DELETE("/api/admin/system-alerts/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			if id == "" {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "id required"})
			}
			record, err := app.FindRecordById("system_alerts", id)
			if err != nil {
				return respondNotFound(e, "alert not found")
			}
			if err := app.Delete(record); err != nil {
				app.Logger().Error("system_alerts: delete failed", "id", id, "error", err)
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
			}
			return respondJSON(e, http.StatusOK, map[string]any{"deleted": true})
		})

		return se.Next()
	})
}

// serializeAlert converts a PocketBase record into the JSON shape the
// frontend consumes.
func serializeAlert(r *core.Record) map[string]any {
	return map[string]any{
		"id":              r.Id,
		"severity":        r.GetString("severity"),
		"event":           r.GetString("event"),
		"title":           r.GetString("title"),
		"details":         r.GetString("details"),
		"metadata":        r.Get("metadata"),
		"acknowledged_at": r.GetString("acknowledged_at"),
		"created":         r.GetString("created"),
	}
}

