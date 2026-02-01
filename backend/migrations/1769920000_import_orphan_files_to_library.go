package migrations

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Scans storage for ALL files and creates media_library records for any that don't exist.
// This ensures every file in storage is accessible via media_library for MediaPicker.
func init() {
	m.Register(func(app core.App) error {
		app.Logger().Info("=== ORPHAN FILE IMPORT MIGRATION STARTING ===")

		mediaLibrary, err := app.FindCollectionByNameOrId("media_library")
		if err != nil {
			app.Logger().Error("media_library collection not found", "error", err)
			return nil // Don't fail migration if collection doesn't exist
		}
		app.Logger().Info("Found media_library collection", "id", mediaLibrary.Id)

		storageBase := os.Getenv("UPLOADS_DIR")
		if storageBase == "" {
			storageBase = filepath.Join(app.DataDir(), "storage")
		}
		app.Logger().Info("Using storage base", "path", storageBase)

		// Track what we've created
		created := 0
		skipped := 0

		// Walk all collection directories
		entries, err := os.ReadDir(storageBase)
		if err != nil {
			app.Logger().Error("Failed to read storage directory", "path", storageBase, "error", err)
			return nil
		}
		app.Logger().Info("Found storage entries", "count", len(entries))

		for _, collEntry := range entries {
			if !collEntry.IsDir() || !strings.HasPrefix(collEntry.Name(), "pbc_") {
				continue
			}

			collectionID := collEntry.Name()
			collectionPath := filepath.Join(storageBase, collectionID)

			// Skip media_library's own storage - those records should already exist
			if collectionID == mediaLibrary.Id {
				app.Logger().Debug("Skipping media_library storage", "id", collectionID)
				continue
			}

			app.Logger().Info("Processing collection storage", "collectionID", collectionID)

			// Walk record directories within this collection
			recordEntries, err := os.ReadDir(collectionPath)
			if err != nil {
				app.Logger().Error("Failed to read collection directory", "path", collectionPath, "error", err)
				continue
			}
			app.Logger().Info("Found record entries", "collectionID", collectionID, "count", len(recordEntries))

			for _, recEntry := range recordEntries {
				if !recEntry.IsDir() {
					continue
				}

				recordID := recEntry.Name()
				recordPath := filepath.Join(collectionPath, recordID)

				// Find actual media files (not .attrs, not thumbnails)
				files, err := os.ReadDir(recordPath)
				if err != nil {
					continue
				}

				for _, fileEntry := range files {
					if fileEntry.IsDir() {
						continue
					}

					filename := fileEntry.Name()

					// Skip metadata and thumbnail files
					if strings.HasSuffix(filename, ".attrs") ||
						strings.Contains(filename, "_thumb.") {
						continue
					}

					// Generate a unique ID for this file
					// Use collection + record + hash of filename to ensure uniqueness
					mediaID := generateUniqueMediaID(collectionID, recordID, filename)

					// Check if already exists in media_library
					if _, err := app.FindRecordById(mediaLibrary, mediaID); err == nil {
						skipped++
						continue
					}

					// Determine mime type
					mime := guessMimeFromFilename(filename)

					// Build a title from the filename
					title := cleanFilenameForTitle(filename)

					// Create media_library record
					record := core.NewRecord(mediaLibrary)
					record.Id = mediaID
					record.Set("type", "upload")
					record.Set("file", filename)
					record.Set("title", title)
					record.Set("mime", mime)

					// Copy the file to media_library storage
					srcPath := filepath.Join(recordPath, filename)
					dstDir := filepath.Join(storageBase, mediaLibrary.Id, mediaID)
					dstPath := filepath.Join(dstDir, filename)

					if err := os.MkdirAll(dstDir, 0755); err != nil {
						app.Logger().Error("Failed to create media_library dir", "path", dstDir, "error", err)
						continue
					}

					// Copy main file
					if err := copyFileSimple(srcPath, dstPath); err != nil {
						app.Logger().Error("Failed to copy file", "src", srcPath, "dst", dstPath, "error", err)
						continue
					}

					// Also copy thumbnail if it exists
					thumbName := buildThumbName(filename)
					thumbSrc := filepath.Join(recordPath, thumbName)
					if _, err := os.Stat(thumbSrc); err == nil {
						thumbDst := filepath.Join(dstDir, thumbName)
						copyFileSimple(thumbSrc, thumbDst)
					}

					// Save the record
					if err := app.Save(record); err != nil {
						app.Logger().Error("Failed to save media_library record", "id", mediaID, "error", err)
						// Clean up copied file
						os.RemoveAll(dstDir)
						continue
					}

					created++
					app.Logger().Debug("Imported orphan file", "id", mediaID, "file", filename, "source", collectionID+"/"+recordID)
				}
			}
		}

		app.Logger().Info("Orphan file import complete", "created", created, "skipped", skipped)
		return nil
	}, func(app core.App) error {
		// Rollback: We can't easily distinguish imported records from originals
		// Just log that rollback was requested
		app.Logger().Warn("Rollback requested for orphan file import - manual cleanup may be needed")
		return nil
	})
}

func generateUniqueMediaID(collectionID, recordID, filename string) string {
	// Create a deterministic but unique ID
	// Take parts of each component to create a 15-char ID (PocketBase standard)
	// Format: 4 chars from collection + 4 chars from record + 7 chars from filename hash

	collPart := collectionID
	if len(collPart) > 4 {
		collPart = collPart[len(collPart)-4:] // Take last 4 chars (more unique than first 4)
	}

	recPart := recordID
	if len(recPart) > 4 {
		recPart = recPart[:4]
	}

	// Simple hash of filename to get remaining chars
	fileHash := simpleHash(filename)

	return collPart + recPart + fileHash
}

func simpleHash(s string) string {
	// Simple deterministic hash to create a 7-char string
	var h uint64 = 0
	for _, c := range s {
		h = h*31 + uint64(c)
	}

	// Convert to base36 for compact representation
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, 7)
	for i := 6; i >= 0; i-- {
		result[i] = chars[h%36]
		h /= 36
	}
	return string(result)
}

func guessMimeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".heic": "image/heic",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
	}
	if m, ok := mimes[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

func cleanFilenameForTitle(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove common suffixes like _abc123def
	if idx := strings.LastIndex(name, "_"); idx > 0 && len(name)-idx <= 12 {
		name = name[:idx]
	}

	// Replace underscores with spaces
	name = strings.ReplaceAll(name, "_", " ")

	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(string(name[0])) + name[1:]
	}

	return name
}

func buildThumbName(filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	return base + "_thumb.jpg"
}

func copyFileSimple(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
