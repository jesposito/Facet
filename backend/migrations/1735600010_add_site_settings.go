package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("site_settings")

		collection.Fields.Add(&core.BoolField{
			Name: "homepage_enabled",
		})

		collection.Fields.Add(&core.TextField{
			Name: "landing_page_message",
			Max:  2000,
		})

		authRule := "@request.auth.id != ''"
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		if err := app.Save(collection); err != nil {
			return err
		}

		// Seed a default settings record
		record := models.NewRecord(collection)
		record.Set("homepage_enabled", true)
		record.Set("landing_page_message", "This profile is being set up.")

		if err := app.Save(record); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		settingsCollection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}
		return app.Delete(settingsCollection)
	})
}
