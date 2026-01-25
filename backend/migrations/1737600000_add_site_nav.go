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

		// Add site_nav JSON field for navigation configuration
		// Structure: { enabled: bool, home_label: string, items: [{view_id, label, visible}] }
		collection.Fields.Add(&core.JSONField{
			Name: "site_nav",
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

		collection.Fields.RemoveByName("site_nav")
		return app.Save(collection)
	})
}
