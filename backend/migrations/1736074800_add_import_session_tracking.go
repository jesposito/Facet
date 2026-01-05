package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Collections that need import_session_id tracking:
		// - skills: to track which resume import created each skill
		// - education: to track source and support deduplication
		// - awards: to track source and support deduplication
		// - experience: to track source and enable faceted views

		collectionsToUpdate := []string{"skills", "education", "awards", "experience", "projects"}

		for _, collectionName := range collectionsToUpdate {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				// Collection doesn't exist yet, skip
				continue
			}

			// Add import_session_id field to track which resume import created this record
			// This allows us to:
			// 1. Dedupe within same session (same resume file)
			// 2. Apply different deduplication rules across sessions (different resumes)
			// 3. Tag items with the source resume for user visibility
			collection.Fields.Add(&core.TextField{
				Name: "import_session_id",
				Max:  36, // UUID length
			})

			// Add import_filename field to store the original resume filename for user visibility
			collection.Fields.Add(&core.TextField{
				Name: "import_filename",
				Max:  255,
			})

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove the fields
		collectionsToUpdate := []string{"skills", "education", "awards", "experience", "projects"}

		for _, collectionName := range collectionsToUpdate {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue
			}

			// Remove import fields
			if field := collection.Fields.GetByName("import_session_id"); field != nil {
				collection.Fields.RemoveById(field.GetId())
			}
			if field := collection.Fields.GetByName("import_filename"); field != nil {
				collection.Fields.RemoveById(field.GetId())
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
