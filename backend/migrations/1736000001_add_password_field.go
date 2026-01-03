package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// add_password_field adds a temporary password field to views collection
// This field is used to accept plaintext passwords from the frontend,
// which are then hashed and stored in password_hash by the OnRecordCreate/Update hooks
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		// Add temporary password field (not stored, used only for hashing)
		// Note: NOT hidden so it can be accepted from API requests, but cleared by hook
		collection.Fields.Add(&core.TextField{
			Name: "password",
		})

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback - remove password field
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("password")

		return app.Save(collection)
	}, "1736000001")
}
