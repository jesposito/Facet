package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds missing contact method types: whatsapp, telegram, discord, slack, other
// Fixes GitHub issue #277
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("contact_methods")
		if err != nil {
			return nil // Collection doesn't exist, skip
		}

		// Find the type field and update its allowed values
		typeField := collection.Fields.GetByName("type")
		if typeField == nil {
			return nil
		}

		selectField, ok := typeField.(*core.SelectField)
		if !ok {
			return nil
		}

		// Updated list with all contact types
		selectField.Values = []string{
			"email",
			"phone",
			"linkedin",
			"github",
			"twitter",
			"facebook",
			"instagram",
			"youtube",
			"mastodon",
			"bluesky",
			"website",
			"portfolio",
			"blog",
			"whatsapp",
			"telegram",
			"discord",
			"slack",
			"other",
			"custom", // Keep for backwards compatibility
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: restore original values (optional, but good practice)
		collection, err := app.FindCollectionByNameOrId("contact_methods")
		if err != nil {
			return nil
		}

		typeField := collection.Fields.GetByName("type")
		if typeField == nil {
			return nil
		}

		selectField, ok := typeField.(*core.SelectField)
		if !ok {
			return nil
		}

		selectField.Values = []string{
			"email",
			"phone",
			"linkedin",
			"github",
			"twitter",
			"facebook",
			"instagram",
			"youtube",
			"mastodon",
			"bluesky",
			"website",
			"portfolio",
			"blog",
			"custom",
		}

		return app.Save(collection)
	})
}
