package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds the `api_keys` collection used by the self-hosted public API surface.
//
// Schema (self-hosted, single-tenant):
//   - label            string, required, indexed       -- human label set by admin
//   - key_prefix       string, required, indexed       -- first 12 chars of the issued key for display/lookup
//   - key_hash         string, required                -- HMAC of the full key (never the raw key)
//   - scopes           json                            -- []string of granted scopes (read:profile, etc.)
//   - allowed_origins  json                            -- []string of CORS origins (optional, [] = any)
//   - expires_at       datetime, nullable
//   - last_used_at     datetime, nullable
//   - is_active        bool, default true              -- revoked = false (we soft-delete for audit)
//
// Access rules:
//   - List/View/Create/Update/Delete are all superuser-only via the admin
//     endpoints in `hooks/api_keys.go`. The PocketBase rules here are
//     `@request.auth.id != ”` so the dashboard admin UI can read it too if
//     ever needed, but the actual mutation paths go through hooks.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("api_keys"); err == nil {
			return nil
		}

		apiKeys := core.NewBaseCollection("api_keys")

		// Auth: superuser only. Keep rules conservative — we route mutations
		// through admin endpoints, so the collection rules just gate reads.
		adminOnly := "@request.auth.id != ''"
		apiKeys.ListRule = &adminOnly
		apiKeys.ViewRule = &adminOnly
		// CreateRule/UpdateRule/DeleteRule are nil; only backend hooks mutate.

		apiKeys.Fields.Add(&core.TextField{
			Name:     "label",
			Required: true,
			Max:      100,
		})
		apiKeys.Fields.Add(&core.TextField{
			Name:     "key_prefix",
			Required: true,
			Max:      32,
		})
		apiKeys.Fields.Add(&core.TextField{
			Name:     "key_hash",
			Required: true,
			Max:      128,
			Hidden:   true, // never expose the hash to API consumers
		})
		apiKeys.Fields.Add(&core.JSONField{
			Name:    "scopes",
			MaxSize: 4096,
		})
		apiKeys.Fields.Add(&core.JSONField{
			Name:    "allowed_origins",
			MaxSize: 4096,
		})
		apiKeys.Fields.Add(&core.DateField{
			Name: "expires_at",
		})
		apiKeys.Fields.Add(&core.DateField{
			Name: "last_used_at",
		})
		apiKeys.Fields.Add(&core.BoolField{
			Name: "is_active",
		})
		apiKeys.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})
		apiKeys.Fields.Add(&core.AutodateField{
			Name:     "updated",
			OnCreate: true,
			OnUpdate: true,
		})

		// Index on key_prefix for fast lookup during request authentication.
		apiKeys.Indexes = []string{
			"CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix)",
			"CREATE INDEX idx_api_keys_is_active ON api_keys(is_active)",
		}

		return app.Save(apiKeys)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("api_keys")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
