package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// paidFileFields are the file fields that hold what a buyer pays for.
// PocketBase serves an unprotected file to anyone who knows its URL, which
// skipped the purchase check. Protected files need a PocketBase file token, so
// buyers get them through the signed links at /api/downloads/{token} instead.
var paidFileFields = map[string]string{
	"posts":               "downloadable_files",
	"projects":            "downloadable_files",
	"custom_content":      "downloadable_files",
	"talks":               "downloadable_files",
	"demo_posts":          "downloadable_files",
	"demo_projects":       "downloadable_files",
	"demo_custom_content": "downloadable_files",
	"demo_talks":          "downloadable_files",
	"lessons":             "attachments",
}

func init() {
	m.Register(func(app core.App) error {
		return setPaidFilesProtected(app, true)
	}, func(app core.App) error {
		return setPaidFilesProtected(app, false)
	})
}

func setPaidFilesProtected(app core.App, protected bool) error {
	for collectionName, fieldName := range paidFileFields {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil || collection == nil {
			continue
		}
		field, ok := collection.Fields.GetByName(fieldName).(*core.FileField)
		if !ok || field.Protected == protected {
			continue
		}
		field.Protected = protected
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
