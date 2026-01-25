package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("custom"); err == nil {
			return nil
		}

		custom := core.NewBaseCollection("custom")
		rule := "visibility = 'public' && is_draft = false"
		custom.ListRule = &rule
		custom.ViewRule = &rule

		custom.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 300})
		custom.Fields.Add(&core.EditorField{Name: "content"})
		custom.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		custom.Fields.Add(&core.BoolField{Name: "is_draft"})
		custom.Fields.Add(&core.NumberField{Name: "sort_order"})
		custom.Fields.Add(&core.FileField{
			Name:      "media",
			MaxSelect: 10,
			MaxSize:   10485760,
			MimeTypes: []string{
				"image/jpeg",
				"image/png",
				"image/gif",
				"image/webp",
				"image/svg+xml",
			},
		})

		return app.Save(custom)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("custom")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
