package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/migrate"
)

// add_password_field adds a temporary password field to views collection
// This field is used to accept plaintext passwords from the frontend,
// which are then hashed and stored in password_hash by the OnRecordCreate/Update hooks
func init() {
	migrate.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		// Add temporary password field (not stored, used only for hashing)
		collection.Fields.Add(&core.TextField{
			Name:   "password",
			Hidden: true, // Hide from API responses
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
