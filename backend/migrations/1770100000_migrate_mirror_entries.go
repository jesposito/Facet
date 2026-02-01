package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

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
			srcPath, _, found := findFile(sourceCollectionId, sourceRecordId, filename)
			if !found {
				log(fmt.Sprintf("ERROR: File not found for %s - tried: %v", external.Id, storagePaths))
				log(fmt.Sprintf("  Looking for: %s/%s/%s", sourceCollectionId, sourceRecordId, filename))
				continue
			}
			log(fmt.Sprintf("Found source file: %s", srcPath))

			// Create a proper file object from the source file
			// PocketBase requires files to be set via its filesystem API, not by copying
			file, err := filesystem.NewFileFromPath(srcPath)
			if err != nil {
				log(fmt.Sprintf("ERROR: Failed to create file object from %s: %v", srcPath, err))
				continue
			}
			// Preserve original filename
			file.OriginalName = filename

			// Create media_library record with the external_media ID (preserves relations)
			record := core.NewRecord(mediaLibrary)
			record.Id = external.Id // CRITICAL: Preserve original ID for media_refs

			record.Set("type", "upload")
			record.Set("file", file)
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
				log(fmt.Sprintf("ERROR: Failed to save record %s: %v", external.Id, err))
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
