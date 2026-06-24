package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add footer_cta_enabled to site_settings collection.
		// Defaults to true (in Go and for existing records below) so existing
		// instances are unchanged: the Soft Premium footer CTA band keeps showing.
		siteSettings, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil // Collection doesn't exist yet, skip
		}

		// Check if field already exists
		if siteSettings.Fields.GetByName("footer_cta_enabled") == nil {
			// Add footer_cta_enabled boolean field (defaults to true in Go)
			siteSettings.Fields.Add(&core.BoolField{
				Name: "footer_cta_enabled",
			})

			if err := app.Save(siteSettings); err != nil {
				return err
			}
		}

		// Set footer_cta_enabled to true for existing site_settings so the band
		// keeps showing on instances that predate this toggle (zero regression).
		settingsRecords, err := app.FindRecordsByFilter(
			"site_settings",
			"",
			"",
			1,
			0,
			nil,
		)
		if err != nil {
			return nil // Not critical, skip
		}

		for _, record := range settingsRecords {
			record.Set("footer_cta_enabled", true)
			if err := app.Save(record); err != nil {
				app.Logger().Warn("Failed to set footer_cta_enabled", "error", err)
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove field
		siteSettings, err := app.FindCollectionByNameOrId("site_settings")
		if err == nil {
			siteSettings.Fields.RemoveByName("footer_cta_enabled")
			app.Save(siteSettings)
		}

		return nil
	})
}
