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

		collection.Fields.Add(&core.BoolField{
			Name: "hide_login_button",
		})

		collection.Fields.Add(&core.BoolField{
			Name: "hide_demo_toggle",
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("hide_login_button")
		collection.Fields.RemoveByName("hide_demo_toggle")
		return app.Save(collection)
	})
}
