package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add is_default field to views collection
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		// Add is_default boolean field
		// Only one view can be marked as default at a time (enforced by hook)
		viewsCollection.Fields.Add(&core.BoolField{
			Name: "is_default",
		})

		if err := app.Save(viewsCollection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove is_default field
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		field := viewsCollection.Fields.GetByName("is_default")
		if field != nil {
			viewsCollection.Fields.RemoveById(field.GetId())
			return app.Save(viewsCollection)
		}

		return nil
	})
}
