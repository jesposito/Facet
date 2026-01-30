package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add last_used_at to uploads collection
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err != nil {
			return err
		}

		if uploadsCollection.Fields.GetByName("last_used_at") == nil {
			uploadsCollection.Fields.Add(&core.DateField{
				Name: "last_used_at",
			})
			if err := app.Save(uploadsCollection); err != nil {
				return err
			}
		}

		// Add last_used_at to external_media collection
		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// external_media may not exist yet - not an error
			return nil
		}

		if externalCollection.Fields.GetByName("last_used_at") == nil {
			externalCollection.Fields.Add(&core.DateField{
				Name: "last_used_at",
			})
			if err := app.Save(externalCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove last_used_at field from media collections
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err == nil {
			if field := uploadsCollection.Fields.GetByName("last_used_at"); field != nil {
				uploadsCollection.Fields.RemoveById(field.GetId())
				app.Save(uploadsCollection)
			}
		}

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err == nil {
			if field := externalCollection.Fields.GetByName("last_used_at"); field != nil {
				externalCollection.Fields.RemoveById(field.GetId())
				app.Save(externalCollection)
			}
		}

		return nil
	})
}
