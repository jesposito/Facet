package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Clone existing collections to create demo_* shadow tables
		collectionsToClone := []string{
			"profile", "experience", "projects", "education",
			"skills", "certifications", "posts", "talks",
			"awards", "views", "share_tokens",
		}

		for _, collName := range collectionsToClone {
			demoCollName := "demo_" + collName

			// Skip if demo collection already exists
			if _, err := app.FindCollectionByNameOrId(demoCollName); err == nil {
				continue
			}

			// Get the source collection
			sourceCollection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue // Skip if source doesn't exist
			}

			// Create new collection with same schema
			demoCollection := core.NewBaseCollection(demoCollName)

			// Copy all fields from source
			for _, field := range sourceCollection.Fields {
				demoCollection.Fields.Add(field)
			}

			// Copy access rules from source collection
			// This ensures demo_* collections have same permissions as original tables
			demoCollection.ListRule = sourceCollection.ListRule
			demoCollection.ViewRule = sourceCollection.ViewRule
			demoCollection.CreateRule = sourceCollection.CreateRule
			demoCollection.UpdateRule = sourceCollection.UpdateRule
			demoCollection.DeleteRule = sourceCollection.DeleteRule

			if err := app.Save(demoCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete all demo tables
		tables := []string{
			"demo_profile", "demo_experience", "demo_projects", "demo_education",
			"demo_skills", "demo_certifications", "demo_posts", "demo_talks",
			"demo_awards", "demo_views", "demo_share_tokens",
		}
		for _, table := range tables {
			if coll, err := app.FindCollectionByNameOrId(table); err == nil {
				app.Delete(coll)
			}
		}
		return nil
	})
}
