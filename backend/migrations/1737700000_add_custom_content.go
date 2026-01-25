package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Creates the custom_content collection for user-defined content sections.
// Custom content items can be added to facets/views just like other content types.
func init() {
	m.Register(func(app core.App) error {
		// Check if collection already exists
		if _, err := app.FindCollectionByNameOrId("custom_content"); err == nil {
			return nil
		}

		collection := core.NewBaseCollection("custom_content")

		// Public read access (filtered by visibility in application logic)
		publicRule := ""
		authRule := "@request.auth.id != ''"

		collection.ListRule = &publicRule
		collection.ViewRule = &publicRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		// Title/header for the custom content section
		collection.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      500,
		})

		// Rich text content (markdown/HTML)
		collection.Fields.Add(&core.EditorField{
			Name: "content",
		})

		// Cover image for the section
		collection.Fields.Add(&core.FileField{
			Name:      "cover_image",
			MaxSize:   10485760, // 10MB
			MaxSelect: 1,
			MimeTypes: []string{
				"image/jpeg",
				"image/png",
				"image/webp",
			},
		})

		// Media gallery (multiple images)
		collection.Fields.Add(&core.FileField{
			Name:      "media",
			MaxSize:   10485760, // 10MB per file
			MaxSelect: 20,       // Up to 20 images
			MimeTypes: []string{
				"image/jpeg",
				"image/png",
				"image/webp",
				"image/svg+xml",
			},
		})

		// Visibility control
		collection.Fields.Add(&core.SelectField{
			Name:      "visibility",
			MaxSelect: 1,
			Values:    []string{"public", "unlisted", "private"},
		})

		// Draft status
		collection.Fields.Add(&core.BoolField{
			Name: "is_draft",
		})

		// Sort order for display
		collection.Fields.Add(&core.NumberField{
			Name: "sort_order",
		})

		// Per-view visibility JSON (same pattern as other content types)
		collection.Fields.Add(&core.JSONField{
			Name:     "view_visibility",
			Required: false,
		})

		// Save collection first to get its ID for admin_tags relation
		if err := app.Save(collection); err != nil {
			return err
		}

		// Add admin_tags relation (if admin_tags collection exists)
		tagsCollection, err := app.FindCollectionByNameOrId("admin_tags")
		if err == nil {
			// Re-fetch the collection to add the relation
			collection, err = app.FindCollectionByNameOrId("custom_content")
			if err != nil {
				return err
			}

			collection.Fields.Add(&core.RelationField{
				Name:          "admin_tags",
				CollectionId:  tagsCollection.Id,
				MaxSelect:     10,
				CascadeDelete: false,
			})

			return app.Save(collection)
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete the collection
		collection, err := app.FindCollectionByNameOrId("custom_content")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
