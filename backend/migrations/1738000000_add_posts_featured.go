package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return nil // Collection doesn't exist, skip
		}

		// Add featured field if it doesn't exist
		if collection.Fields.GetByName("featured") == nil {
			collection.Fields.Add(&core.BoolField{
				Name: "featured",
			})
			return app.Save(collection)
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("featured")
		if field != nil {
			collection.Fields.RemoveByName("featured")
			return app.Save(collection)
		}

		return nil
	})
}
