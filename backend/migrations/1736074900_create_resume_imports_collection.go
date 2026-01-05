package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create resume_imports collection to track imported resume files
		// This prevents duplicate imports of the same file (by hash)
		collection := core.NewBaseCollection("resume_imports")

		// Fields
		collection.Fields.Add(&core.TextField{
			Name:     "hash",
			Required: true,
			Max:      64, // SHA256 = 64 hex chars
		})
		collection.Fields.Add(&core.TextField{
			Name:     "filename",
			Required: true,
			Max:      255,
		})
		collection.Fields.Add(&core.NumberField{
			Name: "file_size",
		})
		collection.Fields.Add(&core.JSONField{
			Name: "record_counts",
		})
		collection.Fields.Add(&core.DateField{
			Name: "imported_at",
		})

		// Create unique index on hash to prevent duplicate imports
		collection.Indexes = []string{
			"CREATE UNIQUE INDEX idx_resume_imports_hash ON resume_imports (hash)",
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: delete the collection
		collection, err := app.FindCollectionByNameOrId("resume_imports")
		if err != nil {
			return nil // Already deleted or never created
		}
		return app.Delete(collection)
	})
}
