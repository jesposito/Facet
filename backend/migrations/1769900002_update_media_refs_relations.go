package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Updates media_refs relation fields to point to media_library instead of external_media.
// Since we preserved original IDs during migration, the actual relation values don't change.
func init() {
	m.Register(func(app core.App) error {
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return err
		}

		// Collections that have media_refs fields
		collectionsWithMediaRefs := []string{
			"posts",
			"projects",
			"talks",
			"custom_content",
		}

		for _, collName := range collectionsWithMediaRefs {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				// Collection doesn't exist - skip
				continue
			}

			field := collection.Fields.GetByName("media_refs")
			if field == nil {
				continue
			}

			// Get the existing relation field
			relationField, ok := field.(*core.RelationField)
			if !ok {
				continue
			}

			// Update to point to media_library
			relationField.CollectionId = mediaLibrary.Id

			if err := app.Save(collection); err != nil {
				app.Logger().Error("Failed to update media_refs relation", "collection", collName, "error", err)
				continue
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: point media_refs back to external_media
		externalMedia, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// external_media doesn't exist, can't rollback
			return nil
		}

		collectionsWithMediaRefs := []string{
			"posts",
			"projects",
			"talks",
			"custom_content",
		}

		for _, collName := range collectionsWithMediaRefs {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			field := collection.Fields.GetByName("media_refs")
			if field == nil {
				continue
			}

			relationField, ok := field.(*core.RelationField)
			if !ok {
				continue
			}

			relationField.CollectionId = externalMedia.Id
			app.Save(collection)
		}

		return nil
	})
}
