package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		if collection.Fields.GetByName("homepage_sections") != nil {
			return nil
		}

		collection.Fields.Add(&core.JSONField{
			Name:     "homepage_sections",
			MaxSize:  100000,
			Required: false,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("homepage_sections")
		if field == nil {
			return nil
		}

		collection.Fields.RemoveById(field.GetId())
		return app.Save(collection)
	})
}
