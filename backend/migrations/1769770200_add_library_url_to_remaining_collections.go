package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration adds *_library_url fields to collections that are missing them,
// enabling media library reuse for all image fields.
// Existing library_url fields (from migration 1738200001):
//   - experience.company_logo_library_url
//   - education.institution_logo_library_url
//   - projects.cover_image_library_url
//   - posts.cover_image_library_url
//   - talks.cover_image_library_url
//
// New fields added by this migration:
//   - profile.hero_image_library_url
//   - profile.avatar_library_url
//   - views.hero_image_library_url
//   - site_settings.favicon_library_url
//   - custom_content.cover_image_library_url
//   - testimonials.avatar_library_url (if avatar field exists)
func init() {
	m.Register(func(app core.App) error {
		// Profile: hero_image and avatar
		if err := addLibraryURLField(app, "profile", "hero_image_library_url"); err != nil {
			// Log but continue
		}
		if err := addLibraryURLField(app, "profile", "avatar_library_url"); err != nil {
			// Log but continue
		}

		// Views: hero_image
		if err := addLibraryURLField(app, "views", "hero_image_library_url"); err != nil {
			// Log but continue
		}

		// Site settings: favicon
		if err := addLibraryURLField(app, "site_settings", "favicon_library_url"); err != nil {
			// Log but continue
		}

		// Custom content: cover_image
		if err := addLibraryURLField(app, "custom_content", "cover_image_library_url"); err != nil {
			// Log but continue
		}

		// Testimonials: avatar (if avatar field exists)
		if err := addLibraryURLFieldIfSourceExists(app, "testimonials", "avatar", "avatar_library_url"); err != nil {
			// Log but continue
		}

		return nil
	}, func(app core.App) error {
		// Rollback
		removeLibraryURLField(app, "profile", "hero_image_library_url")
		removeLibraryURLField(app, "profile", "avatar_library_url")
		removeLibraryURLField(app, "views", "hero_image_library_url")
		removeLibraryURLField(app, "site_settings", "favicon_library_url")
		removeLibraryURLField(app, "custom_content", "cover_image_library_url")
		removeLibraryURLField(app, "testimonials", "avatar_library_url")
		return nil
	})
}

func addLibraryURLField(app core.App, collectionName, fieldName string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil // Collection doesn't exist, skip
	}

	if collection.Fields.GetByName(fieldName) != nil {
		return nil // Field already exists
	}

	// Use TextField instead of URLField because we store relative paths like /api/files/...
	collection.Fields.Add(&core.TextField{
		Name: fieldName,
	})

	return app.Save(collection)
}

func addLibraryURLFieldIfSourceExists(app core.App, collectionName, sourceField, fieldName string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil // Collection doesn't exist, skip
	}

	// Only add the library URL field if the source file field exists
	if collection.Fields.GetByName(sourceField) == nil {
		return nil // Source field doesn't exist, skip
	}

	if collection.Fields.GetByName(fieldName) != nil {
		return nil // Field already exists
	}

	collection.Fields.Add(&core.TextField{
		Name: fieldName,
	})

	return app.Save(collection)
}

func removeLibraryURLField(app core.App, collectionName, fieldName string) {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return
	}

	field := collection.Fields.GetByName(fieldName)
	if field != nil {
		collection.Fields.RemoveById(field.GetId())
		app.Save(collection)
	}
}
