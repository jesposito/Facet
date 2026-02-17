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
			// Use the record's updated timestamp if available, otherwise use now.
			records, err := app.FindAllRecords(name)
			if err != nil {
				continue
			}
			now := time.Now()
			backfilled := 0
			for _, record := range records {
				created := record.GetDateTime("created")
				if !created.IsZero() {
					continue
				}
				// Use updated if available, otherwise fall back to current time
				updated := record.GetDateTime("updated")
				if !updated.IsZero() {
					record.Set("created", updated)
				} else {
					record.Set("created", now)
				}
				if err := app.Save(record); err != nil {
					log.Printf("Failed to backfill %s/%s: %v", name, record.Id, err)
				} else {
					backfilled++
				}
			}
			if backfilled > 0 {
				log.Printf("Backfilled %d records in %s with missing created timestamp", backfilled, name)
			}
		}

		return nil
	}, func(app core.App) error {
		// No rollback — autodate fields are expected on all collections
		return nil
	})
}
