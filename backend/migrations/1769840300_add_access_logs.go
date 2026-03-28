package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Creates the access_logs collection for analytics tracking.
// All fields are created at once. Access rules are nil (admin-only via custom endpoints).
// Indexes are added for common query patterns.
func init() {
	m.Register(func(app core.App) error {
		// Check if collection already exists (idempotent)
		existing, _ := app.FindCollectionByNameOrId("access_logs")
		if existing != nil {
			return nil
		}

		accessLogs := core.NewBaseCollection("access_logs")

		// All fields for access log entries
		accessLogs.Fields.Add(&core.TextField{
			Name:     "view_id",
			Required: true,
			Max:      15,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "view_slug",
			Max:  255,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "referrer",
			Max:  2048,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "country_code",
			Max:  2,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "path",
			Max:  500,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "user_agent",
			Max:  500,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "ip_hash",
			Max:  64,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "share_token_id",
			Max:  50,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "utm_source",
			Max:  255,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "utm_medium",
			Max:  255,
		})
		accessLogs.Fields.Add(&core.TextField{
			Name: "utm_campaign",
			Max:  255,
		})

		// Admin-only via nil rules — custom endpoints handle access
		accessLogs.ListRule = nil
		accessLogs.ViewRule = nil
		accessLogs.CreateRule = nil
		accessLogs.UpdateRule = nil
		accessLogs.DeleteRule = nil

		if err := app.Save(accessLogs); err != nil {
			return err
		}

		// Add indexes for common query patterns
		app.DB().NewQuery("CREATE INDEX IF NOT EXISTS idx_access_logs_created ON access_logs(created)").Execute()
		app.DB().NewQuery("CREATE INDEX IF NOT EXISTS idx_access_logs_view ON access_logs(view_id)").Execute()

		return nil
	}, func(app core.App) error {
		// Rollback: delete the collection
		collection, err := app.FindCollectionByNameOrId("access_logs")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
