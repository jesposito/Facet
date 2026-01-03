package services

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MediaItem represents a single stored file and its context.
type MediaItem struct {
	Collection    string    `json:"collection"`
	CollectionID  string    `json:"collection_id"`
	RecordID      string    `json:"record_id"`
	Field         string    `json:"field"`
	Filename      string    `json:"filename"`
	URL           string    `json:"url"`
	Size          int64     `json:"size"`
	Mime          string    `json:"mime"`
	UploadedAt    time.Time `json:"uploaded_at"`
	RelativePath  string    `json:"relative_path"`
	DisplayName   string    `json:"display_name,omitempty"`
	RecordLabel   string    `json:"record_label,omitempty"`
	CollectionKey string    `json:"collection_key,omitempty"`
	Orphan        bool      `json:"orphan,omitempty"`
	ThumbnailURL  string    `json:"thumbnail_url,omitempty"`
	External      bool      `json:"external,omitempty"`
	Provider      string    `json:"provider,omitempty"`
	EmbedURL      string    `json:"embed_url,omitempty"`
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

// BuildMediaItem constructs MediaItem metadata by inspecting the file on disk.
// dataDir should be the PocketBase data dir (e.g., pb_data).
func BuildMediaItem(dataDir, collectionName, collectionID, recordID, field, filename string, recordCreated time.Time) (MediaItem, error) {
	item := MediaItem{
		Collection:    collectionName,
		CollectionID:  collectionID,
		RecordID:      recordID,
		Field:         field,
		Filename:      filename,
		URL:           fmt.Sprintf("/api/files/%s/%s/%s", collectionID, recordID, filename),
		RelativePath:  filepath.ToSlash(filepath.Join("storage", collectionID, recordID, filename)),
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

	return item, nil
}
