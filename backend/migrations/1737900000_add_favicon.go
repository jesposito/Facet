package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil // Collection doesn't exist yet, skip
		}

		// Add favicon file field
		collection.Fields.Add(&core.FileField{
			Name:      "favicon",
			MaxSelect: 1,
			MaxSize:   524288, // 512KB
			MimeTypes: []string{
				"image/x-icon",
				"image/vnd.microsoft.icon",
				"image/png",
				"image/svg+xml",
				"image/gif",
			},
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("favicon")

		return app.Save(collection)
	})
}
