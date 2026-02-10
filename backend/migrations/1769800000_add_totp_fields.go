package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds TOTP two-factor authentication fields to the users collection.
// - totp_secret: AES-encrypted TOTP secret key
// - totp_enabled: whether 2FA is active for this user
// - totp_recovery_codes: AES-encrypted JSON array of bcrypt-hashed recovery codes
// - totp_session_nonce: random nonce for session verification after TOTP
// - totp_session_expires: when the TOTP session verification expires
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		if collection.Fields.GetByName("totp_secret") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "totp_secret",
			})
		}

		if collection.Fields.GetByName("totp_enabled") == nil {
			collection.Fields.Add(&core.BoolField{
				Name: "totp_enabled",
			})
		}

		if collection.Fields.GetByName("totp_recovery_codes") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "totp_recovery_codes",
			})
		}

		if collection.Fields.GetByName("totp_session_nonce") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "totp_session_nonce",
			})
		}

		if collection.Fields.GetByName("totp_session_expires") == nil {
			collection.Fields.Add(&core.DateField{
				Name: "totp_session_expires",
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("totp_secret")
		collection.Fields.RemoveByName("totp_enabled")
		collection.Fields.RemoveByName("totp_recovery_codes")
		collection.Fields.RemoveByName("totp_session_nonce")
		collection.Fields.RemoveByName("totp_session_expires")

		return app.Save(collection)
	})
}
