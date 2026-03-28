package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Combined migration: view metrics, GA measurement ID, and homepage metrics.
// All additions are idempotent (checked before adding).
func init() {
	m.Register(func(app core.App) error {
		// 1. Add view_count and last_viewed_at to views collection
		views, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			changed := false
			if views.Fields.GetByName("view_count") == nil {
				views.Fields.Add(&core.NumberField{
					Name: "view_count",
				})
				changed = true
			}
			if views.Fields.GetByName("last_viewed_at") == nil {
				views.Fields.Add(&core.DateField{
					Name: "last_viewed_at",
				})
				changed = true
			}
			if changed {
				if err := app.Save(views); err != nil {
					return err
				}
			}
		}

		// 2. Add ga_measurement_id, homepage_view_count, and homepage_last_viewed_at to site_settings
		siteSettings, err := app.FindCollectionByNameOrId("site_settings")
		if err == nil {
			changed := false
			if siteSettings.Fields.GetByName("ga_measurement_id") == nil {
				siteSettings.Fields.Add(&core.TextField{
					Name: "ga_measurement_id",
					Max:  100,
				})
				changed = true
			}
			if siteSettings.Fields.GetByName("homepage_view_count") == nil {
				siteSettings.Fields.Add(&core.NumberField{
					Name: "homepage_view_count",
				})
				changed = true
			}
			if siteSettings.Fields.GetByName("homepage_last_viewed_at") == nil {
				siteSettings.Fields.Add(&core.DateField{
					Name: "homepage_last_viewed_at",
				})
				changed = true
			}
			if changed {
				if err := app.Save(siteSettings); err != nil {
					return err
				}
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove all added fields
		views, err := app.FindCollectionByNameOrId("views")
		if err == nil {
			views.Fields.RemoveByName("view_count")
			views.Fields.RemoveByName("last_viewed_at")
			app.Save(views)
		}

		siteSettings, err := app.FindCollectionByNameOrId("site_settings")
		if err == nil {
			siteSettings.Fields.RemoveByName("ga_measurement_id")
			siteSettings.Fields.RemoveByName("homepage_view_count")
			siteSettings.Fields.RemoveByName("homepage_last_viewed_at")
			app.Save(siteSettings)
		}

		return nil
	})
}
