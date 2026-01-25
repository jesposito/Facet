package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
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

		newTypes := []string{
			"email", "phone", "linkedin", "github", "twitter",
			"facebook", "instagram", "youtube", "mastodon", "bluesky",
			"website", "portfolio", "blog", "custom",
			"whatsapp", "telegram", "discord", "slack", "other",
		}

		selectField.Values = newTypes

		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
