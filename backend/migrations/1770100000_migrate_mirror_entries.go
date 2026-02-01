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

		// Get storage base path
		storageBase := os.Getenv("UPLOADS_DIR")
		if storageBase == "" {
			storageBase = filepath.Join(app.DataDir(), "storage")
		}

		// Pattern to extract collection ID, record ID, and filename from mirror URLs
		// Format: /api/files/{collection_id}/{record_id}/{filename}
		mirrorPattern := regexp.MustCompile(`/api/files/([^/]+)/([^/]+)/(.+)$`)

		externals, err := app.FindAllRecords(externalCollection)
		if err != nil {
			return err
		}

		for _, external := range externals {
			url := external.GetString("url")

			// Only process mirror entries
			if !strings.Contains(url, "/api/files/") {
				continue
			}

			// Check if already migrated
			if _, err := app.FindRecordById(mediaLibrary, external.Id); err == nil {
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

			// Find the source file
			srcPath := filepath.Join(storageBase, sourceCollectionId, sourceRecordId, filename)
			if _, err := os.Stat(srcPath); err != nil {
				app.Logger().Warn("Mirror source file not found", "id", external.Id, "path", srcPath)
				continue
			}

			// Create destination directory (using external_media ID to preserve relations)
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

			// Also copy any existing thumbnail
			ext := filepath.Ext(filename)
			nameWithoutExt := strings.TrimSuffix(filename, ext)
			thumbFilename := nameWithoutExt + "_thumb.webp"
			thumbSrc := filepath.Join(storageBase, sourceCollectionId, sourceRecordId, thumbFilename)
			if _, err := os.Stat(thumbSrc); err == nil {
				copyFileMirror(thumbSrc, filepath.Join(dstDir, thumbFilename))
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
		}

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

		storageBase := os.Getenv("UPLOADS_DIR")
		if storageBase == "" {
			storageBase = filepath.Join(app.DataDir(), "storage")
		}

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

			// Remove copied files
			if filename := record.GetString("file"); filename != "" {
				recordDir := filepath.Join(storageBase, mediaLibrary.Id, record.Id)
				os.RemoveAll(recordDir)
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
