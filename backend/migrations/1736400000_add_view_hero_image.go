package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("hero_image") != nil {
			return nil
		}

		collection.Fields.Add(&core.FileField{
			Name:      "hero_image",
			MaxSize:   10485760,
			MimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
		})

		return app.Save(collection)
	}, nil)
}
