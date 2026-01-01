package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ai_providers")
		if err != nil {
			return nil // Collection doesn't exist yet, skip
		}

		// Find and update the api_key field to not be hidden
		// Hidden fields can't be received in API requests, which breaks our encryption flow
		for _, field := range collection.Fields {
			if field.GetName() == "api_key" {
				if textField, ok := field.(*core.TextField); ok {
					textField.Hidden = false
				}
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: make api_key hidden again
		collection, err := app.FindCollectionByNameOrId("ai_providers")
		if err != nil {
			return nil
		}

		for _, field := range collection.Fields {
			if field.GetName() == "api_key" {
				if textField, ok := field.(*core.TextField); ok {
					textField.Hidden = true
				}
			}
		}

		return app.Save(collection)
	})
}
