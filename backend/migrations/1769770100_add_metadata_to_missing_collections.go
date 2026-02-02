package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration adds admin_tags relation to collections that are missing it,
// enabling media tagging for all file-containing collections.
// Missing collections identified in audit: testimonials, media_library, view_exports, resume_imports
func init() {
	m.Register(func(app core.App) error {
		// Get admin_tags collection for relation
		tagsCollection, err := app.FindCollectionByNameOrId("admin_tags")
		if err != nil {
			// admin_tags doesn't exist yet - not an error, will be added later
			return nil
		}

		// Collections with file fields that are missing admin_tags
		collections := []string{
			"testimonials",
			"media_library",
			"view_exports",
			"resume_imports",
		}

		for _, collName := range collections {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				// Collection doesn't exist - skip
				continue
			}

			if collection.Fields.GetByName("admin_tags") == nil {
				collection.Fields.Add(&core.RelationField{
					Name:          "admin_tags",
					CollectionId:  tagsCollection.Id,
					MaxSelect:     10,
					CascadeDelete: false,
				})
				if err := app.Save(collection); err != nil {
					// Log but continue with other collections
					continue
				}
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove admin_tags field from collections
		collections := []string{
			"testimonials",
			"media_library",
			"view_exports",
			"resume_imports",
		}

		for _, collName := range collections {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			if field := collection.Fields.GetByName("admin_tags"); field != nil {
				collection.Fields.RemoveById(field.GetId())
				app.Save(collection)
			}
		}

		return nil
	})
}
