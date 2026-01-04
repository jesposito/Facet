package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add SVG support to image fields in all collections
		collections := []string{
			"profile", "projects", "posts", "experience",
			"demo_profile", "demo_projects", "demo_posts", "demo_experience",
		}

		for _, collectionName := range collections {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			// Update avatar field (profile collections)
			if collectionName == "profile" || collectionName == "demo_profile" {
				if field := collection.Fields.GetByName("avatar"); field != nil {
					if fileField, ok := field.(*core.FileField); ok {
						fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}
					}
				}
				if field := collection.Fields.GetByName("hero_image"); field != nil {
					if fileField, ok := field.(*core.FileField); ok {
						fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif", "image/svg+xml"}
					}
				}
			}

			// Update cover_image field (projects and posts collections)
			if field := collection.Fields.GetByName("cover_image"); field != nil {
				if fileField, ok := field.(*core.FileField); ok {
					fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}
				}
			}

			// Update cover field (projects collections)
			if field := collection.Fields.GetByName("cover"); field != nil {
				if fileField, ok := field.(*core.FileField); ok {
					fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}
				}
			}

			// Update media field (projects and experience collections - multi-file galleries)
			if field := collection.Fields.GetByName("media"); field != nil {
				if fileField, ok := field.(*core.FileField); ok {
					fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}
				}
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Revert: Remove SVG support
		collections := []string{
			"profile", "projects", "posts", "experience",
			"demo_profile", "demo_projects", "demo_posts", "demo_experience",
		}

		for _, collectionName := range collections {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue
			}

			// Revert avatar field
			if collectionName == "profile" || collectionName == "demo_profile" {
				if field := collection.Fields.GetByName("avatar"); field != nil {
					if fileField, ok := field.(*core.FileField); ok {
						fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp"}
					}
				}
				if field := collection.Fields.GetByName("hero_image"); field != nil {
					if fileField, ok := field.(*core.FileField); ok {
						fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
					}
				}
			}

			// Revert cover_image field
			if field := collection.Fields.GetByName("cover_image"); field != nil {
				if fileField, ok := field.(*core.FileField); ok {
					fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp"}
				}
			}

			// Revert cover field
			if field := collection.Fields.GetByName("cover"); field != nil {
				if fileField, ok := field.(*core.FileField); ok {
					fileField.MimeTypes = []string{"image/jpeg", "image/png", "image/webp"}
				}
			}

			// Revert media field (remove MIME type restrictions to accept all files)
			if field := collection.Fields.GetByName("media"); field != nil {
				if fileField, ok := field.(*core.FileField); ok {
					fileField.MimeTypes = []string{}
				}
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
