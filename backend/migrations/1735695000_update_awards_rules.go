package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Bring the awards collection in line with other content collections:
// - Direct collection access requires authentication for all operations.
// - Public rendering continues to flow through our custom endpoints, which bypass collection rules.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("awards")
		if err != nil {
			return nil
		}

		authRule := "@request.auth.id != ''"

		collection.ListRule = &authRule
		collection.ViewRule = &authRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("awards")
		if err != nil {
			return nil
		}

		// Restore the original public read rules and admin-only writes.
		publicRule := "visibility = 'public' && is_draft = false"
		collection.ListRule = &publicRule
		collection.ViewRule = &publicRule
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		return app.Save(collection)
	})
}
