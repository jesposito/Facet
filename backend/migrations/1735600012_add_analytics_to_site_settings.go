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

		if collection.Fields.GetByName("ga_measurement_id") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "ga_measurement_id",
				Max:  100,
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("ga_measurement_id")
		if field != nil {
			collection.Fields.RemoveById(field.GetId())
		}

		return app.Save(collection)
	})
}
