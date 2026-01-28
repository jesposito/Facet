package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get the education collection
		collection, err := app.FindCollectionByNameOrId("education")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("institution_logo") != nil {
			return nil
		}

		// Add institution_logo field - single file upload for institution logos
		collection.Fields.Add(&core.FileField{
			Name:      "institution_logo",
			MaxSize:   5242880, // 5MB max size (logos should be smaller than cover images)
			MaxSelect: 1,       // Single logo per institution
			MimeTypes: []string{
				"image/jpeg",
				"image/jpg",
				"image/png",
				"image/webp",
				"image/svg+xml", // SVG logos are common for institutions
			},
		})

		// Save the collection
		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: remove the institution_logo field
		collection, err := app.FindCollectionByNameOrId("education")
		if err != nil {
			return err
		}

		// Find and remove the institution_logo field
		field := collection.Fields.GetByName("institution_logo")
		if field != nil {
			collection.Fields.RemoveById(field.GetId())
			return app.Save(collection)
		}

		return nil
	})
}
