package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds external_media collection for link-based media references.
func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("external_media")

		collection.Fields.Add(&core.TextField{
			Name:     "url",
			Required: true,
		})

		collection.Fields.Add(&core.TextField{
			Name: "title",
			Max:  255,
		})

		collection.Fields.Add(&core.TextField{
			Name: "mime",
			Max:  255,
		})

		collection.Fields.Add(&core.TextField{
			Name: "thumbnail_url",
			Max:  500,
		})

		authRule := "@request.auth.id != ''"
		publicRule := ""
		collection.ListRule = &publicRule
		collection.ViewRule = &publicRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
