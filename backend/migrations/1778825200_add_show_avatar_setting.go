package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds show_avatar boolean to site_settings (default true).
// When false, public pages omit the profile avatar/photo.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		if collection.Fields.GetByName("show_avatar") != nil {
			return nil // already exists
		}

		collection.Fields.Add(&core.BoolField{
			Name: "show_avatar",
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		// Default existing records to true (show avatar)
		records, err := app.FindAllRecords("site_settings")
		if err != nil {
			return nil
		}
		for _, r := range records {
			r.Set("show_avatar", true)
			if saveErr := app.Save(r); saveErr != nil {
				return saveErr
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}
		if field := collection.Fields.GetByName("show_avatar"); field != nil {
			collection.Fields.RemoveById(field.GetId())
			return app.Save(collection)
		}
		return nil
	})
}
