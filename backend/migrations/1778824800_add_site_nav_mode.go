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
		if collection.Fields.GetByName("site_nav_mode") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "site_nav_mode",
				Values:    []string{"bar", "chips"},
				MaxSelect: 1,
			})
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err == nil {
			field := collection.Fields.GetByName("site_nav_mode")
			if field != nil {
				collection.Fields.RemoveById(field.GetId())
				app.Save(collection)
			}
		}
		return nil
	})
}
