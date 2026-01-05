package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// add_password_changed_flag adds password_changed_from_default field to users collection
// This is used to track if the admin has changed from the default "changeme123" password
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		// Add boolean field to track if password has been changed from default
		collection.Fields.Add(&core.BoolField{
			Name:     "password_changed_from_default",
			Required: false,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback - remove password_changed_from_default field
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("password_changed_from_default")

		return app.Save(collection)
	}, "1736100000")
}
