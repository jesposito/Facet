package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add media gallery field to posts and talks collections
		collections := []string{
			"posts", "demo_posts",
			"talks", "demo_talks",
		}

		for _, collectionName := range collections {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			// Check if media field already exists
			if collection.Fields.GetByName("media") != nil {
				continue // Skip if already exists
			}

			// Add media field (multi-file gallery)
			collection.Fields.Add(&core.FileField{
				Name:      "media",
				MaxSize:   10485760, // 10MB
				MaxSelect: 20,       // Up to 20 images
				MimeTypes: []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"},
			})

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Revert: This is a non-destructive migration, no revert needed
		// Media fields can remain even if migration is rolled back
		return nil
	})
}
