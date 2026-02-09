package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("audit_logs")
		if err != nil {
			return err
		}

		// Add created/updated autodate fields that were missing from the original migration.
		// NewBaseCollection() does not add these automatically in PocketBase v0.23.
		collection.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})
		collection.Fields.Add(&core.AutodateField{
			Name:     "updated",
			OnCreate: true,
			OnUpdate: true,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("audit_logs")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("created")
		collection.Fields.RemoveByName("updated")

		return app.Save(collection)
	})
}
