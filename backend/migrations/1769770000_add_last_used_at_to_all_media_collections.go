package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration adds the last_used_at field to all collections with file fields,
// enabling usage tracking across the entire media system (not just uploads and external_media).
func init() {
	m.Register(func(app core.App) error {
		// Collections with file fields that need last_used_at for usage tracking
		// Note: uploads and external_media already have this field from migration 1769758100
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
			"testimonials",
			"media_library",
			"view_exports",
			"resume_imports",
		}

		for _, collName := range collections {
			if err := addLastUsedAtField(app, collName); err != nil {
				// Log but continue with other collections
				continue
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove last_used_at field from collections
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
			"testimonials",
			"media_library",
			"view_exports",
			"resume_imports",
		}

		for _, collName := range collections {
			removeLastUsedAtField(app, collName)
		}

		return nil
	})
}

func addLastUsedAtField(app core.App, collectionName string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil // Collection doesn't exist, skip
	}

	if collection.Fields.GetByName("last_used_at") != nil {
		return nil // Field already exists
	}

	collection.Fields.Add(&core.DateField{
		Name: "last_used_at",
	})

	return app.Save(collection)
}

func removeLastUsedAtField(app core.App, collectionName string) {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return
	}

	field := collection.Fields.GetByName("last_used_at")
	if field != nil {
		collection.Fields.RemoveById(field.GetId())
		app.Save(collection)
	}
}
