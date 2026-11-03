package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Public newsletter signups now wait in "pending" until the address owner
// follows a confirmation link. The token is stored as an HMAC, like the
// unsubscribe token, and confirmation_sent_at limits how often one address
// can be sent a confirmation email.
func init() {
	m.Register(func(app core.App) error {
		subscribers, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return nil
		}
		if status, ok := subscribers.Fields.GetByName("status").(*core.SelectField); ok {
			hasPending := false
			for _, v := range status.Values {
				if v == "pending" {
					hasPending = true
				}
			}
			if !hasPending {
				status.Values = append(status.Values, "pending")
			}
		}
		if subscribers.Fields.GetByName("confirm_token_hash") == nil {
			subscribers.Fields.Add(&core.TextField{Name: "confirm_token_hash", Max: 200, Hidden: true})
		}
		if subscribers.Fields.GetByName("confirmation_sent_at") == nil {
			subscribers.Fields.Add(&core.TextField{Name: "confirmation_sent_at", Max: 64})
		}
		return app.Save(subscribers)
	}, func(app core.App) error {
		subscribers, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return nil
		}
		for _, name := range []string{"confirm_token_hash", "confirmation_sent_at"} {
			if f := subscribers.Fields.GetByName(name); f != nil {
				subscribers.Fields.RemoveById(f.GetId())
			}
		}
		return app.Save(subscribers)
	})
}
