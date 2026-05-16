package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds collections for outbound webhook delivery (self-hosted single-tenant):
//   - webhooks: subscription endpoints (URL + HMAC secret + event filter)
//   - webhook_deliveries: per-attempt log used for the admin "Deliveries" modal
//
// No managed-credit/plan gating — this is a self-hosted feature. Auth rules are
// superuser-only at the API layer (see backend/hooks/webhooks.go); collection
// rules are admin-only as a defense-in-depth measure.
func init() {
	m.Register(func(app core.App) error {
		// Idempotent: bail if already migrated
		if _, err := app.FindCollectionByNameOrId("webhooks"); err == nil {
			return nil
		}

		// Admin-only rule (empty string = superuser only in PocketBase)
		adminRule := ""

		webhooks := core.NewBaseCollection("webhooks")
		webhooks.ListRule = &adminRule
		webhooks.ViewRule = &adminRule
		webhooks.CreateRule = &adminRule
		webhooks.UpdateRule = &adminRule
		webhooks.DeleteRule = &adminRule

		webhooks.Fields.Add(&core.TextField{
			Name:     "label",
			Required: true,
			Max:      200,
		})
		webhooks.Fields.Add(&core.URLField{
			Name:     "url",
			Required: true,
		})
		webhooks.Fields.Add(&core.TextField{
			Name: "secret",
		})
		webhooks.Fields.Add(&core.JSONField{
			Name: "events",
		})
		webhooks.Fields.Add(&core.BoolField{
			Name: "is_active",
		})
		webhooks.Fields.Add(&core.NumberField{
			Name: "failure_count",
		})
		webhooks.Fields.Add(&core.DateField{
			Name: "last_delivered_at",
		})
		webhooks.Fields.Add(&core.DateField{
			Name: "last_failure_at",
		})
		webhooks.Fields.Add(&core.TextField{
			Name: "last_error",
			Max:  500,
		})
		webhooks.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})
		webhooks.Fields.Add(&core.AutodateField{
			Name:     "updated",
			OnCreate: true,
			OnUpdate: true,
		})

		webhooks.Indexes = []string{
			"CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks(is_active)",
		}

		if err := app.Save(webhooks); err != nil {
			return err
		}

		// Deliveries log — keyed off webhooks.id
		deliveries := core.NewBaseCollection("webhook_deliveries")
		deliveries.ListRule = &adminRule
		deliveries.ViewRule = &adminRule
		deliveries.CreateRule = &adminRule
		deliveries.UpdateRule = &adminRule
		deliveries.DeleteRule = &adminRule

		deliveries.Fields.Add(&core.RelationField{
			Name:         "webhook_id",
			MaxSelect:    1,
			Required:     true,
			CollectionId: webhooks.Id,
			CascadeDelete: true,
		})
		deliveries.Fields.Add(&core.TextField{
			Name:     "event_name",
			Required: true,
			Max:      100,
		})
		deliveries.Fields.Add(&core.TextField{
			Name: "request_body",
		})
		deliveries.Fields.Add(&core.NumberField{
			Name: "response_status",
		})
		deliveries.Fields.Add(&core.TextField{
			Name: "response_body",
		})
		deliveries.Fields.Add(&core.NumberField{
			Name: "attempt_count",
		})
		deliveries.Fields.Add(&core.BoolField{
			Name: "succeeded",
		})
		deliveries.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})

		deliveries.Indexes = []string{
			"CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_wh ON webhook_deliveries(webhook_id, created DESC)",
		}

		return app.Save(deliveries)
	}, func(app core.App) error {
		// Rollback: delete in reverse order (deliveries first because of FK)
		for _, name := range []string{"webhook_deliveries", "webhooks"} {
			if c, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(c); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
