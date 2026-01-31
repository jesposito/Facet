package migrations

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migrates existing data from uploads and external_media to media_library.
// - Uploads are copied with type="upload" (files are also copied to new storage location)
// - External media (non-mirrors) are copied with type="external"
// - Mirror entries (URLs containing /api/files/) are skipped
// - Original IDs are preserved to maintain existing relations
func init() {
	m.Register(func(app core.App) error {
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return err
		}

		// Get storage base path - check UPLOADS_DIR first, then fall back to pb_data/storage
		storageBase := os.Getenv("UPLOADS_DIR")
		if storageBase == "" {
			storageBase = filepath.Join(app.DataDir(), "storage")
		}

		// Migrate uploads collection
		uploadsCollection, err := app.FindCollectionByNameOrId("uploads")
		if err == nil {
			uploads, err := app.FindAllRecords(uploadsCollection)
			if err != nil {
				return err
			}

			for _, upload := range uploads {
				// Check if already migrated (same ID exists in media_library)
				if _, err := app.FindRecordById(mediaLibrary, upload.Id); err == nil {
					continue
				}

				filename := upload.GetString("file")

				// Copy file from uploads storage to media_library storage
				if filename != "" {
					srcDir := filepath.Join(storageBase, uploadsCollection.Id, upload.Id)
					dstDir := filepath.Join(storageBase, mediaLibrary.Id, upload.Id)
					srcPath := filepath.Join(srcDir, filename)

					// Check if source file exists
					if _, err := os.Stat(srcPath); err == nil {
						// Create destination directory
						if err := os.MkdirAll(dstDir, 0755); err != nil {
							app.Logger().Error("Failed to create directory for upload migration", "id", upload.Id, "error", err)
							continue
						}

						// Copy the main file
						if err := copyFile(srcPath, filepath.Join(dstDir, filename)); err != nil {
							app.Logger().Error("Failed to copy file during upload migration", "id", upload.Id, "error", err)
							continue
						}

						// Also copy any existing thumbnail (filename_thumb.webp)
						ext := filepath.Ext(filename)
						nameWithoutExt := strings.TrimSuffix(filename, ext)
						thumbFilename := nameWithoutExt + "_thumb.webp"
						thumbSrc := filepath.Join(srcDir, thumbFilename)
						if _, err := os.Stat(thumbSrc); err == nil {
							copyFile(thumbSrc, filepath.Join(dstDir, thumbFilename))
						}
					}
				}

				record := core.NewRecord(mediaLibrary)
				record.Id = upload.Id // Preserve original ID

				record.Set("type", "upload")
				record.Set("file", filename)
				record.Set("title", upload.GetString("title"))
				record.Set("mime", upload.GetString("mime"))
				record.Set("alt_text", upload.GetString("alt_text"))
				record.Set("description", upload.GetString("description"))
				record.Set("admin_tags", upload.Get("admin_tags"))
				record.Set("last_used_at", upload.Get("last_used_at"))

				// Ensure title is set (required field)
				if record.GetString("title") == "" {
					if filename != "" {
						record.Set("title", filename)
					} else {
						record.Set("title", "Untitled Upload")
					}
				}

				if err := app.Save(record); err != nil {
					// Log error but continue with other records
					app.Logger().Error("Failed to migrate upload", "id", upload.Id, "error", err)
					continue
				}
			}
		}

		// Migrate external_media collection (skip mirrors)
		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err == nil {
			externals, err := app.FindAllRecords(externalCollection)
			if err != nil {
				return err
			}

			for _, external := range externals {
				url := external.GetString("url")

				// Skip mirror entries (internal file URLs)
				if strings.Contains(url, "/api/files/") {
					continue
				}

				// Check if already migrated
				if _, err := app.FindRecordById(mediaLibrary, external.Id); err == nil {
					continue
				}

				record := core.NewRecord(mediaLibrary)
				record.Id = external.Id // Preserve original ID

				record.Set("type", "external")
				record.Set("url", url)
				record.Set("thumbnail_url", external.GetString("thumbnail_url"))
				record.Set("title", external.GetString("title"))
				record.Set("mime", external.GetString("mime"))
				record.Set("alt_text", external.GetString("alt_text"))
				record.Set("description", external.GetString("description"))
				record.Set("admin_tags", external.Get("admin_tags"))
				record.Set("last_used_at", external.Get("last_used_at"))

				// Ensure title is set (required field)
				if record.GetString("title") == "" {
					if url != "" {
						record.Set("title", url)
					} else {
						record.Set("title", "Untitled External Media")
					}
				}

				if err := app.Save(record); err != nil {
					app.Logger().Error("Failed to migrate external_media", "id", external.Id, "error", err)
					continue
				}
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete all records from media_library and their copied files
		// (The collection itself will be deleted by the previous migration's rollback)
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return nil
		}

		storageBase := os.Getenv("UPLOADS_DIR")
		if storageBase == "" {
			storageBase = filepath.Join(app.DataDir(), "storage")
		}

		records, err := app.FindAllRecords(mediaLibrary)
		if err != nil {
			return nil
		}

		for _, record := range records {
			// Remove copied files for upload types
			if record.GetString("type") == "upload" {
				if filename := record.GetString("file"); filename != "" {
					filePath := filepath.Join(storageBase, mediaLibrary.Id, record.Id, filename)
					os.Remove(filePath)
					// Try to remove the record directory if empty
					os.Remove(filepath.Join(storageBase, mediaLibrary.Id, record.Id))
				}
			}
			app.Delete(record)
		}

		// Try to remove the collection directory if empty
		os.Remove(filepath.Join(storageBase, mediaLibrary.Id))

		return nil
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
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

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
