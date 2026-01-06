package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("share_tokens")
		if err != nil {
			return nil
		}

		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}

		viewIdField := collection.Fields.GetByName("view_id")
		if viewIdField == nil {
			return nil
		}

		if _, ok := viewIdField.(*core.TextField); !ok {
			return nil
		}

		collection.Fields.RemoveByName("view_id")

		collection.Fields.Add(&core.RelationField{
			Name:         "view_id",
			Required:     true,
			CollectionId: viewsCollection.Id,
			MaxSelect:    1,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("share_tokens")
		if err != nil {
			return nil
		}

		viewIdField := collection.Fields.GetByName("view_id")
		if viewIdField == nil {
			return nil
		}

		if _, ok := viewIdField.(*core.RelationField); !ok {
			return nil
		}

		collection.Fields.RemoveByName("view_id")

		collection.Fields.Add(&core.TextField{
			Name:     "view_id",
			Required: true,
		})

		return app.Save(collection)
	})
}
