package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds basic view analytics fields to the views collection.
// - view_count: total public/unauthed view fetches
// - last_viewed_at: timestamp of most recent public/unauthed fetch
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}

		// Add view_count if missing
		if collection.Fields.GetByName("view_count") == nil {
			collection.Fields.Add(&core.NumberField{
				Name: "view_count",
			})
		}

		// Add last_viewed_at if missing
		if collection.Fields.GetByName("last_viewed_at") == nil {
			collection.Fields.Add(&core.DateField{
				Name: "last_viewed_at",
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}

		// Remove fields on rollback (only if they exist)
		collection.Fields.RemoveByName("view_count")
		collection.Fields.RemoveByName("last_viewed_at")

		return app.Save(collection)
	})
}
