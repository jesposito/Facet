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

		// Collections with file fields that need admin_tags for media tagging
		collections := []string{
			"profile",
			"experience",
			"projects",
			"education",
			"certifications",
			"posts",
			"talks",
			"views",
			"site_settings",
			"custom_content",
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
			"profile",
			"experience",
			"projects",
			"education",
			"certifications",
			"posts",
			"talks",
			"views",
			"site_settings",
			"custom_content",
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
