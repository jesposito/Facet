package migrations

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	// PocketBase v0.23's NewBaseCollection() does not add created/updated
	// autodate fields automatically. This was previously fixed for audit_logs
	// (1769780000) but all other base collections have the same issue.
	//
	// Without these fields, records get created with empty created/updated
	// timestamps, causing "Created Never" display bugs in the frontend.
	m.Register(func(app core.App) error {
		collections := []string{
			"profile",
			"experience",
			"projects",
			"education",
			"certifications",
			"skills",
			"posts",
			"talks",
			"views",
			"share_tokens",
			"sources",
			"ai_providers",
			"import_proposals",
			"settings",
			"external_media",
			"custom_content",
			"site_settings",
			"media_display_names",
			"contact_methods",
			"resume_imports",
			"testimonials",
			"testimonial_requests",
			"email_verification_tokens",
			"admin_tags",
			"view_exports",
			"uploads",
			"awards",
		}

		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				log.Printf("Skipping %s: %v", name, err)
				continue
			}

			// Only add if not already present
			if collection.Fields.GetByName("created") == nil {
				collection.Fields.Add(&core.AutodateField{
					Name:     "created",
					OnCreate: true,
				})
			}
			if collection.Fields.GetByName("updated") == nil {
				collection.Fields.Add(&core.AutodateField{
					Name:     "updated",
					OnCreate: true,
					OnUpdate: true,
				})
			}

			if err := app.Save(collection); err != nil {
				log.Printf("Failed to fix autodate for %s: %v", name, err)
				continue
			}

			// Backfill existing records that have empty created timestamps.
			// Use raw SQL to bypass AutodateField interceptors which prevent
			// setting "created" on update operations.
			nowStr := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

			// Set created = updated where created is empty but updated exists
			res1, err := app.DB().NewQuery(
				"UPDATE " + collection.Name +
					" SET created = updated" +
					" WHERE (created IS NULL OR created = '')" +
					" AND updated IS NOT NULL AND updated != ''",
			).Execute()
			if err != nil {
				log.Printf("Failed to backfill %s (from updated): %v", name, err)
			}

			// Set created = now where both created and updated are empty
			res2, err := app.DB().NewQuery(
				"UPDATE " + collection.Name +
					" SET created = {:now}" +
					" WHERE (created IS NULL OR created = '')",
			).Bind(map[string]any{"now": nowStr}).Execute()
			if err != nil {
				log.Printf("Failed to backfill %s (from now): %v", name, err)
			}

			var count1, count2 int64
			if res1 != nil {
				count1, _ = res1.RowsAffected()
			}
			if res2 != nil {
				count2, _ = res2.RowsAffected()
			}
			if count1+count2 > 0 {
				log.Printf("Backfilled %d records in %s with missing created timestamp", count1+count2, name)
			}
		}

		return nil
	}, func(app core.App) error {
		// No rollback — autodate fields are expected on all collections
		return nil
	})
}
