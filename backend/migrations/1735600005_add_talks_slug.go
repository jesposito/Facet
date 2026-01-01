package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find the talks collection
		talksCollection, err := app.FindCollectionByNameOrId("talks")
		if err != nil {
			// Collection doesn't exist, nothing to do
			return nil
		}

		// Check if slug field already exists
		if talksCollection.Fields.GetByName("slug") != nil {
			return nil
		}

		// Add slug field
		talksCollection.Fields.Add(&core.TextField{Name: "slug", Max: 200})

		// Add unique index on slug (allowing empty slugs)
		talksCollection.Indexes = append(talksCollection.Indexes, "CREATE UNIQUE INDEX idx_talks_slug ON talks(slug) WHERE slug != ''")

		return app.Save(talksCollection)
	}, func(app core.App) error {
		// Rollback: remove slug field from talks collection
		talksCollection, err := app.FindCollectionByNameOrId("talks")
		if err != nil {
			return nil
		}

		slugField := talksCollection.Fields.GetByName("slug")
		if slugField != nil {
			talksCollection.Fields.RemoveById(slugField.GetId())
		}

		// Remove the index
		for i, idx := range talksCollection.Indexes {
			if idx == "CREATE UNIQUE INDEX idx_talks_slug ON talks(slug) WHERE slug != ''" {
				talksCollection.Indexes = append(talksCollection.Indexes[:i], talksCollection.Indexes[i+1:]...)
				break
			}
		}

		return app.Save(talksCollection)
	})
}
