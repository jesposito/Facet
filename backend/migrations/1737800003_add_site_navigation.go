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

		// Add site_nav_enabled boolean field
		collection.Fields.Add(&core.BoolField{
			Name: "site_nav_enabled",
		})

		// Add site_nav_items JSON field for navigation button configuration
		// Structure: [{ "viewId": "xxx", "enabled": true, "label": "About Me" }, ...]
		collection.Fields.Add(&core.JSONField{
			Name:    "site_nav_items",
			MaxSize: 100000,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("site_nav_enabled")
		collection.Fields.RemoveByName("site_nav_items")

		return app.Save(collection)
	})
}
