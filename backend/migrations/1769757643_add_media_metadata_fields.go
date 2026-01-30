package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add alt_text and description to uploads collection
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err != nil {
			return err
		}

		if uploadsCollection.Fields.GetByName("alt_text") == nil {
			uploadsCollection.Fields.Add(&core.TextField{
				Name:     "alt_text",
				Max:      255,
			})
		}

		if uploadsCollection.Fields.GetByName("description") == nil {
			uploadsCollection.Fields.Add(&core.TextField{
				Name:     "description",
				Max:      1000,
			})
		}

		if err := app.Save(uploadsCollection); err != nil {
			return err
		}

		// Add alt_text and description to external_media collection
		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// external_media may not exist yet - not an error
			return nil
		}

		if externalCollection.Fields.GetByName("alt_text") == nil {
			externalCollection.Fields.Add(&core.TextField{
				Name:     "alt_text",
				Max:      255,
			})
		}

		if externalCollection.Fields.GetByName("description") == nil {
			externalCollection.Fields.Add(&core.TextField{
				Name:     "description",
				Max:      1000,
			})
		}

		return app.Save(externalCollection)
	}, func(app core.App) error {
		// Rollback: remove alt_text and description fields
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err == nil {
			if field := uploadsCollection.Fields.GetByName("alt_text"); field != nil {
				uploadsCollection.Fields.RemoveById(field.GetId())
			}
			if field := uploadsCollection.Fields.GetByName("description"); field != nil {
				uploadsCollection.Fields.RemoveById(field.GetId())
			}
			app.Save(uploadsCollection)
		}

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err == nil {
			if field := externalCollection.Fields.GetByName("alt_text"); field != nil {
				externalCollection.Fields.RemoveById(field.GetId())
			}
			if field := externalCollection.Fields.GetByName("description"); field != nil {
				externalCollection.Fields.RemoveById(field.GetId())
			}
			app.Save(externalCollection)
		}

		return nil
	})
}
