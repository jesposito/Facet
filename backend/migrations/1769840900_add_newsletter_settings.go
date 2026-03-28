package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds newsletter sender settings to the profile collection.
// Allows the site owner to set a custom sender name and reply-to address
// that will appear in newsletter emails, separate from the global SMTP settings.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil // profile collection may not exist in all setups
		}

		changed := false

		if collection.Fields.GetByName("newsletter_sender_name") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "newsletter_sender_name",
				Max:  200,
			})
			changed = true
		}

		if collection.Fields.GetByName("newsletter_sender_email") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "newsletter_sender_email",
				Max:  255,
			})
			changed = true
		}

		if collection.Fields.GetByName("newsletter_reply_to") == nil {
			collection.Fields.Add(&core.EmailField{
				Name: "newsletter_reply_to",
			})
			changed = true
		}

		if !changed {
			return nil
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		for _, name := range []string{"newsletter_sender_name", "newsletter_sender_email", "newsletter_reply_to"} {
			if f := collection.Fields.GetByName(name); f != nil {
				collection.Fields.RemoveById(f.GetId())
			}
		}

		return app.Save(collection)
	})
}
