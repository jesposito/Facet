package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add token_prefix field to share_tokens for O(1) lookup
		collection, err := app.FindCollectionByNameOrId("share_tokens")
		if err != nil {
			return err
		}

		// Check if field already exists
		if collection.Fields.GetByName("token_prefix") != nil {
			return nil
		}

		// Add token_prefix field (first 12 chars of raw token for indexed lookup)
		collection.Fields.Add(&core.TextField{
			Name:     "token_prefix",
			Max:      16,
			Required: false, // Not required for backwards compatibility with existing tokens
		})

		// Add composite index for efficient lookup: prefix + is_active
		collection.Indexes = append(collection.Indexes,
			"CREATE INDEX idx_share_tokens_prefix_active ON share_tokens(token_prefix, is_active) WHERE token_prefix != ''",
		)

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: remove token_prefix field
		collection, err := app.FindCollectionByNameOrId("share_tokens")
		if err != nil {
			return err
		}

		field := collection.Fields.GetByName("token_prefix")
		if field != nil {
			collection.Fields.RemoveById(field.GetId())
		}

		// Note: Index will be removed automatically when field is removed
		return app.Save(collection)
	})
}
