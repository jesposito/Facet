package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get resume_imports collection for relation
		resumeImportsCollection, err := app.FindCollectionByNameOrId("resume_imports")
		if err != nil {
			return err // Can't add relation if target collection doesn't exist
		}

		// Add resume_import_id relation field to all collections that support resume imports
		// This replaces the old import_session_id + import_filename approach with a proper relation
		collectionsToUpdate := []string{"skills", "education", "awards", "experience", "projects"}

		for _, collectionName := range collectionsToUpdate {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				// Collection doesn't exist yet, skip
				continue
			}

			// Add resume_import_id relation field pointing to resume_imports collection
			collection.Fields.Add(&core.RelationField{
				Name:          "resume_import_id",
				MaxSelect:     1,
				CollectionId:  resumeImportsCollection.Id,
				CascadeDelete: false,
				Required:      false,
			})

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove resume_import_id field
		collectionsToUpdate := []string{"skills", "education", "awards", "experience", "projects"}

		for _, collectionName := range collectionsToUpdate {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue
			}

			if field := collection.Fields.GetByName("resume_import_id"); field != nil {
				collection.Fields.RemoveById(field.GetId())
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
