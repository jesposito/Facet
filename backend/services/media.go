package services

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// MediaUsageItem represents a single content item that references a media item.
type MediaUsageItem struct {
	Collection string `json:"collection"`
	RecordID   string `json:"record_id"`
	Title      string `json:"title"`
	Slug       string `json:"slug,omitempty"`
}

// MediaUsage represents usage information for a media item.
type MediaUsage struct {
	UsageCount int              `json:"usage_count"`
	UsedBy     []MediaUsageItem `json:"used_by"`
}

// FindMediaUsage queries all collections to find which content references the given external_media ID.
// It checks both media_refs relations AND *_library_url fields that may contain the media's URL.
func FindMediaUsage(app *pocketbase.PocketBase, externalMediaID string) (MediaUsage, error) {
	usage := MediaUsage{
		UsageCount: 0,
		UsedBy:     []MediaUsageItem{},
	}

	// Track already-added records to avoid duplicates
	seen := make(map[string]bool)

	// Collections that have media_refs relation to external_media
	collectionsWithMediaRefs := []string{"posts", "projects", "talks", "custom_content"}

	for _, collName := range collectionsWithMediaRefs {
		collection, err := app.FindCollectionByNameOrId(collName)
		if err != nil {
			continue
		}

		// Check if collection has media_refs field
		if collection.Fields.GetByName("media_refs") == nil {
			continue
		}

		// Find records that reference this external_media ID
		// PocketBase multi-relation: expand relation and check if any ID matches
		filter := fmt.Sprintf("media_refs.id ?= '%s'", externalMediaID)
		records, err := app.FindRecordsByFilter(collName, filter, "", 100, 0, nil)
		if err != nil {
			continue
		}

		for _, record := range records {
			key := collName + ":" + record.Id
			if seen[key] {
				continue
			}
			seen[key] = true

			title := record.GetString("title")
			if title == "" {
				title = record.GetString("name")
			}
			if title == "" {
				title = record.Id
			}

			item := MediaUsageItem{
				Collection: collName,
				RecordID:   record.Id,
				Title:      title,
				Slug:       record.GetString("slug"),
			}
			usage.UsedBy = append(usage.UsedBy, item)
			usage.UsageCount++
		}
	}

	// Also check *_library_url fields that might reference this external media's URL
	// First, get the external_media record to find its URL
	extRecord, err := app.FindRecordById("external_media", externalMediaID)
	if err == nil {
		mediaURL := extRecord.GetString("url")
		if mediaURL != "" {
			// Collections and their library URL fields
			libraryURLFields := map[string][]string{
				"experience":     {"company_logo_library_url"},
				"education":      {"institution_logo_library_url"},
				"projects":       {"cover_image_library_url"},
				"posts":          {"cover_image_library_url"},
				"talks":          {"cover_image_library_url"},
				"profile":        {"hero_image_library_url", "avatar_library_url"},
				"views":          {"hero_image_library_url"},
				"site_settings":  {"favicon_library_url"},
				"custom_content": {"cover_image_library_url"},
				"testimonials":   {"author_photo_library_url"},
				"certifications": {"badge_image_library_url"},
			}

			for collName, fields := range libraryURLFields {
				collection, err := app.FindCollectionByNameOrId(collName)
				if err != nil {
					continue
				}

				for _, fieldName := range fields {
					// Check if field exists
					if collection.Fields.GetByName(fieldName) == nil {
						continue
					}

					// Find records where this field matches our URL
					filter := fmt.Sprintf("%s = '%s'", fieldName, mediaURL)
					records, err := app.FindRecordsByFilter(collName, filter, "", 100, 0, nil)
					if err != nil {
						continue
					}

					for _, record := range records {
						key := collName + ":" + record.Id
						if seen[key] {
							continue
						}
						seen[key] = true

						title := record.GetString("title")
						if title == "" {
							title = record.GetString("name")
						}
						if title == "" {
							title = record.GetString("company")
						}
						if title == "" {
							title = record.GetString("institution")
						}
						if title == "" {
							title = record.Id
						}

						usage.UsedBy = append(usage.UsedBy, MediaUsageItem{
							Collection: collName,
							RecordID:   record.Id,
							Title:      title,
							Slug:       record.GetString("slug"),
						})
						usage.UsageCount++
					}
				}
			}
		}
	}

	return usage, nil
}

