package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Fix subscribers collection ACL so authenticated admin users can list/view
// records (e.g. for dashboard widgets that query the collection directly).
//
// The original migration set all rules to nil, which means superuser-only.
// Aligns with cloud behaviour: any authenticated user can list/view/update/delete;
// create is still handled by custom endpoints (signup form).
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return nil // Collection missing, nothing to do
		}

		authRule := "@request.auth.id != ''"
		collection.ListRule = &authRule
		collection.ViewRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule
		// CreateRule stays nil — public signup goes through custom endpoint
		collection.CreateRule = nil

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: restore nil rules (superuser-only)
		collection, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return nil
		}
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil
		collection.CreateRule = nil
		return app.Save(collection)
	})
}
