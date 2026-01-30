package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Extends media_display_names to store alt_text, description, and admin_tags
// for media files from any collection (not just uploads/external_media).
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("media_display_names")
		if err != nil {
			return nil // Collection doesn't exist yet
		}

		// Get admin_tags collection for relation
		tagsCollection, err := app.FindCollectionByNameOrId("admin_tags")
		if err != nil {
			return nil // admin_tags doesn't exist yet
		}

		modified := false

		// Add alt_text field
		if collection.Fields.GetByName("alt_text") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "alt_text",
				Max:  255,
			})
			modified = true
		}

		// Add description field
		if collection.Fields.GetByName("description") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "description",
				Max:  1000,
			})
			modified = true
		}

		// Add admin_tags relation
		if collection.Fields.GetByName("admin_tags") == nil {
			collection.Fields.Add(&core.RelationField{
				Name:          "admin_tags",
				CollectionId:  tagsCollection.Id,
				MaxSelect:     10,
				CascadeDelete: false,
			})
			modified = true
		}

		// Make display_name optional (was required before)
		if field := collection.Fields.GetByName("display_name"); field != nil {
			if tf, ok := field.(*core.TextField); ok && tf.Required {
				tf.Required = false
				modified = true
			}
		}

		if modified {
			return app.Save(collection)
		}
		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("media_display_names")
		if err != nil {
			return nil
		}

		// Remove added fields
		if field := collection.Fields.GetByName("alt_text"); field != nil {
			collection.Fields.RemoveById(field.GetId())
		}
		if field := collection.Fields.GetByName("description"); field != nil {
			collection.Fields.RemoveById(field.GetId())
		}
		if field := collection.Fields.GetByName("admin_tags"); field != nil {
			collection.Fields.RemoveById(field.GetId())
		}

		return app.Save(collection)
	})
}
