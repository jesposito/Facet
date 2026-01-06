package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("share_tokens")
		if err != nil {
			return nil
		}

		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return nil
		}

		viewIdField := collection.Fields.GetByName("view_id")
		if viewIdField == nil {
			return nil
		}

		if _, ok := viewIdField.(*core.TextField); !ok {
			return nil
		}

		// Step 1: Read all existing tokens and their view_id values before schema change
		existingTokens, err := app.FindAllRecords("share_tokens")
		if err != nil {
			existingTokens = []*core.Record{}
		}

		viewIdBackup := make(map[string]string)
		for _, token := range existingTokens {
			viewIdBackup[token.Id] = token.GetString("view_id")
		}

		// Step 2: Change schema - remove TextField, add RelationField
		collection.Fields.RemoveByName("view_id")

		collection.Fields.Add(&core.RelationField{
			Name:         "view_id",
			Required:     false, // Temporarily not required during migration
			CollectionId: viewsCollection.Id,
			MaxSelect:    1,
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		// Step 3: Backfill - restore view_id values
		for tokenId, viewId := range viewIdBackup {
			if viewId == "" {
				continue
			}
			// Verify the view exists
			_, err := app.FindRecordById("views", viewId)
			if err != nil {
				continue
			}

			token, err := app.FindRecordById("share_tokens", tokenId)
			if err != nil {
				continue
			}

			token.Set("view_id", viewId)
			app.Save(token)
		}

		// Step 4: Make field required now that data is backfilled
		collection, _ = app.FindCollectionByNameOrId("share_tokens")
		viewIdField = collection.Fields.GetByName("view_id")
		if relField, ok := viewIdField.(*core.RelationField); ok {
			relField.Required = true
			app.Save(collection)
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("share_tokens")
		if err != nil {
			return nil
		}

		viewIdField := collection.Fields.GetByName("view_id")
		if viewIdField == nil {
			return nil
		}

		if _, ok := viewIdField.(*core.RelationField); !ok {
			return nil
		}

		// Backup existing relation values
		existingTokens, err := app.FindAllRecords("share_tokens")
		if err != nil {
			existingTokens = []*core.Record{}
		}

		viewIdBackup := make(map[string]string)
		for _, token := range existingTokens {
			viewIdBackup[token.Id] = token.GetString("view_id")
		}

		// Change schema back to TextField
		collection.Fields.RemoveByName("view_id")

		collection.Fields.Add(&core.TextField{
			Name:     "view_id",
			Required: false,
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		// Restore values
		for tokenId, viewId := range viewIdBackup {
			if viewId == "" {
				continue
			}
			token, err := app.FindRecordById("share_tokens", tokenId)
			if err != nil {
				continue
			}
			token.Set("view_id", viewId)
			app.Save(token)
		}

		// Make required again
		collection, _ = app.FindCollectionByNameOrId("share_tokens")
		viewIdField = collection.Fields.GetByName("view_id")
		if textField, ok := viewIdField.(*core.TextField); ok {
			textField.Required = true
			app.Save(collection)
		}

		return nil
	})
}
