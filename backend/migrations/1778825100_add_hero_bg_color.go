package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// hero_bg_color is a hex string applied to the hero header background
		// when no hero image is set. Empty means use the default gradient.
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}
		if profileCollection.Fields.GetByName("hero_bg_color") == nil {
			profileCollection.Fields.Add(&core.TextField{
				Name:    "hero_bg_color",
				Max:     7,
				Pattern: `^#[0-9a-fA-F]{6}$`,
			})
			if err := app.Save(profileCollection); err != nil {
				return err
			}
		}

		// Also allow per-view override
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			if viewsCollection.Fields.GetByName("hero_bg_color") == nil {
				viewsCollection.Fields.Add(&core.TextField{
					Name:    "hero_bg_color",
					Max:     7,
					Pattern: `^#[0-9a-fA-F]{6}$`,
				})
				if err := app.Save(viewsCollection); err != nil {
					return err
				}
			}
		}

		return nil
	}, func(app core.App) error {
		if profileCollection, err := app.FindCollectionByNameOrId("profile"); err == nil {
			if f := profileCollection.Fields.GetByName("hero_bg_color"); f != nil {
				profileCollection.Fields.RemoveById(f.GetId())
				app.Save(profileCollection)
			}
		}
		if viewsCollection, err := app.FindCollectionByNameOrId("views"); err == nil {
			if f := viewsCollection.Fields.GetByName("hero_bg_color"); f != nil {
				viewsCollection.Fields.RemoveById(f.GetId())
				app.Save(viewsCollection)
			}
		}
		return nil
	})
}
