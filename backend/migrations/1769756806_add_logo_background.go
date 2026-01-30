package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add logo_background to experience collection
		expCollection, err := app.FindCollectionByNameOrId("experience")
		if err != nil {
			return err
		}

		if expCollection.Fields.GetByName("logo_background") == nil {
			expCollection.Fields.Add(&core.SelectField{
				Name:     "logo_background",
				MaxSelect: 1,
				Values: []string{
					"none",       // Transparent (default)
					"white",      // #ffffff
					"light-gray", // #f3f4f6 (bg-gray-100)
					"dark-gray",  // #374151 (bg-gray-700)
					"black",      // #000000
					"accent",     // Profile's theme color
				},
			})

			if err := app.Save(expCollection); err != nil {
				return err
			}
		}

		// Add logo_background to education collection
		eduCollection, err := app.FindCollectionByNameOrId("education")
		if err != nil {
			return err
		}

		if eduCollection.Fields.GetByName("logo_background") == nil {
			eduCollection.Fields.Add(&core.SelectField{
				Name:     "logo_background",
				MaxSelect: 1,
				Values: []string{
					"none",       // Transparent (default)
					"white",      // #ffffff
					"light-gray", // #f3f4f6 (bg-gray-100)
					"dark-gray",  // #374151 (bg-gray-700)
					"black",      // #000000
					"accent",     // Profile's theme color
				},
			})

			if err := app.Save(eduCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove logo_background fields
		expCollection, err := app.FindCollectionByNameOrId("experience")
		if err == nil {
			field := expCollection.Fields.GetByName("logo_background")
			if field != nil {
				expCollection.Fields.RemoveById(field.GetId())
				app.Save(expCollection)
			}
		}

		eduCollection, err := app.FindCollectionByNameOrId("education")
		if err == nil {
			field := eduCollection.Fields.GetByName("logo_background")
			if field != nil {
				eduCollection.Fields.RemoveById(field.GetId())
				app.Save(eduCollection)
			}
		}

		return nil
	})
}
