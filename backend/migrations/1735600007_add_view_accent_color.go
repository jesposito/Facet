package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find the views collection
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			// Collection doesn't exist, nothing to do
			return nil
		}

		// Check if accent_color field already exists
		if viewsCollection.Fields.GetByName("accent_color") != nil {
			return nil
		}

		// Add accent_color field with curated palette values
		// null/empty = inherit from global profile setting
		viewsCollection.Fields.Add(&core.SelectField{
			Name:      "accent_color",
			Values:    []string{"sky", "indigo", "emerald", "rose", "amber", "slate"},
			MaxSelect: 1,
		})

		return app.Save(viewsCollection)
	}, func(app core.App) error {
		// Rollback: remove accent_color field from views collection
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}

		accentField := viewsCollection.Fields.GetByName("accent_color")
		if accentField != nil {
			viewsCollection.Fields.RemoveById(accentField.GetId())
		}

		return app.Save(viewsCollection)
	})
}
