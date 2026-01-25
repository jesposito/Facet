package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return err
		}

		// Add JSON field to store homepage custom content configuration
		// Format: [{ "id": "custom_content_id", "enabled": true }]
		collection.Fields.Add(&core.JSONField{
			Name:     "homepage_custom_content",
			MaxSize:  100000,
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("homepage_custom_content")
		if field != nil {
			collection.Fields.RemoveByName("homepage_custom_content")
			return app.Save(collection)
		}

		return nil
	})
}
