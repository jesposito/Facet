package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add hero_layout to profile collection
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}
		if profileCollection.Fields.GetByName("hero_layout") == nil {
			profileCollection.Fields.Add(&core.SelectField{
				Name:      "hero_layout",
				Values:    []string{"standard", "centered", "split", "minimal", "stacked"},
				MaxSelect: 1,
			})
			if err := app.Save(profileCollection); err != nil {
				return err
			}
		}

		// Add hero_layout to views collection (empty = inherit from profile)
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}
		if viewsCollection.Fields.GetByName("hero_layout") == nil {
			viewsCollection.Fields.Add(&core.SelectField{
				Name:      "hero_layout",
				Values:    []string{"standard", "centered", "split", "minimal", "stacked"},
				MaxSelect: 1,
			})
			if err := app.Save(viewsCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove hero_layout from profile collection
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err == nil {
			heroLayoutField := profileCollection.Fields.GetByName("hero_layout")
			if heroLayoutField != nil {
				profileCollection.Fields.RemoveById(heroLayoutField.GetId())
				app.Save(profileCollection)
			}
		}

		// Rollback: remove hero_layout from views collection
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			heroLayoutField := viewsCollection.Fields.GetByName("hero_layout")
			if heroLayoutField != nil {
				viewsCollection.Fields.RemoveById(heroLayoutField.GetId())
				app.Save(viewsCollection)
			}
		}

		return nil
	})
}
