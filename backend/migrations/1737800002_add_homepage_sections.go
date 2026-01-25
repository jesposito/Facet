package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil // Collection doesn't exist yet, skip
		}

		// Add homepage_sections JSON field for per-section item selections
		// Structure: { "experience": { "enabled": true, "items": ["id1", "id2"], "layout": "default" }, ... }
		collection.Fields.Add(&core.JSONField{
			Name:    "homepage_sections",
			MaxSize: 500000,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("homepage_sections")
		if field != nil {
			collection.Fields.RemoveByName("homepage_sections")
			return app.Save(collection)
		}

		return nil
	})
}