// RemoveMediaRefFromRecord removes an external_media ID from a record's media_refs field.
func RemoveMediaRefFromRecord(app *pocketbase.PocketBase, collectionName, recordID, externalMediaID string) error {
	record, err := app.FindRecordById(collectionName, recordID)
	if err != nil {
		return err
	}

	// Get current media_refs
	mediaRefs := record.GetStringSlice("media_refs")
	if mediaRefs == nil {
		return nil // Nothing to remove
	}

	// Filter out the external media ID
	newRefs := make([]string, 0, len(mediaRefs))
	for _, ref := range mediaRefs {
		if ref != externalMediaID {
			newRefs = append(newRefs, ref)
		}
	}

	// Update the record
	record.Set("media_refs", newRefs)
	return app.Save(record)
}

// MediaItem represents a single stored file and its context.
type MediaItem struct {
	Collection    string           `json:"collection"`
	CollectionID  string           `json:"collection_id"`
	RecordID      string           `json:"record_id"`
	Field         string           `json:"field"`
	Filename      string           `json:"filename"`
	URL           string           `json:"url"`
	Size          int64            `json:"size"`
	Mime          string           `json:"mime"`
	Width         int              `json:"width,omitempty"`
	Height        int              `json:"height,omitempty"`
	UploadedAt    time.Time        `json:"uploaded_at"`
	RelativePath  string           `json:"relative_path"`
	DisplayName   string           `json:"display_name,omitempty"`
	AltText       string           `json:"alt_text,omitempty"`
	Description   string           `json:"description,omitempty"`
	RecordLabel   string           `json:"record_label,omitempty"`
	CollectionKey string           `json:"collection_key,omitempty"`
	Orphan        bool             `json:"orphan,omitempty"`
	ThumbnailURL  string           `json:"thumbnail_url,omitempty"`
	External      bool             `json:"external,omitempty"`
	Provider      string           `json:"provider,omitempty"`
	EmbedURL      string           `json:"embed_url,omitempty"`
	UsageCount    int              `json:"usage_count,omitempty"`
	UsedBy        []MediaUsageItem `json:"used_by,omitempty"`
	Tags          []MediaTag       `json:"tags,omitempty"`
}

