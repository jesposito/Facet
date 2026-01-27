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

		// Add default_locale field if it doesn't exist
		if collection.Fields.GetByName("default_locale") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "default_locale",
				Values:    []string{"en", "de", "elvish", "klingon", "lolcat"},
				MaxSelect: 1,
			})
			if err := app.Save(collection); err != nil {
				return err
			}

			// Update existing record with default value
			records, err := app.FindAllRecords("site_settings")
			if err == nil && len(records) > 0 {
				records[0].Set("default_locale", "en")
				return app.Save(records[0])
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("default_locale")
		if field != nil {
			collection.Fields.RemoveByName("default_locale")
			return app.Save(collection)
		}

		return nil
	})
}
