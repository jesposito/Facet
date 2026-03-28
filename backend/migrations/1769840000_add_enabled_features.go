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

		if collection.Fields.GetByName("enabled_features") == nil {
			collection.Fields.Add(&core.JSONField{
				Name:    "enabled_features",
				MaxSize: 5000,
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}
		if f := collection.Fields.GetByName("enabled_features"); f != nil {
			collection.Fields.RemoveById(f.GetId())
			return app.Save(collection)
		}
		return nil
	})
}
