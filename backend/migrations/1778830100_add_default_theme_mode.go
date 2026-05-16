package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds default_theme_mode SelectField to site_settings.
// Values: "system" (follow user's prefers-color-scheme), "light", "dark".
// Default: "system" — preserves existing behavior of respecting the user's OS preference.
// Per-user override via ThemeToggle (localStorage) always wins over this default.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}
		if collection.Fields.GetByName("default_theme_mode") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "default_theme_mode",
				Values:    []string{"system", "light", "dark"},
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
			field := collection.Fields.GetByName("default_theme_mode")
			if field != nil {
				collection.Fields.RemoveById(field.GetId())
				app.Save(collection)
			}
		}
		return nil
	})
}
