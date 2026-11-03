package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Newsletter delivery keeps one row per recipient so an interrupted send can
// resume without mailing anyone twice:
//   - newsletter_recipients: pending, in-flight, accepted, failed (explicit
//     SMTP refusal, safe to retry), uncertain (never retried automatically)
//     and skipped. created/updated give resumes a stable order.
//   - newsletter_sends gains per-state counts and the template and
//     personalisation a resume needs, plus "partial" and "interrupted".
//   - subscribers.confirmation_attempted_at bounds confirmation retries
//     separately from the 24h cooldown after SMTP accepts one.
//   - newsletter_sends and newsletter_events gain the created and updated
//     timestamps that send history and exports sort by. They were created
//     without them, so rows that already exist keep empty values: their real
//     times are unknown and are not invented.
func init() {
	m.Register(func(app core.App) error {
		sends, err := app.FindCollectionByNameOrId("newsletter_sends")
		if err != nil {
			return err
		}
		addNewsletterTimestamps(sends)
		for _, name := range []string{"pending_count", "uncertain_count", "skipped_count"} {
			if sends.Fields.GetByName(name) == nil {
				sends.Fields.Add(&core.NumberField{Name: name, OnlyInt: true})
			}
		}
		if sends.Fields.GetByName("template_slug") == nil {
			sends.Fields.Add(&core.TextField{Name: "template_slug", Max: 100})
		}
		if sends.Fields.GetByName("personalize") == nil {
			sends.Fields.Add(&core.BoolField{Name: "personalize"})
		}
		if status, ok := sends.Fields.GetByName("status").(*core.SelectField); ok {
			status.Values = []string{"draft", "sending", "sent", "partial", "failed", "interrupted"}
		}
		if err := app.Save(sends); err != nil {
			return err
		}

		if events, err := app.FindCollectionByNameOrId("newsletter_events"); err == nil {
			if addNewsletterTimestamps(events) {
				if err := app.Save(events); err != nil {
					return err
				}
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		subscribers, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return err
		}
		if subscribers.Fields.GetByName("confirmation_attempted_at") == nil {
			subscribers.Fields.Add(&core.TextField{Name: "confirmation_attempted_at", Max: 64, Hidden: true})
			if err := app.Save(subscribers); err != nil {
				return err
			}
		}

		if _, err := app.FindCollectionByNameOrId("newsletter_recipients"); err == nil {
			return nil
		}
		recipients := core.NewBaseCollection("newsletter_recipients")
		recipients.ListRule = nil
		recipients.ViewRule = nil
		recipients.UpdateRule = nil
		recipients.DeleteRule = nil
		recipients.CreateRule = nil
		recipients.Fields.Add(&core.TextField{Name: "send_id", Required: true, Max: 15})
		recipients.Fields.Add(&core.TextField{Name: "subscriber_id", Required: true, Max: 15})
		recipients.Fields.Add(&core.SelectField{
			Name: "state", Required: true, MaxSelect: 1,
			Values: []string{"pending", "in-flight", "accepted", "failed", "uncertain", "skipped"},
		})
		recipients.Fields.Add(&core.TextField{Name: "error", Max: 2000})
		recipients.Fields.Add(&core.DateField{Name: "claimed_at"})
		recipients.Fields.Add(&core.DateField{Name: "completed_at"})
		recipients.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		recipients.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		recipients.AddIndex("idx_newsletter_recipient_unique", true, "send_id, subscriber_id", "")
		recipients.AddIndex("idx_newsletter_recipient_state", false, "send_id, state", "")
		return app.Save(recipients)
	}, func(app core.App) error {
		if recipients, err := app.FindCollectionByNameOrId("newsletter_recipients"); err == nil {
			if err := app.Delete(recipients); err != nil {
				return err
			}
		}
		if subscribers, err := app.FindCollectionByNameOrId("subscribers"); err == nil {
			if field := subscribers.Fields.GetByName("confirmation_attempted_at"); field != nil {
				subscribers.Fields.RemoveById(field.GetId())
				if err := app.Save(subscribers); err != nil {
					return err
				}
			}
		}
		sends, err := app.FindCollectionByNameOrId("newsletter_sends")
		if err != nil {
			return nil
		}
		// Fold the new statuses into the old ones before they disappear.
		if _, err := app.DB().NewQuery(`UPDATE newsletter_sends SET status = 'sent' WHERE status = 'partial'`).Execute(); err != nil {
			return err
		}
		if _, err := app.DB().NewQuery(`UPDATE newsletter_sends SET status = 'failed' WHERE status = 'interrupted'`).Execute(); err != nil {
			return err
		}
		// created and updated stay on newsletter_sends and newsletter_events.
		// Nothing records whether this migration added them or an install
		// already had them, and dropping them would lose real timestamps.
		for _, name := range []string{"pending_count", "uncertain_count", "skipped_count", "template_slug", "personalize"} {
			if field := sends.Fields.GetByName(name); field != nil {
				sends.Fields.RemoveById(field.GetId())
			}
		}
		if status, ok := sends.Fields.GetByName("status").(*core.SelectField); ok {
			status.Values = []string{"draft", "sending", "sent", "failed"}
		}
		return app.Save(sends)
	})
}

// addNewsletterTimestamps adds whichever of created and updated is missing
// and reports whether it changed the collection.
func addNewsletterTimestamps(collection *core.Collection) bool {
	changed := false
	if collection.Fields.GetByName("created") == nil {
		collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		changed = true
	}
	if collection.Fields.GetByName("updated") == nil {
		collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		changed = true
	}
	return changed
}
