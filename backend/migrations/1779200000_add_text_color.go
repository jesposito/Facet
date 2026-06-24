package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds a nullable text_color TextField to the profile collection.
//
// text_color holds the operator's chosen body/heading text hue as #RRGGBB (empty
// = default neutral ink). The frontend derives a per-mode, hue-preserved, AAA
// (7:1) clamped ink from this hue (deriveTextInk in colors.ts) and injects it as
// --text-ink / --text-ink-dark, so the operator's hue is honored while contrast
// can never fall below AAA. Empty ⇒ nothing injected ⇒ render unchanged, so
// existing instances are byte-identical until an operator opts in.
func init() {
	m.Register(func(app core.App) error {
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			// Collection doesn't exist, nothing to do.
			return nil
		}

		if profileCollection.Fields.GetByName("text_color") == nil {
			profileCollection.Fields.Add(&core.TextField{
				Name:    "text_color",
				Max:     7, // #RRGGBB
				Pattern: `^#[0-9a-fA-F]{6}$`,
			})

			if err := app.Save(profileCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		if f := profileCollection.Fields.GetByName("text_color"); f != nil {
			profileCollection.Fields.RemoveById(f.GetId())
			return app.Save(profileCollection)
		}

		return nil
	})
}
