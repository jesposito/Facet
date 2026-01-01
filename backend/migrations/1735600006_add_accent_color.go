package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find the profile collection
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			// Collection doesn't exist, nothing to do
			return nil
		}

		// Check if accent_color field already exists
		if profileCollection.Fields.GetByName("accent_color") != nil {
			return nil
		}

		// Add accent_color field with curated palette values
		// Default to 'sky' which is the current primary color
		profileCollection.Fields.Add(&core.SelectField{
			Name:      "accent_color",
			Values:    []string{"sky", "indigo", "emerald", "rose", "amber", "slate"},
			MaxSelect: 1,
		})

		return app.Save(profileCollection)
	}, func(app core.App) error {
		// Rollback: remove accent_color field from profile collection
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		accentField := profileCollection.Fields.GetByName("accent_color")
		if accentField != nil {
			profileCollection.Fields.RemoveById(accentField.GetId())
		}

		return app.Save(profileCollection)
	})
}
