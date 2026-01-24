package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get the experience collection
		collection, err := app.FindCollectionByNameOrId("experience")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("company_logo") != nil {
			return nil
		}

		// Add company_logo field - single file upload for company logos
		collection.Fields.Add(&core.FileField{
			Name:      "company_logo",
			MaxSize:   5242880, // 5MB max size (logos should be smaller than cover images)
			MaxSelect: 1,       // Single logo per company
			MimeTypes: []string{
				"image/jpeg",
				"image/jpg",
				"image/png",
				"image/webp",
				"image/svg+xml", // SVG logos are common for companies
			},
		})

		// Save the collection
		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: remove the company_logo field
		collection, err := app.FindCollectionByNameOrId("experience")
		if err != nil {
			return err
		}

		// Find and remove the company_logo field
		field := collection.Fields.GetByName("company_logo")
		if field != nil {
			collection.Fields.RemoveById(field.GetId())
			return app.Save(collection)
		}

		return nil
	})
}
