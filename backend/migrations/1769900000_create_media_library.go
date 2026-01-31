package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Creates the unified media_library collection that replaces uploads and external_media.
// This collection uses a type discriminator to handle both uploaded files and external URLs.
func init() {
	m.Register(func(app core.App) error {
		// Check if collection already exists
		if _, err := app.FindCollectionByNameOrId("media_library"); err == nil {
			return nil
		}

		// Get admin_tags collection for relation (optional - may not exist)
		tagsCollection, _ := app.FindCollectionByNameOrId("admin_tags")

		collection := core.NewBaseCollection("media_library")

		// Type discriminator: "upload" for files, "external" for URLs
		collection.Fields.Add(&core.SelectField{
			Name:      "type",
			Required:  true,
			Values:    []string{"upload", "external"},
			MaxSelect: 1,
		})

		// File field (for type="upload")
		collection.Fields.Add(&core.FileField{
			Name:      "file",
			MaxSelect: 1,
			MaxSize:   20971520, // 20MB
		})

		// URL fields (for type="external")
		collection.Fields.Add(&core.TextField{
			Name: "url",
			Max:  2000,
		})
		collection.Fields.Add(&core.TextField{
			Name: "thumbnail_url",
			Max:  2000,
		})
		collection.Fields.Add(&core.TextField{
			Name: "provider",
			Max:  100,
		})
		collection.Fields.Add(&core.TextField{
			Name: "embed_url",
			Max:  2000,
		})

		// Common metadata fields
		collection.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      255,
		})
		collection.Fields.Add(&core.TextField{
			Name: "mime",
			Max:  255,
		})
		collection.Fields.Add(&core.TextField{
			Name: "alt_text",
			Max:  255,
		})
		collection.Fields.Add(&core.TextField{
			Name: "description",
			Max:  1000,
		})

		// Tags relation (only if admin_tags collection exists)
		if tagsCollection != nil {
			collection.Fields.Add(&core.RelationField{
				Name:          "admin_tags",
				CollectionId:  tagsCollection.Id,
				MaxSelect:     10,
				CascadeDelete: false,
			})
		}

		// Usage tracking
		collection.Fields.Add(&core.DateField{
			Name: "last_used_at",
		})

		// Access rules:
		// - Public can view external media (for public pages)
		// - Auth required for uploads and all write operations
		// Note: A hook will enforce auth for type="upload" on view
		publicRule := ""
		authRule := "@request.auth.id != ''"

		collection.ListRule = &publicRule
		collection.ViewRule = &publicRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		// Indexes for common queries
		collection.Indexes = []string{
			"CREATE INDEX idx_media_library_type ON media_library (type)",
			"CREATE INDEX idx_media_library_url ON media_library (url)",
			"CREATE INDEX idx_media_library_last_used ON media_library (last_used_at)",
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: delete the media_library collection
		collection, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
