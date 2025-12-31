package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Helper for public read access
		publicRule := ""

		// Check if posts collection exists
		_, err := app.FindCollectionByNameOrId("posts")
		if err == nil {
			// Collection already exists, nothing to do
			return nil
		}

		// Create Posts collection
		postsCollection := core.NewBaseCollection("posts")
		postsCollection.ListRule = &publicRule
		postsCollection.ViewRule = &publicRule
		postsCollection.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 500})
		postsCollection.Fields.Add(&core.TextField{Name: "slug", Max: 200})
		postsCollection.Fields.Add(&core.TextField{Name: "excerpt", Max: 1000})
		postsCollection.Fields.Add(&core.EditorField{Name: "content"})
		postsCollection.Fields.Add(&core.FileField{Name: "cover_image", MaxSize: 10485760, MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}})
		postsCollection.Fields.Add(&core.JSONField{Name: "tags"})
		postsCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		postsCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		postsCollection.Fields.Add(&core.DateField{Name: "published_at"})

		postsCollection.Indexes = append(postsCollection.Indexes, "CREATE UNIQUE INDEX idx_posts_slug ON posts(slug) WHERE slug != ''")

		return app.Save(postsCollection)
	}, func(app core.App) error {
		// Rollback: delete posts collection if it was created by this migration
		collection, err := app.FindCollectionByNameOrId("posts")
		if err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
