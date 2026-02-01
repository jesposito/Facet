package migrations

import (
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Repairs media_refs that were incorrectly changed by the old version of
// 1769900002_update_media_refs_relations.go. That migration replaced mirror IDs
// with upload IDs, but the mirror migration (1770100000) creates records with
// mirror IDs. This migration detects and fixes the mismatch.
func init() {
	m.Register(func(app core.App) error {
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			// media_library doesn't exist yet, nothing to repair
			return nil
		}

		externalMedia, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// No external_media means no mirrors to worry about
			return nil
		}

		// Build reverse mapping: upload ID -> mirror ID
		// This lets us detect if media_refs were incorrectly replaced
		uploadToMirror := make(map[string]string)
		mirrorURLPattern := regexp.MustCompile(`/api/files/[^/]+/([^/]+)/`)

		externals, err := app.FindAllRecords(externalMedia.Name)
		if err != nil {
			return nil
		}

		for _, ext := range externals {
			url := ext.GetString("url")
			if strings.Contains(url, "/api/files/") {
				matches := mirrorURLPattern.FindStringSubmatch(url)
				if len(matches) >= 2 {
					uploadID := matches[1]
					uploadToMirror[uploadID] = ext.Id
					app.Logger().Debug("Repair mapping", "upload", uploadID, "mirror", ext.Id)
				}
			}
		}

		if len(uploadToMirror) == 0 {
			app.Logger().Info("No mirror mappings found, nothing to repair")
			return nil
		}

		app.Logger().Info("Built upload-to-mirror mapping for repair", "count", len(uploadToMirror))

		// Collections that have media_refs fields
		collectionsWithMediaRefs := []string{
			"posts",
			"projects",
			"talks",
			"custom_content",
		}

		repairedCount := 0

		for _, collName := range collectionsWithMediaRefs {
			records, err := app.FindAllRecords(collName)
			if err != nil {
				continue
			}

			for _, record := range records {
				mediaRefs := record.GetStringSlice("media_refs")
				if len(mediaRefs) == 0 {
					continue
				}

				repaired := false
				newRefs := make([]string, 0, len(mediaRefs))

				for _, refID := range mediaRefs {
					// Check if this ID exists in media_library
					if _, err := app.FindRecordById(mediaLibrary, refID); err == nil {
						// ID exists in media_library, keep it
						newRefs = append(newRefs, refID)
						continue
					}

					// ID doesn't exist in media_library - check if it's an upload ID
					// that should be a mirror ID
					if mirrorID, wasMisConverted := uploadToMirror[refID]; wasMisConverted {
						// Check if the mirror ID exists in media_library
						if _, err := app.FindRecordById(mediaLibrary, mirrorID); err == nil {
							// Yes! This was incorrectly converted, fix it
							newRefs = append(newRefs, mirrorID)
							repaired = true
							app.Logger().Info("Repaired media_ref",
								"collection", collName,
								"record", record.Id,
								"was", refID,
								"now", mirrorID)
							continue
						}
					}

					// Keep the original ID (might be valid external media)
					newRefs = append(newRefs, refID)
				}

				if repaired {
					record.Set("media_refs", newRefs)
					if err := app.Save(record); err != nil {
						app.Logger().Error("Failed to repair media_refs",
							"collection", collName,
							"record", record.Id,
							"error", err)
					} else {
						repairedCount++
					}
				}
			}
		}

		app.Logger().Info("Media refs repair complete", "repaired", repairedCount)
		return nil
	}, func(app core.App) error {
		// No rollback needed - this is a repair migration
		return nil
	})
}
