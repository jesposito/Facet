package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds stripe_secret_key and stripe_webhook_secret fields to site_settings.
// Both fields store AES-256-GCM encrypted values (via CryptoService).
// Empty string means "not configured in DB - fall back to environment variable".
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil // site_settings may not exist in all setups
		}

		changed := false

		if collection.Fields.GetByName("stripe_secret_key") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "stripe_secret_key",
				Max:  255,
			})
			changed = true
		}

		if collection.Fields.GetByName("stripe_webhook_secret") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "stripe_webhook_secret",
				Max:  255,
			})
			changed = true
		}

		if !changed {
			return nil
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		for _, name := range []string{"stripe_secret_key", "stripe_webhook_secret"} {
			if f := collection.Fields.GetByName(name); f != nil {
				collection.Fields.RemoveById(f.GetId())
			}
		}

		return app.Save(collection)
	})
}
