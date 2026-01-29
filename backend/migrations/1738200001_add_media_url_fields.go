package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration adds URL fields for media library selection.
// These fields store external URLs when selecting from the media library,
// as an alternative to uploading files directly.
func init() {
	m.Register(func(app core.App) error {
		// Add company_logo_library_url to experience
		if err := addURLField(app, "experience", "company_logo_library_url"); err != nil {
			return err
		}

		// Add institution_logo_library_url to education
		if err := addURLField(app, "education", "institution_logo_library_url"); err != nil {
			return err
		}

		// Add cover_image_library_url to projects
		if err := addURLField(app, "projects", "cover_image_library_url"); err != nil {
			return err
		}

		// Add cover_image_library_url to posts
		if err := addURLField(app, "posts", "cover_image_library_url"); err != nil {
			return err
		}

		// Add cover_image_library_url to talks
		if err := addURLField(app, "talks", "cover_image_library_url"); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback
		removeURLField(app, "experience", "company_logo_library_url")
		removeURLField(app, "education", "institution_logo_library_url")
		removeURLField(app, "projects", "cover_image_library_url")
		removeURLField(app, "posts", "cover_image_library_url")
		removeURLField(app, "talks", "cover_image_library_url")
		return nil
	})
}

func addURLField(app core.App, collectionName, fieldName string) error {
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

func removeURLField(app core.App, collectionName, fieldName string) {
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
