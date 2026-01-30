package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get admin_tags collection for relation
		tagsCollection, err := app.FindCollectionByNameOrId("admin_tags")
		if err != nil {
			// admin_tags doesn't exist yet - not an error, will be added later
			return nil
		}

		// Add admin_tags to uploads collection
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err != nil {
			return err
		}

		if uploadsCollection.Fields.GetByName("admin_tags") == nil {
			uploadsCollection.Fields.Add(&core.RelationField{
				Name:          "admin_tags",
				CollectionId:  tagsCollection.Id,
				MaxSelect:     10,
				CascadeDelete: false,
			})
			if err := app.Save(uploadsCollection); err != nil {
				return err
			}
		}

		// Add admin_tags to external_media collection
		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// external_media may not exist yet - not an error
			return nil
		}

		if externalCollection.Fields.GetByName("admin_tags") == nil {
			externalCollection.Fields.Add(&core.RelationField{
				Name:          "admin_tags",
				CollectionId:  tagsCollection.Id,
				MaxSelect:     10,
				CascadeDelete: false,
			})
			if err := app.Save(externalCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove admin_tags field from media collections
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err == nil {
			if field := uploadsCollection.Fields.GetByName("admin_tags"); field != nil {
				uploadsCollection.Fields.RemoveById(field.GetId())
				app.Save(uploadsCollection)
			}
		}

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err == nil {
			if field := externalCollection.Fields.GetByName("admin_tags"); field != nil {
				externalCollection.Fields.RemoveById(field.GetId())
				app.Save(externalCollection)
			}
		}

		return nil
	})
}
