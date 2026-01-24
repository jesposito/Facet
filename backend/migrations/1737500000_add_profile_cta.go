package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		if profileCollection.Fields.GetByName("cta_text") != nil {
			return nil
		}

		profileCollection.Fields.Add(&core.TextField{
			Name:     "cta_text",
			Required: false,
		})

		profileCollection.Fields.Add(&core.URLField{
			Name:     "cta_url",
			Required: false,
		})

		return app.Save(profileCollection)
	}, func(app core.App) error {
		profileCollection, err := app.FindCollectionByNameOrId("profile")
		if err != nil {
			return nil
		}

		ctaTextField := profileCollection.Fields.GetByName("cta_text")
		if ctaTextField != nil {
			profileCollection.Fields.RemoveById(ctaTextField.GetId())
		}

		ctaUrlField := profileCollection.Fields.GetByName("cta_url")
		if ctaUrlField != nil {
			profileCollection.Fields.RemoveById(ctaUrlField.GetId())
		}

		return app.Save(profileCollection)
	})
}
