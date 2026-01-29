package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration converts *_library_url fields from URLField to TextField
// because we store relative paths like /api/files/... which aren't valid URLs.
func init() {
	m.Register(func(app core.App) error {
		fieldsToFix := map[string][]string{
			"experience": {"company_logo_library_url"},
			"education":  {"institution_logo_library_url"},
			"projects":   {"cover_image_library_url"},
			"posts":      {"cover_image_library_url"},
			"talks":      {"cover_image_library_url"},
		}

		for collectionName, fieldNames := range fieldsToFix {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				continue // Collection doesn't exist, skip
			}

			needsSave := false
			for _, fieldName := range fieldNames {
				field := collection.Fields.GetByName(fieldName)
				if field == nil {
					continue // Field doesn't exist
				}

				// Check if it's a URLField (type "url")
				if field.Type() == "url" {
					// Remove the old URLField and add a TextField with the same name
					collection.Fields.RemoveById(field.GetId())
					collection.Fields.Add(&core.TextField{
						Name: fieldName,
					})
					needsSave = true
				}
			}

			if needsSave {
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}

		return nil
	}, nil) // No rollback needed
}
