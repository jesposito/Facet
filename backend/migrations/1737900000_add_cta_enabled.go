package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add site_cta_enabled to site_settings collection
		siteSettings, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil // Collection doesn't exist yet, skip
		}

		// Check if field already exists
		if siteSettings.Fields.GetByName("site_cta_enabled") == nil {
			// Add site_cta_enabled boolean field (defaults to true in Go)
			siteSettings.Fields.Add(&core.BoolField{
				Name: "site_cta_enabled",
			})

			if err := app.Save(siteSettings); err != nil {
				return err
			}
		}

		// Add cta_enabled to views collection
		views, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		// Check if field already exists
		if views.Fields.GetByName("cta_enabled") == nil {
			// Add cta_enabled boolean field (defaults to true in Go)
			views.Fields.Add(&core.BoolField{
				Name: "cta_enabled",
			})

			if err := app.Save(views); err != nil {
				return err
			}
		}

		// Set default value of true for all existing views that have a CTA configured
		// This preserves existing behavior - views with CTA URLs will continue to show the button
		viewRecords, err := app.FindRecordsByFilter(
			"views",
			"cta_url != ''",
			"",
			0,
			0,
			nil,
		)
		if err != nil {
			return nil // Not critical, skip
		}

		for _, record := range viewRecords {
			record.Set("cta_enabled", true)
			if err := app.Save(record); err != nil {
				app.Logger().Warn("Failed to set cta_enabled for view", "id", record.Id, "error", err)
			}
		}

		// Set site_cta_enabled to true for existing site_settings
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
			record.Set("site_cta_enabled", true)
			if err := app.Save(record); err != nil {
				app.Logger().Warn("Failed to set site_cta_enabled", "error", err)
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove fields
		siteSettings, err := app.FindCollectionByNameOrId("site_settings")
		if err == nil {
			siteSettings.Fields.RemoveByName("site_cta_enabled")
			app.Save(siteSettings)
		}

		views, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			views.Fields.RemoveByName("cta_enabled")
			app.Save(views)
		}

		return nil
	})
}
