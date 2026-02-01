package migrations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// copyMirrorFile copies a file from src to dst
func copyMirrorFile(src, dst string) error {
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

// migrationLog writes to stderr AND a persistent file in /data for visibility
func migrationLog(dataDir, msg string) {
	// Always write to stderr (more reliable in Docker than stdout)
	fmt.Fprintln(os.Stderr, "[MIRROR-MIGRATION]", msg)

	// Also write to persistent file in data directory
	logPath := filepath.Join(dataDir, "mirror_migration.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil && f != nil {
		f.WriteString(msg + "\n")
		f.Close()
	}
}

// Migrates mirror entries from external_media to media_library.
// Mirror entries have URLs containing /api/files/ pointing to internal storage.
// These are converted to type="upload" records with the actual file copied.
// Original IDs are preserved to maintain existing media_refs relations.
func init() {
	m.Register(func(app core.App) error {
		dataDir := app.DataDir()
		log := func(msg string) { migrationLog(dataDir, msg) }

		log("=== MIRROR MIGRATION STARTING ===")
		log(fmt.Sprintf("DataDir: %s", dataDir))

		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			log(fmt.Sprintf("ERROR: media_library not found: %v", err))
			return err
		}
		log(fmt.Sprintf("Found media_library: %s", mediaLibrary.Id))

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			log(fmt.Sprintf("No external_media collection, skipping migration: %v", err))
			return nil
		}
		log(fmt.Sprintf("Found external_media: %s", externalCollection.Id))

		// Build list of possible storage paths to check
		// Docker containers typically mount uploads at /uploads
		// UPLOADS_DIR env var can override this
		// Fallback to app.DataDir()/storage (PocketBase default)
		var storagePaths []string
		if envDir := os.Getenv("UPLOADS_DIR"); envDir != "" {
			log(fmt.Sprintf("UPLOADS_DIR env set: %s", envDir))
			storagePaths = append(storagePaths, envDir)
		}
		storagePaths = append(storagePaths, "/uploads") // Docker default
		storagePaths = append(storagePaths, filepath.Join(dataDir, "storage"))

		// Log which paths exist
		for _, p := range storagePaths {
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				log(fmt.Sprintf("Storage path EXISTS: %s", p))
			} else {
				log(fmt.Sprintf("Storage path missing: %s", p))
			}
		}

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
			log(fmt.Sprintf("ERROR finding records: %v", err))
			return err
		}

		log(fmt.Sprintf("Found %d external_media records total", len(externals)))

		migratedCount := 0
		skippedCount := 0
		mirrorCount := 0

		// First pass: count mirrors
		for _, external := range externals {
			url := external.GetString("url")
			if strings.Contains(url, "/api/files/") {
				mirrorCount++
			}
		}
		log(fmt.Sprintf("Found %d mirror entries to process", mirrorCount))

		for _, external := range externals {
			url := external.GetString("url")

			// Only process mirror entries
			if !strings.Contains(url, "/api/files/") {
				skippedCount++
				continue
			}

			log(fmt.Sprintf("Processing mirror: %s -> %s", external.Id, url))

			// Check if already migrated
			if _, err := app.FindRecordById(mediaLibrary, external.Id); err == nil {
				log(fmt.Sprintf("Already migrated, skipping: %s", external.Id))
				skippedCount++
				continue
			}

			// Extract file info from mirror URL
			matches := mirrorPattern.FindStringSubmatch(url)
			if len(matches) != 4 {
				log(fmt.Sprintf("ERROR: Could not parse URL '%s' with pattern", url))
				continue
			}

			sourceCollectionId := matches[1]
			sourceRecordId := matches[2]
			filename := matches[3]
			log(fmt.Sprintf("Parsed: collection=%s record=%s file=%s", sourceCollectionId, sourceRecordId, filename))

			// Find the source file in any storage location
			srcPath, storageBase, found := findFile(sourceCollectionId, sourceRecordId, filename)
			if !found {
				log(fmt.Sprintf("ERROR: File not found for %s - tried: %v", external.Id, storagePaths))
				log(fmt.Sprintf("  Looking for: %s/%s/%s", sourceCollectionId, sourceRecordId, filename))
				continue
			}
			log(fmt.Sprintf("Found source file: %s", srcPath))

			// Copy file to the media_library storage location (same base as source)
			dstDir := filepath.Join(storageBase, mediaLibrary.Id, external.Id)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				log(fmt.Sprintf("ERROR: Failed to create dir %s: %v", dstDir, err))
				continue
			}

			dstPath := filepath.Join(dstDir, filename)
			if err := copyMirrorFile(srcPath, dstPath); err != nil {
				log(fmt.Sprintf("ERROR: Failed to copy file: %v", err))
				continue
			}
			log(fmt.Sprintf("Copied to: %s", dstPath))

			// Get title
			title := external.GetString("title")
			if title == "" {
				title = filename
			}
			if title == "" {
				title = "Untitled Upload"
			}

			// Use raw SQL to insert the record, bypassing PocketBase's file validation
			// PocketBase stores files as just the filename in the database
			_, err = app.DB().NewQuery(`
				INSERT INTO ` + mediaLibrary.Name + ` (id, type, file, title, mime, alt_text, description, created, updated)
				VALUES ({:id}, 'upload', {:file}, {:title}, {:mime}, {:alt_text}, {:description}, {:created}, {:updated})
			`).Bind(map[string]any{
				"id":          external.Id,
				"file":        filename,
				"title":       title,
				"mime":        external.GetString("mime"),
				"alt_text":    external.GetString("alt_text"),
				"description": external.GetString("description"),
				"created":     external.GetRaw("created"),
				"updated":     external.GetRaw("updated"),
			}).Execute()

			if err != nil {
				log(fmt.Sprintf("ERROR: Failed to insert record %s: %v", external.Id, err))
				os.RemoveAll(dstDir)
				continue
			}

			log(fmt.Sprintf("SUCCESS: Migrated %s -> media_library", external.Id))
			migratedCount++
		}

		log(fmt.Sprintf("=== MIGRATION COMPLETE: migrated=%d, skipped=%d ===", migratedCount, skippedCount))
		return nil
	}, func(app core.App) error {
		// Rollback: delete migrated mirror entries from media_library
		// PocketBase handles file cleanup automatically when records are deleted
		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			return nil
		}

		externalCollection, err := app.FindCollectionByNameOrId("external_media")
		if err != nil {
			return nil
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

			app.Delete(record)
		}

		return nil
	})
}
