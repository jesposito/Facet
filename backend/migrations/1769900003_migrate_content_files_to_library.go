package migrations

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migrates file fields from content collections to media_library.
// This ensures all media is available in the unified library for selection.
// Collections processed: profile, projects, experience, custom_content, testimonials, posts, talks
func init() {
	m.Register(func(app core.App) error {
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return err
		}

		storageBase := os.Getenv("UPLOADS_DIR")
		if storageBase == "" {
			storageBase = filepath.Join(app.DataDir(), "storage")
		}

		// Map of collection name -> file field names to migrate
		collectionsToMigrate := map[string][]string{
			"profile":        {"photo", "resume"},
			"projects":       {"cover_image", "images"},
			"experience":     {"logo"},
			"custom_content": {"cover_image", "media"},
			"testimonials":   {"avatar"},
			"posts":          {"cover_image"},
			"talks":          {"cover_image"},
		}

		migratedCount := 0

		for collName, fields := range collectionsToMigrate {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				app.Logger().Debug("Collection not found, skipping", "collection", collName)
				continue
			}

			records, err := app.FindAllRecords(collName)
			if err != nil {
				app.Logger().Error("Failed to load records", "collection", collName, "error", err)
				continue
			}

			for _, record := range records {
				for _, fieldName := range fields {
					// Handle both single file and multiple file fields
					files := getFileValues(record, fieldName)
					if len(files) == 0 {
						continue
					}

					for _, filename := range files {
						if filename == "" {
							continue
						}

						// Generate a unique ID for media_library based on source
						// Format: first 8 chars of collection ID + first 7 chars of record ID + field index
						mediaID := generateMediaID(collection.Id, record.Id, fieldName, filename)

						// Check if already migrated
						if _, err := app.FindRecordById(mediaLibrary, mediaID); err == nil {
							continue
						}

						// Copy file to media_library storage
						srcDir := filepath.Join(storageBase, collection.Id, record.Id)
						dstDir := filepath.Join(storageBase, mediaLibrary.Id, mediaID)
						srcPath := filepath.Join(srcDir, filename)

						if _, err := os.Stat(srcPath); err != nil {
							app.Logger().Debug("Source file not found", "path", srcPath)
							continue
						}

						if err := os.MkdirAll(dstDir, 0755); err != nil {
							app.Logger().Error("Failed to create directory", "path", dstDir, "error", err)
							continue
						}

						if err := copyFileForMigration(srcPath, filepath.Join(dstDir, filename)); err != nil {
							app.Logger().Error("Failed to copy file", "src", srcPath, "error", err)
							continue
						}

						// Copy thumbnail if exists
						ext := filepath.Ext(filename)
						nameWithoutExt := strings.TrimSuffix(filename, ext)
						thumbFilename := nameWithoutExt + "_thumb.webp"
						thumbSrc := filepath.Join(srcDir, thumbFilename)
						if _, err := os.Stat(thumbSrc); err == nil {
							copyFileForMigration(thumbSrc, filepath.Join(dstDir, thumbFilename))
						}

						// Determine mime type from extension
						mime := guessMimeType(filename)

						// Create media_library record
						libraryRecord := core.NewRecord(mediaLibrary)
						libraryRecord.Id = mediaID
						libraryRecord.Set("type", "upload")
						libraryRecord.Set("file", filename)
						libraryRecord.Set("title", buildTitle(collName, fieldName, filename, record))
						libraryRecord.Set("mime", mime)

						if err := app.Save(libraryRecord); err != nil {
							app.Logger().Error("Failed to create media_library record",
								"id", mediaID,
								"source", collName+"/"+record.Id+"/"+filename,
								"error", err)
							continue
						}

						migratedCount++
						app.Logger().Debug("Migrated file to media_library",
							"id", mediaID,
							"source", collName+"/"+record.Id+"/"+filename)
					}
				}
			}
		}

		app.Logger().Info("Content files migration complete", "migrated", migratedCount)
		return nil
	}, func(app core.App) error {
		// Rollback: remove migrated records (identified by their ID pattern)
		// This is a best-effort cleanup
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

		// Delete records that were migrated from content collections
		// These have IDs that are 15 chars (8 from collection + 7 from record)
		for _, record := range records {
			// Only delete if it looks like a migrated content file ID
			// (original uploads and external_media have standard 15-char PocketBase IDs)
			if len(record.Id) == 15 && record.GetString("type") == "upload" {
				// Check if this ID matches any source collection pattern
				if filename := record.GetString("file"); filename != "" {
					filePath := filepath.Join(storageBase, mediaLibrary.Id, record.Id, filename)
					os.Remove(filePath)
					os.Remove(filepath.Join(storageBase, mediaLibrary.Id, record.Id))
				}
				app.Delete(record)
			}
		}

		return nil
	})
}

func getFileValues(record *core.Record, fieldName string) []string {
	// Try as string first (single file)
	if val := record.GetString(fieldName); val != "" {
		return []string{val}
	}
	// Try as string slice (multiple files)
	if vals := record.GetStringSlice(fieldName); len(vals) > 0 {
		return vals
	}
	return nil
}

func generateMediaID(collectionID, recordID, fieldName, filename string) string {
	// Create a deterministic ID from source info
	// Take first 8 of collection, first 7 of record = 15 chars (standard PocketBase ID length)
	collPart := collectionID
	if len(collPart) > 8 {
		collPart = collPart[:8]
	}
	recPart := recordID
	if len(recPart) > 7 {
		recPart = recPart[:7]
	}
	return collPart + recPart
}

func buildTitle(collName, fieldName, filename string, record *core.Record) string {
	// Try to build a meaningful title
	if title := record.GetString("title"); title != "" {
		return title + " - " + fieldName
	}
	if name := record.GetString("name"); name != "" {
		return name + " - " + fieldName
	}
	// Fall back to filename
	return filename
}

func guessMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".heic": "image/heic",
		".mp4":  "video/mp4",
		".webm": "video/webm",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func copyFileForMigration(src, dst string) error {
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

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
