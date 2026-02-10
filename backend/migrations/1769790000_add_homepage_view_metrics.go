package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds homepage view metrics fields to the site_settings collection.
// - homepage_view_count: total public/unauthed homepage fetches
// - homepage_last_viewed_at: timestamp of most recent public/unauthed homepage fetch
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		// Add homepage_view_count if missing
		if collection.Fields.GetByName("homepage_view_count") == nil {
			collection.Fields.Add(&core.NumberField{
				Name: "homepage_view_count",
			})
		}

		// Add homepage_last_viewed_at if missing
		if collection.Fields.GetByName("homepage_last_viewed_at") == nil {
			collection.Fields.Add(&core.DateField{
				Name: "homepage_last_viewed_at",
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		// Remove fields on rollback (only if they exist)
		collection.Fields.RemoveByName("homepage_view_count")
		collection.Fields.RemoveByName("homepage_last_viewed_at")

		return app.Save(collection)
	})
}
