package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Creates the system_alerts collection — operator-visible events worth
// surfacing in the admin UI (failed backups, SMTP errors, webhook delivery
// failures, security warnings, etc.).
//
// Unlike audit_logs (which records *every* admin action for compliance),
// system_alerts is a small, actionable inbox: events the admin should
// acknowledge so they know the system is healthy.
//
// Single-user / self-hosted only. No tenant scoping, no tier gating.
func init() {
	m.Register(func(app core.App) error {
		existing, _ := app.FindCollectionByNameOrId("system_alerts")
		if existing != nil {
			return nil
		}

		alerts := core.NewBaseCollection("system_alerts")

		// Severity: info | warning | critical
		alerts.Fields.Add(&core.TextField{
			Name:     "severity",
			Required: true,
			Max:      20,
		})

		// Event key — short machine-readable identifier, e.g. "backup_failed",
		// "smtp_error", "webhook_delivery_failed", "login_lockout".
		alerts.Fields.Add(&core.TextField{
			Name:     "event",
			Required: true,
			Max:      100,
		})

		// Short human-readable title.
		alerts.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      255,
		})

		// Optional longer description (one paragraph, plain text).
		alerts.Fields.Add(&core.TextField{
			Name: "details",
			Max:  2000,
		})

		// Optional structured payload for richer rendering / debugging.
		alerts.Fields.Add(&core.JSONField{
			Name: "metadata",
		})

		// When acknowledged — null until the admin clears it.
		alerts.Fields.Add(&core.DateField{
			Name: "acknowledged_at",
		})

		// Auth rule — admin-only via PocketBase superuser auth.
		authRule := "@request.auth.id != ''"
		alerts.ListRule = &authRule
		alerts.ViewRule = &authRule
		alerts.CreateRule = nil // Only backend hooks create alerts
		alerts.UpdateRule = nil // Use /api/admin/system-alerts/{id}/acknowledge
		alerts.DeleteRule = &authRule

		if err := app.Save(alerts); err != nil {
			return err
		}

		// Indexes for the common query: unacknowledged-only, sorted by created desc.
		app.DB().NewQuery("CREATE INDEX IF NOT EXISTS idx_system_alerts_created ON system_alerts(created)").Execute()
		app.DB().NewQuery("CREATE INDEX IF NOT EXISTS idx_system_alerts_ack ON system_alerts(acknowledged_at)").Execute()

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("system_alerts")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
