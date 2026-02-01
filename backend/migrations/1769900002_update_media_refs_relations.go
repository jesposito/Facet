package migrations

import (
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Updates media_refs relation fields to point to media_library instead of external_media.
// Also replaces any mirror entry IDs in media_refs with the corresponding upload IDs.
// Mirror entries (external_media records with URLs like /api/files/...) are not migrated
// to media_library, so we need to resolve them to the actual upload IDs.
func init() {
	m.Register(func(app core.App) error {
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return err
		}

		// Build a mapping of mirror IDs to upload IDs
		// Mirror entries in external_media have URLs like: /api/files/{collection_id}/{record_id}/{filename}
		mirrorToUpload := make(map[string]string)
		externalMedia, err := app.FindCollectionByNameOrId("external_media")
		if err == nil {
			externals, err := app.FindAllRecords(externalMedia.Name)
			if err == nil {
				// Regex to extract record_id from mirror URLs
				// Format: /api/files/{collection_id}/{record_id}/{filename}
				mirrorURLPattern := regexp.MustCompile(`/api/files/[^/]+/([^/]+)/`)

				for _, ext := range externals {
					url := ext.GetString("url")
					if strings.Contains(url, "/api/files/") {
						// This is a mirror entry - extract the upload record ID
						matches := mirrorURLPattern.FindStringSubmatch(url)
						if len(matches) >= 2 {
							uploadID := matches[1]
							mirrorToUpload[ext.Id] = uploadID
							app.Logger().Debug("Mirror mapping", "mirror", ext.Id, "upload", uploadID)
						}
					}
				}
			}
		}
		app.Logger().Info("Built mirror-to-upload mapping", "count", len(mirrorToUpload))

		// Collections that have media_refs fields
		collectionsWithMediaRefs := []string{
			"posts",
			"projects",
			"talks",
			"custom_content",
		}

		for _, collName := range collectionsWithMediaRefs {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				// Collection doesn't exist - skip
				continue
			}

			field := collection.Fields.GetByName("media_refs")
			if field == nil {
				continue
			}

			// Get the existing relation field
			relationField, ok := field.(*core.RelationField)
			if !ok {
				continue
			}

			// First, update existing records to replace mirror IDs with upload IDs
			if len(mirrorToUpload) > 0 {
				records, err := app.FindAllRecords(collName)
				if err == nil {
					for _, record := range records {
						mediaRefs := record.GetStringSlice("media_refs")
						if len(mediaRefs) == 0 {
							continue
						}

						updated := false
						newRefs := make([]string, 0, len(mediaRefs))
						for _, refID := range mediaRefs {
							if uploadID, isMirror := mirrorToUpload[refID]; isMirror {
								// Replace mirror ID with upload ID
								newRefs = append(newRefs, uploadID)
								updated = true
								app.Logger().Debug("Replacing mirror ref", "collection", collName, "record", record.Id, "old", refID, "new", uploadID)
							} else {
								newRefs = append(newRefs, refID)
							}
						}

						if updated {
							record.Set("media_refs", newRefs)
							if err := app.Save(record); err != nil {
								app.Logger().Error("Failed to update media_refs values", "collection", collName, "record", record.Id, "error", err)
							}
						}
					}
				}
			}

			// Update relation field to point to media_library
			relationField.CollectionId = mediaLibrary.Id

			if err := app.Save(collection); err != nil {
				app.Logger().Error("Failed to update media_refs relation", "collection", collName, "error", err)
				continue
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: point media_refs back to external_media
		externalMedia, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// external_media doesn't exist, can't rollback
			return nil
		}

		collectionsWithMediaRefs := []string{
			"posts",
			"projects",
			"talks",
			"custom_content",
		}

		for _, collName := range collectionsWithMediaRefs {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			field := collection.Fields.GetByName("media_refs")
			if field == nil {
				continue
			}

			relationField, ok := field.(*core.RelationField)
			if !ok {
				continue
			}

			relationField.CollectionId = externalMedia.Id
			app.Save(collection)
		}

		return nil
	})
}
