package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add hero_spacing to profile (homepage default) — empty = default bucket
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err == nil {
			if profileCollection.Fields.GetByName("hero_spacing") == nil {
				profileCollection.Fields.Add(&core.SelectField{
					Name:      "hero_spacing",
					Values:    []string{"compact", "default", "spacious"},
					MaxSelect: 1,
				})
				if err := app.Save(profileCollection); err != nil {
					return err
				}
			}
		}

		// Add hero_spacing to views (per-facet override — empty = inherit from profile)
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			if viewsCollection.Fields.GetByName("hero_spacing") == nil {
				viewsCollection.Fields.Add(&core.SelectField{
					Name:      "hero_spacing",
					Values:    []string{"compact", "default", "spacious"},
					MaxSelect: 1,
				})
				if err := app.Save(viewsCollection); err != nil {
					return err
				}
			}
		}

		return nil
	}, func(app core.App) error {
		if profileCollection, err := app.FindCollectionByNameOrId("profile"); err == nil {
			if f := profileCollection.Fields.GetByName("hero_spacing"); f != nil {
				profileCollection.Fields.RemoveById(f.GetId())
				app.Save(profileCollection)
			}
		}
		if viewsCollection, err := app.FindCollectionByNameOrId("views"); err == nil {
			if f := viewsCollection.Fields.GetByName("hero_spacing"); f != nil {
				viewsCollection.Fields.RemoveById(f.GetId())
				app.Save(viewsCollection)
			}
		}
		return nil
	})
}
