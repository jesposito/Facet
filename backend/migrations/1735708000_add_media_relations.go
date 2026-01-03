package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds media_refs relation to projects/posts/talks to link external_media entries.
func init() {
	m.Register(func(app core.App) error {
		external, err := app.FindCollectionByNameOrId("external_media")
		if err != nil || external == nil {
			// If external_media doesn't exist (stale data), skip adding relations gracefully.
			return nil
		}

		addRelation := func(collectionName string) error {
			collection, err := app.FindCollectionByNameOrId(collectionName)
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
		}

		if err := addRelation("projects"); err != nil {
			return err
		}
		if err := addRelation("posts"); err != nil {
			return err
		}
		if err := addRelation("talks"); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		removeRelation := func(collectionName string) error {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil || collection == nil {
				return nil
			}
			collection.Fields.RemoveByName("media_refs")
			return app.Save(collection)
		}

		_ = removeRelation("projects")
		_ = removeRelation("posts")
		_ = removeRelation("talks")
		return nil
	})
}
