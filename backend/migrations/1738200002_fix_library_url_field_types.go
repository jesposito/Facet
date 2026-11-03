package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration converts *_library_url fields from URLField to TextField
// because we store relative paths like /api/files/... which aren't valid URLs.
//
// PocketBase won't change a field's type in place, so each URL field is
// replaced by a new TextField with the same name. That drops and recreates the
// column, so the existing values are copied out first and written back with
// raw SQL. Raw SQL leaves other fields, timestamps and record hooks alone.
func init() {
	m.Register(func(app core.App) error {
		fieldsToFix := map[string][]string{
			"experience": {"company_logo_library_url"},
			"education":  {"institution_logo_library_url"},
			"projects":   {"cover_image_library_url"},
			"posts":      {"cover_image_library_url"},
			"talks":      {"cover_image_library_url"},
		}

		return app.RunInTransaction(func(txApp core.App) error {
			for collectionName, fieldNames := range fieldsToFix {
				collection, err := txApp.FindCollectionByNameOrId(collectionName)
				if errors.Is(err, sql.ErrNoRows) {
					continue // Collection doesn't exist, skip
				}
				if err != nil {
					return fmt.Errorf("find %s: %w", collectionName, err)
				}

				for _, fieldName := range fieldNames {
					if err := convertURLFieldToText(txApp, collection, fieldName); err != nil {
						return fmt.Errorf("convert %s.%s: %w", collectionName, fieldName, err)
					}
				}
			}
			return nil
		})
	}, nil) // No rollback needed
}

func convertURLFieldToText(app core.App, collection *core.Collection, fieldName string) error {
	field := collection.Fields.GetByName(fieldName)
	if field == nil || field.Type() != core.FieldTypeURL {
		return nil // Missing or already converted
	}
	urlField, ok := field.(*core.URLField)
	if !ok {
		return fmt.Errorf("unexpected field implementation %T", field)
	}

	type savedValue struct {
		Id    string         `db:"id"`
		Value sql.NullString `db:"value"`
	}
	var saved []savedValue
	err := app.DB().
		NewQuery(fmt.Sprintf("SELECT [[id]], [[%s]] AS [[value]] FROM {{%s}}", fieldName, collection.Name)).
		All(&saved)
	if err != nil {
		return fmt.Errorf("read values: %w", err)
	}

	collection.Fields.RemoveById(urlField.Id)
	collection.Fields.Add(&core.TextField{
		Name:        urlField.Name,
		System:      urlField.System,
		Hidden:      urlField.Hidden,
		Presentable: urlField.Presentable,
		Required:    urlField.Required,
	})
	if err := app.Save(collection); err != nil {
		return fmt.Errorf("save collection: %w", err)
	}

	for _, row := range saved {
		if !row.Value.Valid || row.Value.String == "" {
			continue // The new column already defaults to an empty string
		}
		_, err := app.DB().Update(collection.Name, dbx.Params{fieldName: row.Value.String}, dbx.HashExp{"id": row.Id}).Execute()
		if err != nil {
			return fmt.Errorf("restore record %s: %w", row.Id, err)
		}
	}

	return nil
}
