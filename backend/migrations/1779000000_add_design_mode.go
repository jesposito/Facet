package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds design SelectField to site_settings.
// Values: "classic" (the existing look), "soft-premium" (the warm editorial rebrand).
// Default: "classic" — existing instances render identically until the operator
// opts in. Accessibility improvements (contrast clamps) apply regardless of this
// setting; only the visual theme is gated by it.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}
		if collection.Fields.GetByName("design") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "design",
				Values:    []string{"classic", "soft-premium"},
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
			field := collection.Fields.GetByName("design")
			if field != nil {
				collection.Fields.RemoveById(field.GetId())
				app.Save(collection)
			}
		}
		return nil
	})
}
