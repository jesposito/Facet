package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds contact_methods collection for granular contact information with anti-scraping protection.
// Part of Phase 11: Contact Protection & Social Links
func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("contact_methods")

		// Type of contact (email, phone, linkedin, github, twitter, etc.)
		collection.Fields.Add(&core.SelectField{
			Name:     "type",
			Required: true,
			MaxSelect: 1,
			Values: []string{
				"email",
				"phone",
				"linkedin",
				"github",
				"twitter",
				"facebook",
				"instagram",
				"youtube",
				"mastodon",
				"bluesky",
				"website",
				"portfolio",
				"blog",
				"custom",
			},
		})

		// The actual contact value (email address, phone number, URL, etc.)
		collection.Fields.Add(&core.TextField{
			Name:     "value",
			Required: true,
			Max:      500,
		})

		// User-friendly label (e.g., "Work Email", "Personal Phone")
		collection.Fields.Add(&core.TextField{
			Name: "label",
			Max:  100,
		})

		// Protection level for anti-scraping
		collection.Fields.Add(&core.SelectField{
			Name:     "protection_level",
			Required: true,
			MaxSelect: 1,
			Values: []string{
				"none",           // Plaintext, no protection
				"obfuscation",    // CSS tricks + decoy characters
				"click_to_reveal", // User must click to see value
				"captcha",        // Cloudflare Turnstile required
			},
		})

		// JSON object mapping view IDs to visibility boolean
		// Example: {"view_abc123": true, "view_xyz789": false}
		collection.Fields.Add(&core.JSONField{
			Name:     "view_visibility",
			Required: false,
		})

		// Whether this is the primary contact method of its type
		collection.Fields.Add(&core.BoolField{
			Name: "is_primary",
		})

		// Display order within its category
		collection.Fields.Add(&core.NumberField{
			Name:    "sort_order",
			Min:     floatPtr(0),
			Max:     floatPtr(999),
		})

		// Optional icon override (defaults to type-based icon)
		collection.Fields.Add(&core.TextField{
			Name: "icon",
			Max:  50,
		})

		// Access rules: Only authenticated users can manage
		authRule := "@request.auth.id != ''"
		publicRule := "" // Public read for rendering in views

		collection.ListRule = &publicRule
		collection.ViewRule = &publicRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: delete the collection
		collection, err := app.FindCollectionByNameOrId("contact_methods")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}

// Helper function for number field min/max
func floatPtr(f float64) *float64 {
	return &f
}
