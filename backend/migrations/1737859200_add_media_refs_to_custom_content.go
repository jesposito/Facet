package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds media_refs relation to custom_content to link external_media entries.
func init() {
	m.Register(func(app core.App) error {
		external, err := app.FindCollectionByNameOrId("external_media")
		if err != nil || external == nil {
			// If external_media doesn't exist, skip gracefully.
			return nil
		}

		collection, err := app.FindCollectionByNameOrId("custom_content")
		if err != nil || collection == nil {
			return nil
		}

		if collection.Fields.GetByName("media_refs") == nil {
			collection.Fields.Add(&core.RelationField{
				Name:         "media_refs",
				CollectionId: external.Id,
				MaxSelect:    20,
			})
			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("custom_content")
		if err != nil || collection == nil {
			return nil
		}
		collection.Fields.RemoveByName("media_refs")
		return app.Save(collection)
	})
}
