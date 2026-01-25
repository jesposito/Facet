package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("custom")
		if err != nil {
			return err
		}

		// 1. Add view_visibility field if missing
		if collection.Fields.GetByName("view_visibility") == nil {
			collection.Fields.Add(&core.JSONField{
				Name:     "view_visibility",
				Required: false,
			})
		}

		// 2. Fix API rules (lock down direct access, allow auth users to manage)
		authRule := "@request.auth.id != ''"

		// Public read is handled by custom endpoints, but we allow list/view for auth users (admin)
		// checking strict equality to avoid nil pointer issues if rule is nil
		collection.ListRule = &authRule
		collection.ViewRule = &authRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		return app.Save(collection)
	}, func(app core.App) error {
		// No revert needed for rules/fields addition typically,
		// but we could remove view_visibility if we wanted to be strict.
		return nil
	})
}
