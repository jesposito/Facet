package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds newsletter_sends and newsletter_events collections.
// Consolidates two fork migrations:
// - newsletter_sends: tracks sent newsletters with engagement metrics
// - newsletter_events: tracks individual open/click events per subscriber
// Access is admin-only; custom endpoints handle all interactions.
func init() {
	m.Register(func(app core.App) error {
		// ── newsletter_sends ────────────────────────────────────────────────
		if _, err := app.FindCollectionByNameOrId("newsletter_sends"); err != nil {
			sends := core.NewBaseCollection("newsletter_sends")
			// All rules nil — superuser-only via custom endpoints
			sends.ListRule = nil
			sends.ViewRule = nil
			sends.UpdateRule = nil
			sends.DeleteRule = nil
			sends.CreateRule = nil

			sends.Fields.Add(&core.TextField{
				Name:     "subject",
				Required: true,
				Max:      500,
			})
			sends.Fields.Add(&core.TextField{
				Name: "body_markdown",
				Max:  50000,
			})
			sends.Fields.Add(&core.TextField{
				Name: "body_html",
				Max:  100000,
			})
			sends.Fields.Add(&core.SelectField{
				Name:     "status",
				Required: true,
				Values:   []string{"draft", "sending", "sent", "failed"},
			})
			sends.Fields.Add(&core.DateField{
				Name: "sent_at",
			})
			sends.Fields.Add(&core.NumberField{
				Name: "recipient_count",
			})
			sends.Fields.Add(&core.NumberField{
				Name: "sent_count",
			})
			sends.Fields.Add(&core.NumberField{
				Name: "failed_count",
			})
			sends.Fields.Add(&core.NumberField{
				Name: "open_count",
			})
			sends.Fields.Add(&core.NumberField{
				Name: "click_count",
			})

			if err := app.Save(sends); err != nil {
				return err
			}
		}

		// ── newsletter_events ────────────────────────────────────────────────
		if _, err := app.FindCollectionByNameOrId("newsletter_events"); err != nil {
			events := core.NewBaseCollection("newsletter_events")
			// All rules nil — written only by backend tracking endpoints
			events.ListRule = nil
			events.ViewRule = nil
			events.UpdateRule = nil
			events.DeleteRule = nil
			events.CreateRule = nil

			events.Fields.Add(&core.TextField{
				Name:     "send_id",
				Required: true,
				Max:      15,
			})
			events.Fields.Add(&core.TextField{
				Name: "subscriber_id",
				Max:  15,
			})
			events.Fields.Add(&core.SelectField{
				Name:      "event_type",
				Required:  true,
				Values:    []string{"delivered", "opened", "clicked", "bounced", "complained"},
				MaxSelect: 1,
			})
			events.Fields.Add(&core.TextField{
				Name: "url",
				Max:  2048,
			})

			events.AddIndex("idx_newsletter_events_send_type", false, "send_id, event_type", "")
			events.AddIndex("idx_newsletter_events_dedup", true, "send_id, subscriber_id, event_type", "")
			events.AddIndex("idx_newsletter_events_subscriber", false, "subscriber_id", "")

			if err := app.Save(events); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		for _, name := range []string{"newsletter_events", "newsletter_sends"} {
			if col, err := app.FindCollectionByNameOrId(name); err == nil {
				if delErr := app.Delete(col); delErr != nil {
					return delErr
				}
			}
		}
		return nil
	})
}
