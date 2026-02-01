package migrations

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migrates mirror entries from external_media to media_library.
// Mirror entries have URLs containing /api/files/ pointing to internal storage.
// These are converted to type="upload" records with the actual file copied.
// Original IDs are preserved to maintain existing media_refs relations.
func init() {
	m.Register(func(app core.App) error {
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return err
		}

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			// No external_media collection, nothing to migrate
			return nil
		}

		// Build list of possible storage paths to check
		// Docker containers typically mount uploads at /uploads
		// UPLOADS_DIR env var can override this
		// Fallback to app.DataDir()/storage (PocketBase default)
		var storagePaths []string
		if envDir := os.Getenv("UPLOADS_DIR"); envDir != "" {
			storagePaths = append(storagePaths, envDir)
		}
		storagePaths = append(storagePaths, "/uploads") // Docker default
		storagePaths = append(storagePaths, filepath.Join(app.DataDir(), "storage"))

		// Pattern to extract collection ID, record ID, and filename from mirror URLs
		// Format: /api/files/{collection_id}/{record_id}/{filename}
		mirrorPattern := regexp.MustCompile(`/api/files/([^/]+)/([^/]+)/(.+)$`)

		// Helper to find file in any of the storage paths
		findFile := func(collectionId, recordId, filename string) (srcPath, storageBase string, found bool) {
			for _, base := range storagePaths {
				path := filepath.Join(base, collectionId, recordId, filename)
				if _, err := os.Stat(path); err == nil {
					return path, base, true
				}
			}
			return "", "", false
		}

		externals, err := app.FindAllRecords(externalCollection)
		if err != nil {
			app.Logger().Error("Failed to find external_media records", "error", err)
			return err
		}

		app.Logger().Info("Starting mirror migration", "total_external_media", len(externals), "storage_paths", storagePaths)

		migratedCount := 0
		skippedCount := 0

		for _, external := range externals {
			url := external.GetString("url")

			// Only process mirror entries
			if !strings.Contains(url, "/api/files/") {
				skippedCount++
				continue
			}

			app.Logger().Info("Processing mirror entry", "id", external.Id, "url", url)

			// Check if already migrated
			if _, err := app.FindRecordById(mediaLibrary, external.Id); err == nil {
				app.Logger().Info("Already migrated, skipping", "id", external.Id)
				skippedCount++
				continue
			}

			// Extract file info from mirror URL
			matches := mirrorPattern.FindStringSubmatch(url)
			if len(matches) != 4 {
				app.Logger().Warn("Could not parse mirror URL", "id", external.Id, "url", url)
				continue
			}

			sourceCollectionId := matches[1]
			sourceRecordId := matches[2]
			filename := matches[3]

			// Find the source file in any storage location
			srcPath, storageBase, found := findFile(sourceCollectionId, sourceRecordId, filename)
			if !found {
				app.Logger().Warn("Mirror source file not found in any storage path",
					"id", external.Id,
					"collection", sourceCollectionId,
					"record", sourceRecordId,
					"filename", filename,
					"searched", storagePaths)
				continue
			}

			// Create destination directory in the same storage location where source was found
			dstDir := filepath.Join(storageBase, mediaLibrary.Id, external.Id)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				app.Logger().Error("Failed to create directory for mirror migration", "id", external.Id, "error", err)
				continue
			}

			// Copy the main file
			if err := copyFileMirror(srcPath, filepath.Join(dstDir, filename)); err != nil {
				app.Logger().Error("Failed to copy file during mirror migration", "id", external.Id, "error", err)
				continue
			}

			// Also copy any existing thumbnails (try common formats)
			ext := filepath.Ext(filename)
			nameWithoutExt := strings.TrimSuffix(filename, ext)
			srcDir := filepath.Join(storageBase, sourceCollectionId, sourceRecordId)
			for _, thumbExt := range []string{"_thumb.webp", "_thumb.jpg", "_thumb.png"} {
				thumbFilename := nameWithoutExt + thumbExt
				thumbSrc := filepath.Join(srcDir, thumbFilename)
				if _, err := os.Stat(thumbSrc); err == nil {
					copyFileMirror(thumbSrc, filepath.Join(dstDir, thumbFilename))
				}
			}

			// Create media_library record with the external_media ID (preserves relations)
			record := core.NewRecord(mediaLibrary)
			record.Id = external.Id // CRITICAL: Preserve original ID for media_refs

			record.Set("type", "upload")
			record.Set("file", filename)
			record.Set("title", external.GetString("title"))
			record.Set("mime", external.GetString("mime"))
			record.Set("alt_text", external.GetString("alt_text"))
			record.Set("description", external.GetString("description"))
			record.Set("admin_tags", external.Get("admin_tags"))
			record.Set("last_used_at", external.Get("last_used_at"))

			// Ensure title is set
			if record.GetString("title") == "" {
				if filename != "" {
					record.Set("title", filename)
				} else {
					record.Set("title", "Untitled Upload")
				}
			}

			if err := app.Save(record); err != nil {
				app.Logger().Error("Failed to migrate mirror entry", "id", external.Id, "error", err)
				// Clean up copied file on failure
				os.RemoveAll(dstDir)
				continue
			}

			app.Logger().Info("Migrated mirror entry", "id", external.Id, "filename", filename)
			migratedCount++
		}

		app.Logger().Info("Mirror migration complete", "migrated", migratedCount, "skipped", skippedCount)
		return nil
	}, func(app core.App) error {
		// Rollback: delete migrated mirror entries from media_library
		// We only delete entries that originated from mirrors (have file but came from external_media)
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return nil
		}

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			return nil
		}

		// Check all possible storage locations
		var storagePaths []string
		if envDir := os.Getenv("UPLOADS_DIR"); envDir != "" {
			storagePaths = append(storagePaths, envDir)
		}
		storagePaths = append(storagePaths, "/uploads")
		storagePaths = append(storagePaths, filepath.Join(app.DataDir(), "storage"))

		externals, err := app.FindAllRecords(externalCollection)
		if err != nil {
			return nil
		}

		for _, external := range externals {
			url := external.GetString("url")
			if !strings.Contains(url, "/api/files/") {
				continue
			}

			// Find and delete the corresponding media_library record
			record, err := app.FindRecordById(mediaLibrary, external.Id)
			if err != nil {
				continue
			}

			// Remove copied files from all possible locations
			if filename := record.GetString("file"); filename != "" {
				for _, base := range storagePaths {
					recordDir := filepath.Join(base, mediaLibrary.Id, record.Id)
					os.RemoveAll(recordDir)
				}
			}

			app.Delete(record)
		}

		return nil
	})
}

// copyFileMirror copies a file from src to dst (same as copyFile but with unique name to avoid redeclaration)
func copyFileMirror(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
