package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/pocketbase/dbx"
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
// It checks media_refs relations, library URL fields, and project experience URLs.
func FindMediaUsage(app *pocketbase.PocketBase, externalMediaID string) (MediaUsage, error) {
	usage := MediaUsage{
		UsageCount: 0,
		UsedBy:     []MediaUsageItem{},
	}

	seen := make(map[string]bool)
	collectionsWithMediaRefs := []string{"posts", "projects", "talks", "custom_content"}

	for _, collName := range collectionsWithMediaRefs {
		collection, err := app.FindCollectionByNameOrId(collName)
		if err != nil {
			continue
		}
		if collection.Fields.GetByName("media_refs") == nil {
			continue
		}

		records, err := app.FindRecordsByFilter(collName, "media_refs ~ {:id}", "", 100, 0, dbx.Params{"id": externalMediaID})
		if err != nil {
			continue
		}

		for _, record := range records {
			key := collName + ":" + record.Id
			if seen[key] {
				continue
			}
			seen[key] = true
			usage.addRecord(collName, record)
		}
	}

	extRecord, err := app.FindRecordById("external_media", externalMediaID)
	if err == nil {
		mediaURL := extRecord.GetString("url")
		if mediaURL != "" {
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
			}

			for collName, fields := range libraryURLFields {
				collection, err := app.FindCollectionByNameOrId(collName)
				if err != nil {
					continue
				}

				for _, fieldName := range fields {
					if collection.Fields.GetByName(fieldName) == nil {
						continue
					}
					filter := fmt.Sprintf("%s = {:url}", fieldName)
					records, err := app.FindRecordsByFilter(collName, filter, "", 100, 0, dbx.Params{"url": mediaURL})
					if err != nil {
						continue
					}
					for _, record := range records {
						key := collName + ":" + record.Id
						if seen[key] {
							continue
						}
						seen[key] = true
						usage.addRecord(collName, record)
					}
				}
			}

			for _, item := range FindProjectExperienceMediaUsage(app, mediaURL).UsedBy {
				key := item.Collection + ":" + item.RecordID
				if seen[key] {
					continue
				}
				seen[key] = true
				usage.UsedBy = append(usage.UsedBy, item)
				usage.UsageCount++
			}
		}
	}

	return usage, nil
}

func (u *MediaUsage) addRecord(collection string, record *core.Record) {
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
	u.UsedBy = append(u.UsedBy, MediaUsageItem{
		Collection: collection,
		RecordID:   record.Id,
		Title:      title,
		Slug:       record.GetString("slug"),
	})
	u.UsageCount++
}

func FindProjectExperienceMediaUsage(app *pocketbase.PocketBase, mediaURL string) MediaUsage {
	usage := MediaUsage{UsedBy: []MediaUsageItem{}}
	if mediaURL == "" {
		return usage
	}
	for _, collectionName := range []string{"projects", "demo_projects"} {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil || collection.Fields.GetByName("experience") == nil {
			continue
		}

		const pageSize = 200
		for offset := 0; ; offset += pageSize {
			records, err := app.FindRecordsByFilter(collectionName, "", "", pageSize, offset, nil)
			if err != nil {
				break
			}
			for _, record := range records {
				_, found := RewriteNestedMediaURLs(record.Get("experience"), mediaURL, mediaURL)
				if found {
					usage.addRecord(collectionName, record)
				}
			}
			if len(records) < pageSize {
				break
			}
		}
	}
	return usage
}

func RewriteNestedMediaURLs(value any, oldURL, newURL string) (any, bool) {
	if oldURL == "" {
		return value, false
	}
	raw, err := nestedMediaJSON(value)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		return value, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return value, false
	}

	changed := false
	var walk func(any, string) any
	walk = func(current any, key string) any {
		switch node := current.(type) {
		case map[string]any:
			for childKey, child := range node {
				node[childKey] = walk(child, childKey)
			}
		case []any:
			for i, child := range node {
				node[i] = walk(child, key)
			}
		case string:
			if (key == "src" || key == "poster" || key == "href") && sameExperienceMediaURL(node, oldURL) {
				changed = true
				if oldURL == newURL {
					return node
				}
				if newURL != "" {
					old, err := url.Parse(node)
					replacement, nextErr := url.Parse(newURL)
					if err == nil && nextErr == nil && replacement.RawQuery == "" {
						replacement.RawQuery = old.RawQuery
						return replacement.String()
					}
				}
				return newURL
			}
		}
		return current
	}
	return walk(document, ""), changed
}

func nestedMediaJSON(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return json.Marshal(value)
	}
}

// RemoveMediaRefFromRecord removes an external_media ID and any nested URLs from a record.
func RemoveMediaRefFromRecord(app *pocketbase.PocketBase, collectionName, recordID, externalMediaID string) error {
	record, err := app.FindRecordById(collectionName, recordID)
	if err != nil {
		return err
	}

	changed := false
	mediaRefs := record.GetStringSlice("media_refs")
	if mediaRefs != nil {
		newRefs := make([]string, 0, len(mediaRefs))
		for _, ref := range mediaRefs {
			if ref != externalMediaID {
				newRefs = append(newRefs, ref)
			}
		}
		record.Set("media_refs", newRefs)
		changed = true
	}

	if collectionName == "projects" || collectionName == "demo_projects" {
		if external, err := app.FindRecordById("external_media", externalMediaID); err == nil {
			experience, rewritten := RewriteNestedMediaURLs(record.Get("experience"), external.GetString("url"), "")
			if rewritten {
				record.Set("experience", experience)
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
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

	filter := "collection_id = {:collId} && record_id = {:recId} && filename = {:fname}"
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, dbx.Params{"collId": collectionID, "recId": recordID, "fname": filename})
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
	filter := "collection_id = {:collId} && record_id = {:recId} && filename = {:fname}"
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, dbx.Params{"collId": collectionID, "recId": recordID, "fname": filename})

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

	filter := "collection_id = {:collId} && record_id = {:recId} && filename = {:fname}"
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, dbx.Params{"collId": collectionID, "recId": recordID, "fname": filename})
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

	filter := "collection_id = {:collId} && record_id = {:recId} && filename = {:fname}"
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, dbx.Params{"collId": collectionID, "recId": recordID, "fname": filename})
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
	filter := "collection_id = {:collId} && record_id = {:recId} && filename = {:fname}"
	records, err := app.FindRecordsByFilter(collection.Name, filter, "", 1, 0, dbx.Params{"collId": collectionID, "recId": recordID, "fname": filename})

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

// A public absolute URL and its local file path refer to the same managed asset.
// Treat matching external paths conservatively as used, too, rather than deleting media.
func sameExperienceMediaURL(value, target string) bool {
	if value == target {
		return true
	}
	a, err := url.Parse(value)
	if err != nil {
		return false
	}
	b, err := url.Parse(target)
	if err != nil {
		return false
	}
	return strings.HasPrefix(a.Path, "/api/files/") && a.Path == b.Path
}
