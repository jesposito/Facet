package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add font_pack to profile collection
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}
		if profileCollection.Fields.GetByName("font_pack") == nil {
			profileCollection.Fields.Add(&core.SelectField{
				Name:      "font_pack",
				Values:    []string{"editorial", "modern", "classic", "technical", "editorial-bold", "humanist", "geometric"},
				MaxSelect: 1,
			})
			if err := app.Save(profileCollection); err != nil {
				return err
			}
		}

		// Add font_pack to views collection (empty = inherit from profile)
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}
		if viewsCollection.Fields.GetByName("font_pack") == nil {
			viewsCollection.Fields.Add(&core.SelectField{
				Name:      "font_pack",
				Values:    []string{"editorial", "modern", "classic", "technical", "editorial-bold", "humanist", "geometric"},
				MaxSelect: 1,
			})
			if err := app.Save(viewsCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove font_pack from profile collection
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err == nil {
			fontPackField := profileCollection.Fields.GetByName("font_pack")
			if fontPackField != nil {
				profileCollection.Fields.RemoveById(fontPackField.GetId())
				app.Save(profileCollection)
			}
		}

		// Rollback: remove font_pack from views collection
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			fontPackField := viewsCollection.Fields.GetByName("font_pack")
			if fontPackField != nil {
				viewsCollection.Fields.RemoveById(fontPackField.GetId())
				app.Save(viewsCollection)
			}
		}

		return nil
	})
}
