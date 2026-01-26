package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil // Collection doesn't exist, skip
		}

		// Add skills_category_order field if it doesn't exist
		if collection.Fields.GetByName("skills_category_order") == nil {
			collection.Fields.Add(&core.JSONField{
				Name: "skills_category_order",
			})
			return app.Save(collection)
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("skills_category_order")
		if field != nil {
			collection.Fields.RemoveByName("skills_category_order")
			return app.Save(collection)
		}

		return nil
	})
}
