package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Reverts media_refs to point back to external_media.
// This is a temporary fix while we debug the media_library migration.
func init() {
	m.Register(func(app core.App) error {
		externalMedia, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			app.Logger().Error("external_media collection not found", "error", err)
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

			// Revert to external_media
			relationField.CollectionId = externalMedia.Id

			if err := app.Save(collection); err != nil {
				app.Logger().Error("Failed to revert media_refs relation", "collection", collName, "error", err)
				continue
			}

			app.Logger().Info("Reverted media_refs to external_media", "collection", collName)
		}

		return nil
	}, func(app core.App) error {
		// No-op rollback
		return nil
	})
}