// MediaTag represents a tag associated with a media item.
type MediaTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// FlattenFileValue normalizes PocketBase file field values (string or []string) into a slice.
func FlattenFileValue(v interface{}) []string {
	switch val := v.(type) {
	case string:
		if val == "" {
			return nil
		}
		return []string{val}
	case []string:
		out := make([]string, 0, len(val))
		for _, f := range val {
			if f != "" {
				out = append(out, f)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, raw := range val {
			if s, ok := raw.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// RemoveFileFromValue returns the updated PocketBase file field value after removing filename.
// It preserves the original type shape (string vs []string) when possible.
func RemoveFileFromValue(current interface{}, filename string) (interface{}, bool) {
	switch val := current.(type) {
	case string:
		if val == filename {
			return "", true
		}
		return val, false
	case []string:
		updated := make([]string, 0, len(val))
		removed := false
		for _, f := range val {
			if f == filename {
				removed = true
				continue
			}
			updated = append(updated, f)
		}
		return updated, removed
	case []interface{}:
		updated := make([]string, 0, len(val))
		removed := false
		for _, raw := range val {
			if s, ok := raw.(string); ok {
				if s == filename {
					removed = true
					continue
				}
				updated = append(updated, s)
			}
		}
		return updated, removed
	default:
		return current, false
	}
}

// extractImageDimensions reads only the image header to get dimensions without loading the full image.
// Returns (0, 0) if dimensions cannot be determined (not an image, corrupt, etc.).
func extractImageDimensions(filePath string) (width, height int) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	config, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}

// BuildMediaItem constructs MediaItem metadata by inspecting the file on disk.
// Uses StorageService to locate files across multiple storage locations.
func BuildMediaItem(storage *StorageService, collectionName, collectionID, recordID, field, filename string, recordCreated time.Time) (MediaItem, error) {
	item := MediaItem{
		Collection:    collectionName,
		CollectionID:  collectionID,
		RecordID:      recordID,
		Field:         field,
		Filename:      filename,
		URL:           fmt.Sprintf("/api/files/%s/%s/%s", collectionID, recordID, filename),
		RelativePath:  filepath.ToSlash(filepath.Join(collectionID, recordID, filename)),
		CollectionKey: collectionName,
	}

	fullPath, found := storage.FindFile(collectionID, recordID, filename)
	if !found {
		return item, fmt.Errorf("file not found: %s/%s/%s", collectionID, recordID, filename)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return item, err
	}

	item.Size = info.Size()

	uploadedAt := info.ModTime()
	if uploadedAt.IsZero() && !recordCreated.IsZero() {
		uploadedAt = recordCreated
	}
	item.UploadedAt = uploadedAt

	// Detect mime
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if mimeType == "" {
		// Read a small sample to detect content type
		f, err := os.Open(fullPath)
		if err == nil {
			defer f.Close()
			sniff := make([]byte, 512)
			n, _ := io.ReadFull(f, sniff)
			mimeType = http.DetectContentType(sniff[:n])
		}
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	item.Mime = mimeType

	// Extract image dimensions for image files (fast: only reads header)
	if strings.HasPrefix(mimeType, "image/") {
		item.Width, item.Height = extractImageDimensions(fullPath)
	}

	// Check for existing thumbnail, regenerate if missing for supported formats
	// Skip if this file is already a thumbnail (prevent recursive thumbnail generation)
	isAlreadyThumbnail := strings.Contains(filename, ThumbnailSuffix)
	if !isAlreadyThumbnail {
		thumbFilename := GetThumbnailPath(filename)
		if _, found := storage.FindFile(collectionID, recordID, thumbFilename); found {
			// Use custom thumbnail endpoint (PocketBase won't serve unregistered files)
			item.ThumbnailURL = fmt.Sprintf("/api/media/thumb/%s/%s/%s", collectionID, recordID, thumbFilename)
		} else if IsSupportedFormat(mimeType) {
			// Thumbnail missing but format is supported - regenerate it
			thumbService := NewThumbnailService(storage)
			if generatedName, err := thumbService.GenerateThumbnail(collectionID, recordID, filename, ThumbnailSize); err == nil {
				// Use custom thumbnail endpoint (PocketBase won't serve unregistered files)
				item.ThumbnailURL = fmt.Sprintf("/api/media/thumb/%s/%s/%s", collectionID, recordID, generatedName)
			}
		}
	}

	return item, nil
}

// BuildMediaItemLegacy constructs MediaItem metadata using the old dataDir-based approach.
// Deprecated: Use BuildMediaItem with StorageService instead.
// This is kept for backward compatibility during migration.
func BuildMediaItemLegacy(dataDir, collectionName, collectionID, recordID, field, filename string, recordCreated time.Time) (MediaItem, error) {
	item := MediaItem{
		Collection:    collectionName,
		CollectionID:  collectionID,
		RecordID:      recordID,
		Field:         field,
		Filename:      filename,
		URL:           fmt.Sprintf("/api/files/%s/%s/%s", collectionID, recordID, filename),
		RelativePath:  filepath.ToSlash(filepath.Join(collectionID, recordID, filename)),
		CollectionKey: collectionName,
	}

	fullPath := filepath.Join(dataDir, "storage", collectionID, recordID, filename)
	info, err := os.Stat(fullPath)
	if err != nil {
		return item, err
	}

	item.Size = info.Size()

	uploadedAt := info.ModTime()
	if uploadedAt.IsZero() && !recordCreated.IsZero() {
		uploadedAt = recordCreated
	}
	item.UploadedAt = uploadedAt

	// Detect mime
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if mimeType == "" {
		// Read a small sample to detect content type
		f, err := os.Open(fullPath)
		if err == nil {
			defer f.Close()
			sniff := make([]byte, 512)
			n, _ := io.ReadFull(f, sniff)
			mimeType = http.DetectContentType(sniff[:n])
		}
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	item.Mime = mimeType

	// Extract image dimensions for image files (fast: only reads header)
	if strings.HasPrefix(mimeType, "image/") {
		item.Width, item.Height = extractImageDimensions(fullPath)
	}

	return item, nil
}

// GetMediaDisplayName retrieves a custom display name for a media file.
// Returns empty string if no custom name is set.
func GetMediaDisplayName(app *pocketbase.PocketBase, collectionID, recordID, filename string) string {
	collection, err := app.FindCollectionByNameOrId("media_display_names")
	if err != nil {
		return ""
	}

	filter := fmt.Sprintf("collection_id = '%s' && record_id = '%s' && filename = '%s'", collectionID, recordID, filename)
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, nil)
	if err != nil || len(records) == 0 {
		return ""
	}

	return records[0].GetString("display_name")
}

// SetMediaDisplayName sets a custom display name for a media file.
// Creates a new record or updates existing one.
func SetMediaDisplayName(app *pocketbase.PocketBase, collectionID, recordID, filename, displayName string) error {
	collection, err := app.FindCollectionByNameOrId("media_display_names")
	if err != nil {
		return fmt.Errorf("media_display_names collection not found: %w", err)
	}

	// Check if record already exists
	filter := fmt.Sprintf("collection_id = '%s' && record_id = '%s' && filename = '%s'", collectionID, recordID, filename)
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, nil)

	var record *core.Record
	if err == nil && len(records) > 0 {
		// Update existing
		record = records[0]
	} else {
		// Create new
		record = core.NewRecord(collection)
		record.Set("collection_id", collectionID)
		record.Set("record_id", recordID)
		record.Set("filename", filename)
	}

	record.Set("display_name", displayName)
	return app.Save(record)
}

// DeleteMediaDisplayName removes a custom display name for a media file.
func DeleteMediaDisplayName(app *pocketbase.PocketBase, collectionID, recordID, filename string) error {
	collection, err := app.FindCollectionByNameOrId("media_display_names")
	if err != nil {
		return nil // Collection doesn't exist, nothing to delete
	}

	filter := fmt.Sprintf("collection_id = '%s' && record_id = '%s' && filename = '%s'", collectionID, recordID, filename)
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, nil)
	if err != nil || len(records) == 0 {
		return nil // No record to delete
	}

	return app.Delete(records[0])
}

// MediaMetadata holds all custom metadata for a media file.
type MediaMetadata struct {
	DisplayName string   `json:"display_name"`
	AltText     string   `json:"alt_text"`
	Description string   `json:"description"`
	TagIDs      []string `json:"tag_ids"`
}

// GetMediaMetadata retrieves all custom metadata for a media file.
func GetMediaMetadata(app *pocketbase.PocketBase, collectionID, recordID, filename string) *MediaMetadata {
	collection, err := app.FindCollectionByNameOrId("media_display_names")
	if err != nil {
		return nil
	}

	filter := fmt.Sprintf("collection_id = '%s' && record_id = '%s' && filename = '%s'", collectionID, recordID, filename)
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, nil)
	if err != nil || len(records) == 0 {
		return nil
	}

	record := records[0]
	return &MediaMetadata{
		DisplayName: record.GetString("display_name"),
		AltText:     record.GetString("alt_text"),
		Description: record.GetString("description"),
		TagIDs:      record.GetStringSlice("admin_tags"),
	}
}

// SetMediaMetadata sets all custom metadata for a media file.
// Creates a new record or updates existing one.
func SetMediaMetadata(app *pocketbase.PocketBase, collectionID, recordID, filename string, metadata MediaMetadata) error {
	collection, err := app.FindCollectionByNameOrId("media_display_names")
	if err != nil {
		return fmt.Errorf("media_display_names collection not found: %w", err)
	}

	// Check if record already exists
	filter := fmt.Sprintf("collection_id = '%s' && record_id = '%s' && filename = '%s'", collectionID, recordID, filename)
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, nil)

	// If all fields are empty and record exists, delete it
	isEmpty := metadata.DisplayName == "" && metadata.AltText == "" && metadata.Description == "" && len(metadata.TagIDs) == 0
	if isEmpty {
		if err == nil && len(records) > 0 {
			return app.Delete(records[0])
		}
		return nil
	}

	var record *core.Record
	if err == nil && len(records) > 0 {
		// Update existing
		record = records[0]
	} else {
		// Create new
		record = core.NewRecord(collection)
		record.Set("collection_id", collectionID)
		record.Set("record_id", recordID)
		record.Set("filename", filename)
	}

	record.Set("display_name", metadata.DisplayName)
	record.Set("alt_text", metadata.AltText)
	record.Set("description", metadata.Description)
	record.Set("admin_tags", metadata.TagIDs)

	return app.Save(record)
}
