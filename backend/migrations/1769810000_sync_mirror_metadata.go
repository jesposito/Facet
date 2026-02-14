package migrations

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Syncs metadata (title, alt_text, description) from uploads records to their
// external_media mirror records. Mirrors are created by the MultiMediaPicker when
// attaching uploads to content via media_refs, and initially receive the filename
// as their title. If the user later sets a meaningful title on the upload in the
// media library, the mirror was not updated. This migration backfills existing mirrors.
func init() {
	m.Register(func(app core.App) error {
		extCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// Collection doesn't exist yet — nothing to migrate
			return nil
		}

		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err != nil {
			return nil
		}

		// Find all external_media records that are mirrors (URL points to /api/files/)
		mirrors, err := app.FindAllRecords(extCollection.Name)
		if err != nil {
			return nil
		}

		synced := 0
		for _, mirror := range mirrors {
			mirrorURL := mirror.GetString("url")
			if !strings.Contains(mirrorURL, "/api/files/") {
				continue
			}

			// Extract the uploads record ID from the URL pattern:
			// .../api/files/{collectionId}/{recordId}/{filename}
			parts := strings.Split(mirrorURL, "/api/files/")
			if len(parts) < 2 {
				continue
			}
			segments := strings.Split(strings.Trim(parts[1], "/"), "/")
			if len(segments) < 3 {
				continue
			}
			// segments[0] = collectionId, segments[1] = recordId, segments[2] = filename
			// Only process mirrors pointing to the uploads collection
			if segments[0] != uploadsCollection.Id {
				continue
			}
			recordID := segments[1]

			upload, err := app.FindRecordById("uploads", recordID)
			if err != nil {
				continue
			}

			changed := false
			if title := upload.GetString("title"); title != "" && mirror.GetString("title") != title {
				mirror.Set("title", title)
				changed = true
			}
			if altText := upload.GetString("alt_text"); altText != "" && mirror.GetString("alt_text") != altText {
				mirror.Set("alt_text", altText)
				changed = true
			}
			if desc := upload.GetString("description"); desc != "" && mirror.GetString("description") != desc {
				mirror.Set("description", desc)
				changed = true
			}

			if changed {
				if err := app.Save(mirror); err != nil {
					fmt.Printf("sync_mirror_metadata: failed to update mirror %s: %v\n", mirror.Id, err)
				} else {
					synced++
				}
			}
		}

		if synced > 0 {
			fmt.Printf("sync_mirror_metadata: synced %d mirror record(s)\n", synced)
		}

		return nil
	}, func(app core.App) error {
		// No rollback — this is a best-effort data sync
		return nil
	})
}
