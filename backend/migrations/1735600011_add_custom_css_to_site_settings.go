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

		collection.Fields.Add(&core.TextField{
			Name: "custom_css",
			Max:  20000,
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		// Ensure existing record has the field set
		records, err := app.FindRecordsByFilter(collection.Name, "", "", 1, 0, nil)
		if err == nil && len(records) > 0 {
			record := records[0]
			if record.GetString("custom_css") == "" {
				record.Set("custom_css", "")
				app.Save(record)
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("custom_css")
		if field != nil {
			collection.Fields.RemoveById(field.GetId())
			return app.Save(collection)
		}

		return nil
	})
}
