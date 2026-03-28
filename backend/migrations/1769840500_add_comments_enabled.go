package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds comments_enabled bool field to posts, projects, custom_content, and lessons.
func init() {
	m.Register(func(app core.App) error {
		contentCollections := []string{"posts", "projects", "custom_content", "lessons"}

		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			if collection.Fields.GetByName("comments_enabled") == nil {
				collection.Fields.Add(&core.BoolField{
					Name: "comments_enabled",
				})
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove comments_enabled from content collections
		contentCollections := []string{"posts", "projects", "custom_content", "lessons"}
		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}

			if f := collection.Fields.GetByName("comments_enabled"); f != nil {
				collection.Fields.RemoveById(f.GetId())
			}
			_ = app.Save(collection)
		}

		return nil
	})
}
