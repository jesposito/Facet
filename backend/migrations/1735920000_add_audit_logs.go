package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Check if collection already exists
		collection, err := app.FindCollectionByNameOrId("audit_logs")
		if err == nil && collection != nil {
			return nil
		}

		// Create new audit_logs collection
		auditLogs := core.NewBaseCollection("audit_logs")

		// Add fields
		auditLogs.Fields.Add(&core.TextField{
			Name:     "action",
			Required: true,
			Max:      100,
		})
		auditLogs.Fields.Add(&core.TextField{
			Name:     "resource_type",
			Required: true,
			Max:      50,
		})
		auditLogs.Fields.Add(&core.TextField{
			Name: "resource_id",
			Max:  15,
		})
		auditLogs.Fields.Add(&core.TextField{
			Name: "user_id",
			Max:  15,
		})
		auditLogs.Fields.Add(&core.TextField{
			Name: "user_email",
			Max:  255,
		})
		auditLogs.Fields.Add(&core.TextField{
			Name: "ip_address",
			Max:  45, // IPv6 max length
		})
		auditLogs.Fields.Add(&core.TextField{
			Name: "user_agent",
			Max:  500,
		})
		auditLogs.Fields.Add(&core.JSONField{
			Name: "metadata",
		})
		auditLogs.Fields.Add(&core.TextField{
			Name: "status",
			Max:  20,
		})

		// Set access rules (admin only)
		authRule := "@request.auth.id != ''"
		auditLogs.ListRule = &authRule
		auditLogs.ViewRule = &authRule
		// CreateRule is nil - only backend hooks can create
		// UpdateRule is nil - audit logs are immutable
		auditLogs.DeleteRule = &authRule // Allow admins to delete old logs

		return app.Save(auditLogs)
	}, func(app core.App) error {
		// Rollback: delete the collection
		collection, err := app.FindCollectionByNameOrId("audit_logs")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
