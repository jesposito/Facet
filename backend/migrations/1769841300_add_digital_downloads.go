package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds digital download and drip content fields to posts, projects, custom_content,
// talks, and modules collections. Also creates the download_logs collection.
func init() {
	m.Register(func(app core.App) error {
		// Add downloadable_files, drip_enabled, drip_unlock_date, drip_days_after_purchase
		// to posts, projects, custom_content
		contentCollections := []string{"posts", "projects", "custom_content"}

		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			if collection.Fields.GetByName("downloadable_files") == nil {
				collection.Fields.Add(&core.FileField{
					Name:      "downloadable_files",
					MaxSelect: 10,
					MaxSize:   104857600, // 100MB
				})
			}

			if collection.Fields.GetByName("drip_enabled") == nil {
				collection.Fields.Add(&core.BoolField{
					Name: "drip_enabled",
				})
			}

			if collection.Fields.GetByName("drip_unlock_date") == nil {
				collection.Fields.Add(&core.DateField{
					Name: "drip_unlock_date",
				})
			}

			if collection.Fields.GetByName("drip_days_after_purchase") == nil {
				collection.Fields.Add(&core.NumberField{
					Name: "drip_days_after_purchase",
					Min:  floatPtr(0),
				})
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		// Add downloadable_files to talks
		if talks, err := app.FindCollectionByNameOrId("talks"); err == nil {
			if talks.Fields.GetByName("downloadable_files") == nil {
				talks.Fields.Add(&core.FileField{
					Name:      "downloadable_files",
					MaxSelect: 10,
					MaxSize:   104857600, // 100MB
				})
			}
			if err := app.Save(talks); err != nil {
				return err
			}
		}

		// Create download_logs collection
		if _, err := app.FindCollectionByNameOrId("download_logs"); err == nil {
			return nil // Already exists
		}

		downloadLogs := core.NewBaseCollection("download_logs")

		authRule := "@request.auth.id != ''"
		downloadLogs.ListRule = &authRule
		downloadLogs.ViewRule = &authRule
		downloadLogs.CreateRule = &authRule
		downloadLogs.UpdateRule = &authRule
		downloadLogs.DeleteRule = &authRule

		downloadLogs.Fields.Add(&core.TextField{
			Name:     "content_type",
			Required: true,
			Max:      50,
		})
		downloadLogs.Fields.Add(&core.TextField{
			Name:     "content_id",
			Required: true,
			Max:      15,
		})
		downloadLogs.Fields.Add(&core.TextField{
			Name:     "filename",
			Required: true,
			Max:      500,
		})
		downloadLogs.Fields.Add(&core.EmailField{
			Name: "buyer_email",
		})
		downloadLogs.Fields.Add(&core.AutodateField{
			Name:     "downloaded_at",
			OnCreate: true,
		})
		downloadLogs.Fields.Add(&core.TextField{
			Name: "ip_hash",
			Max:  64,
		})

		downloadLogs.Indexes = []string{
			"CREATE INDEX idx_download_logs_content ON download_logs (content_type, content_id)",
			"CREATE INDEX idx_download_logs_email ON download_logs (buyer_email)",
		}

		return app.Save(downloadLogs)
	}, func(app core.App) error {
		// Rollback: remove download_logs collection
		if collection, err := app.FindCollectionByNameOrId("download_logs"); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		// Remove download/drip fields from content collections
		contentCollections := []string{"posts", "projects", "custom_content"}
		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}

			fields := collection.Fields
			if f := fields.GetByName("downloadable_files"); f != nil {
				fields.RemoveById(f.GetId())
			}
			if f := fields.GetByName("drip_enabled"); f != nil {
				fields.RemoveById(f.GetId())
			}
			if f := fields.GetByName("drip_unlock_date"); f != nil {
				fields.RemoveById(f.GetId())
			}
			if f := fields.GetByName("drip_days_after_purchase"); f != nil {
				fields.RemoveById(f.GetId())
			}
			_ = app.Save(collection)
		}

		// Remove downloadable_files from talks
		if talks, err := app.FindCollectionByNameOrId("talks"); err == nil {
			if f := talks.Fields.GetByName("downloadable_files"); f != nil {
				talks.Fields.RemoveById(f.GetId())
			}
			_ = app.Save(talks)
		}

		return nil
	})
}
