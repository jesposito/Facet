package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds subscribers collection for newsletter signups.
// Consolidates fields from 4 fork migrations into a single migration:
// - email, name, source, view_id, status (active/unsubscribed/bounced)
// - unsubscribe_token_hash, unsubscribe_token_enc, token_generated_at
// - bounce_count
// Access is admin-only; custom endpoints handle public access.
func init() {
	m.Register(func(app core.App) error {
		// Idempotency check
		if _, err := app.FindCollectionByNameOrId("subscribers"); err == nil {
			return nil
		}

		subscribers := core.NewBaseCollection("subscribers")
		// All rules nil — custom endpoints handle access, admin UI uses superuser auth
		subscribers.ListRule = nil
		subscribers.ViewRule = nil
		subscribers.UpdateRule = nil
		subscribers.DeleteRule = nil
		subscribers.CreateRule = nil

		subscribers.Fields.Add(&core.EmailField{
			Name:     "email",
			Required: true,
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "name",
			Max:  200,
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "source",
			Max:  100,
		})
		subscribers.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"active", "unsubscribed", "bounced"},
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "unsubscribe_token_hash",
			Max:  200,
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "prev_unsubscribe_token_hash",
			Max:  200,
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "unsubscribe_token_enc",
			Max:  512,
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "token_generated_at",
			Max:  64,
		})
		subscribers.Fields.Add(&core.NumberField{
			Name: "bounce_count",
		})
		subscribers.Fields.Add(&core.TextField{
			Name: "view_id",
			Max:  15,
		})

		subscribers.Indexes = []string{
			"CREATE UNIQUE INDEX idx_subscribers_email ON subscribers(email)",
			"CREATE INDEX idx_subscribers_status ON subscribers(status)",
		}

		return app.Save(subscribers)
	}, func(app core.App) error {
		// Rollback: delete collection
		collection, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return nil // Already gone
		}
		return app.Delete(collection)
	})
}
