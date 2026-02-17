package migrations

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	// Follow-up to 1769820000: the original backfill used app.Save(record)
	// which goes through PocketBase's AutodateField interceptors. The
	// AutodateField with OnCreate:true does not set "created" during update
	// operations, so the backfill silently failed to persist.
	//
	// This migration uses raw SQL to bypass the interceptors.
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

		nowStr := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				log.Printf("Skipping %s: %v", name, err)
				continue
			}

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
		return nil
	})
}
