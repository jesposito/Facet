package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add visibility fields to testimonials collection
		collection, err := app.FindCollectionByNameOrId("testimonials")
		if err != nil {
			return nil // Collection doesn't exist yet, skip
		}

		// Add visibility field if it doesn't exist
		if collection.Fields.GetByName("visibility") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "visibility",
				MaxSelect: 1,
				Values:    []string{"public", "unlisted", "private"},
			})
		}

		// Add view_visibility JSON field for per-view visibility (same pattern as other content types)
		if collection.Fields.GetByName("view_visibility") == nil {
			collection.Fields.Add(&core.JSONField{
				Name:     "view_visibility",
				Required: false,
			})
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Set default visibility to "public" for all existing approved testimonials
		// This preserves existing behavior where approved testimonials are publicly visible
		testimonials, err := app.FindRecordsByFilter(
			"testimonials",
			"status = 'approved'",
			"",
			0,
			0,
			nil,
		)
		if err != nil {
			return nil // Not critical, skip
		}

		for _, record := range testimonials {
			// Only set if not already set
			if record.GetString("visibility") == "" {
				record.Set("visibility", "public")
				if err := app.Save(record); err != nil {
					app.Logger().Warn("Failed to set visibility for testimonial", "id", record.Id, "error", err)
				}
			}
		}

		// Set default visibility to "private" for pending/rejected testimonials
		pendingTestimonials, err := app.FindRecordsByFilter(
			"testimonials",
			"status != 'approved'",
			"",
			0,
			0,
			nil,
		)
		if err != nil {
			return nil // Not critical, skip
		}

		for _, record := range pendingTestimonials {
			if record.GetString("visibility") == "" {
				record.Set("visibility", "private")
				if err := app.Save(record); err != nil {
					app.Logger().Warn("Failed to set visibility for testimonial", "id", record.Id, "error", err)
				}
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove visibility fields
		collection, err := app.FindCollectionByNameOrId("testimonials")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("visibility")
		collection.Fields.RemoveByName("view_visibility")

		return app.Save(collection)
	})
}
