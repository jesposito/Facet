package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds media_display_names collection to store custom display names for media files.
// This allows renaming media files without changing the parent record's title.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("media_display_names")
		if err == nil && collection != nil {
			return nil
		}

		coll := core.NewBaseCollection("media_display_names")

		// Composite key: collection_id + record_id + filename
		coll.Fields.Add(&core.TextField{
			Name:     "collection_id",
			Required: true,
		})
		coll.Fields.Add(&core.TextField{
			Name:     "record_id",
			Required: true,
		})
		coll.Fields.Add(&core.TextField{
			Name:     "filename",
			Required: true,
		})
		coll.Fields.Add(&core.TextField{
			Name:     "display_name",
			Required: true,
			Max:      255,
		})

		// Create unique index on the composite key
		coll.Indexes = []string{
			"CREATE UNIQUE INDEX idx_media_key ON media_display_names (collection_id, record_id, filename)",
		}

		authRule := "@request.auth.id != ''"
		coll.ListRule = &authRule
		coll.ViewRule = &authRule
		coll.CreateRule = &authRule
		coll.UpdateRule = &authRule
		coll.DeleteRule = &authRule

		return app.Save(coll)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("media_display_names")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
