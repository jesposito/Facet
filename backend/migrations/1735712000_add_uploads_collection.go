package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds a generic uploads collection for media library direct uploads.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("uploads")
		if err == nil && collection != nil {
			return nil
		}

		uploads := core.NewBaseCollection("uploads")
		uploads.Fields.Add(&core.FileField{
			Name:      "file",
			Required:  true,
			MaxSelect: 1,
			MaxSize:   20971520, // 20MB
		})
		uploads.Fields.Add(&core.TextField{
			Name: "title",
			Max:  255,
		})
		uploads.Fields.Add(&core.TextField{
			Name: "mime",
			Max:  255,
		})

		authRule := "@request.auth.id != ''"
		uploads.ListRule = &authRule
		uploads.ViewRule = &authRule
		uploads.CreateRule = &authRule
		uploads.UpdateRule = &authRule
		uploads.DeleteRule = &authRule

		return app.Save(uploads)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("uploads")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
