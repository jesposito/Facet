package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		if profileCollection.Fields.GetByName("custom_hex_color") == nil {
			profileCollection.Fields.Add(&core.TextField{
				Name:    "custom_hex_color",
				Max:     7, // #RRGGBB
				Pattern: `^#[0-9a-fA-F]{6}$`,
			})

			if err := app.Save(profileCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		if f := profileCollection.Fields.GetByName("custom_hex_color"); f != nil {
			profileCollection.Fields.RemoveById(f.GetId())
			return app.Save(profileCollection)
		}

		return nil
	})
}
