package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create admin_tags collection
		if _, err := app.FindCollectionByNameOrId("admin_tags"); err != nil {
			tagsCollection := core.NewBaseCollection("admin_tags")
			// No public access - admin only
			tagsCollection.Fields.Add(&core.TextField{
				Name:     "name",
				Required: true,
				Max:      50,
			})
			tagsCollection.Fields.Add(&core.TextField{
				Name: "color",
				Max:  20,
			})
			tagsCollection.Fields.Add(&core.NumberField{
				Name: "sort_order",
			})

			tagsCollection.Indexes = append(tagsCollection.Indexes,
				"CREATE UNIQUE INDEX idx_admin_tags_name ON admin_tags(name)")

			if err := app.Save(tagsCollection); err != nil {
				return err
			}
		}

		// Get admin_tags collection for relation
		tagsCollection, err := app.FindCollectionByNameOrId("admin_tags")
		if err != nil {
			return err
		}

		// List of content collections that should have admin_tags
		contentCollections := []string{
			"experience",
			"projects",
			"education",
			"certifications",
			"awards",
			"skills",
			"posts",
			"talks",
			"testimonials",
		}

		// Add admin_tags relation field to each content collection
		for _, collName := range contentCollections {
			coll, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				// Collection doesn't exist, skip
				continue
			}

			// Check if field already exists
			if coll.Fields.GetByName("admin_tags") != nil {
				continue
			}

			// Add relation field (multi-select)
			coll.Fields.Add(&core.RelationField{
				Name:          "admin_tags",
				CollectionId:  tagsCollection.Id,
				MaxSelect:     10, // Allow up to 10 tags per item
				CascadeDelete: false,
			})

			if err := app.Save(coll); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove admin_tags field from content collections
		contentCollections := []string{
			"experience",
			"projects",
			"education",
			"certifications",
			"awards",
			"skills",
			"posts",
			"talks",
			"testimonials",
		}

		for _, collName := range contentCollections {
			coll, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}
			coll.Fields.RemoveByName("admin_tags")
			app.Save(coll)
		}

		// Delete admin_tags collection
		tagsCollection, err := app.FindCollectionByNameOrId("admin_tags")
		if err == nil {
			app.Delete(tagsCollection)
		}

		return nil
	})
}
