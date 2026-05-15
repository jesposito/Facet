package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Lets the admin choose whether the site nav (bar or chips style) sits
		// above the hero header or below it. Default below keeps backward compat.
		settingsCollection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}
		if settingsCollection.Fields.GetByName("site_nav_position") != nil {
			return nil
		}
		settingsCollection.Fields.Add(&core.SelectField{
			Name:      "site_nav_position",
			Values:    []string{"below", "above"},
			MaxSelect: 1,
		})
		return app.Save(settingsCollection)
	}, func(app core.App) error {
		if settingsCollection, err := app.FindCollectionByNameOrId("site_settings"); err == nil {
			if f := settingsCollection.Fields.GetByName("site_nav_position"); f != nil {
				settingsCollection.Fields.RemoveById(f.GetId())
				app.Save(settingsCollection)
			}
		}
		return nil
	})
}
