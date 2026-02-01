package hooks

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"facet/services"
	"facet/services/mediaembed"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Path validation errors
var (
	ErrInvalidPath  = errors.New("invalid path")
	ErrPathEscapes  = errors.New("path escapes storage directory")
	ErrSymlink      = errors.New("refusing to operate on symbolic links")
	ErrAbsolutePath = errors.New("absolute paths not allowed")
	ErrIsDirectory  = errors.New("refusing to delete directories")
)

// collectionsWithImageFields lists collections that have image file fields
// and should have thumbnails generated automatically.
var collectionsWithImageFields = []string{
	"uploads",
	"profile",
	"experience",
	"projects",
	"education",
	"posts",
	"talks",
	"views",
	"site_settings",
	"custom_content",
	"testimonials",
}

// RegisterMediaHooks exposes admin-only media listing and deletion endpoints.
// uploadsDir is the path to the primary uploads directory (e.g., /uploads in Docker).
// If empty, only the fallback location (pb_data/storage) will be used.
func RegisterMediaHooks(app *pocketbase.PocketBase, uploadsDir string) {
	// Register thumbnail generation hooks for all collections with image fields
	for _, collName := range collectionsWithImageFields {
		registerThumbnailHook(app, collName, uploadsDir)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Create StorageService with app's data directory now that it's available
		storage := services.NewStorageService(services.StorageConfig{
			UploadsDir: uploadsDir,
			DataDir:    app.DataDir(),
		})
		se.Router.GET("/api/media", func(e *core.RequestEvent) error {
			// Auth is enforced by middleware; log principal
			if e.Auth != nil {
				app.Logger().Debug("media list auth ok", "id", e.Auth.Id, "email", e.Auth.Email())
			}

			query := e.Request.URL.Query()
			includeOrphans := strings.TrimSpace(strings.ToLower(query.Get("includeOrphans"))) == "1"
			orphansOnly := strings.TrimSpace(strings.ToLower(query.Get("orphans"))) == "1"

			items, referenced, referencedSize, err := collectMediaItems(app, storage)
			if err != nil {
				app.Logger().Error("media list failed", "error", err)
				return apis.NewBadRequestError("failed to enumerate media", err)
			}

			externalItems, err := collectExternalMediaItems(app)
			if err != nil {
				externalItems = []services.MediaItem{}
			}

			orphanItems, orphanSize, storageSize, storageFiles, err := collectOrphanMediaItems(app, storage, referenced)
			if err != nil {
				orphanItems = []services.MediaItem{}
				orphanSize = 0
				storageSize = 0
				storageFiles = 0
			}

			combined := append(items, externalItems...)
			if orphansOnly {
				combined = orphanItems
			} else if includeOrphans {
				combined = append(combined, orphanItems...)
			}

			search := strings.TrimSpace(strings.ToLower(query.Get("q")))
			typeFilter := strings.ToLower(strings.TrimSpace(query.Get("type")))       // "image", "video", "audio", "document", or ""
			collectionFilter := strings.TrimSpace(strings.ToLower(query.Get("collection")))
			tagsFilter := strings.TrimSpace(query.Get("tags"))                        // comma-separated tag IDs
			usageFilter := strings.ToLower(strings.TrimSpace(query.Get("usage")))     // "in_use", "not_in_use", or "" for all
			sortField := strings.ToLower(strings.TrimSpace(query.Get("sort")))        // "date", "name", "size"
			sortOrder := strings.ToLower(strings.TrimSpace(query.Get("order")))       // "asc", "desc"
			if sortOrder == "" {
				sortOrder = "desc"
			}

			// Parse tags filter into a set for efficient lookup
			var tagFilterSet map[string]struct{}
			if tagsFilter != "" {
				tagFilterSet = make(map[string]struct{})
				for _, tagID := range strings.Split(tagsFilter, ",") {
					tagID = strings.TrimSpace(tagID)
					if tagID != "" {
						tagFilterSet[tagID] = struct{}{}
					}
				}
			}

			filtered := make([]services.MediaItem, 0, len(combined))
			for _, item := range combined {
				if search != "" && !strings.Contains(strings.ToLower(item.Filename), search) && !strings.Contains(strings.ToLower(item.DisplayName), search) {
					continue
				}
				if collectionFilter != "" && strings.ToLower(item.Collection) != collectionFilter && strings.ToLower(item.CollectionKey) != collectionFilter {
					continue
				}
				// Tag filtering: item must have at least one of the requested tags
				if tagFilterSet != nil {
					hasTag := false
					for _, tag := range item.Tags {
						if _, ok := tagFilterSet[tag.ID]; ok {
							hasTag = true
							break
						}
					}
					if !hasTag {
						continue
					}
				}
				// Extended type filtering
				if typeFilter != "" {
					var match bool
					switch typeFilter {
					case "image":
						match = strings.HasPrefix(item.Mime, "image/")
					case "video":
						match = strings.HasPrefix(item.Mime, "video/")
					case "audio":
						match = strings.HasPrefix(item.Mime, "audio/")
					case "document":
						match = strings.HasPrefix(item.Mime, "application/pdf") ||
							strings.HasPrefix(item.Mime, "application/msword") ||
							strings.HasPrefix(item.Mime, "application/vnd.openxmlformats-officedocument") ||
							strings.HasPrefix(item.Mime, "application/vnd.ms-") ||
							strings.HasPrefix(item.Mime, "application/vnd.oasis.opendocument") ||
							strings.HasPrefix(item.Mime, "text/")
					}
					if !match {
						continue
					}
				}
				// Usage filtering
				if usageFilter == "in_use" && item.UsageCount == 0 {
					continue
				}
				if usageFilter == "not_in_use" && item.UsageCount > 0 {
					continue
				}
				filtered = append(filtered, item)
			}

			// Sort by requested field and order
			sort.Slice(filtered, func(i, j int) bool {
				var less bool
				switch sortField {
				case "name":
					less = strings.ToLower(filtered[i].Filename) < strings.ToLower(filtered[j].Filename)
				case "size":
					less = filtered[i].Size < filtered[j].Size
				default: // "date" or empty
					less = filtered[i].UploadedAt.Before(filtered[j].UploadedAt)
				}
				if sortOrder == "asc" {
					return less
				}
				return !less
			})

			page := parseIntDefault(query.Get("page"), 1)
			perPage := parseIntDefault(query.Get("perPage"), 50)
			if perPage <= 0 {
				perPage = 50
			}
			if perPage > 200 {
				perPage = 200
			}

			total := len(filtered)
			start := (page - 1) * perPage
			if start > total {
				start = total
			}
			end := start + perPage
			if end > total {
				end = total
			}

			// Build sample items for debug (first 3 items showing usage data)
			sampleItems := make([]map[string]any, 0, 3)
			for i := 0; i < len(filtered) && i < 3; i++ {
				item := filtered[i]
				sampleItems = append(sampleItems, map[string]any{
					"collection":  item.Collection,
					"filename":    item.Filename,
					"external":    item.External,
					"orphan":      item.Orphan,
					"usage_count": item.UsageCount,
					"used_by_len": len(item.UsedBy),
				})
			}

			// Include storage configuration in debug info
			storageInfo := storage.GetStorageInfo()

			response := map[string]any{
				"items":      filtered[start:end],
				"page":       page,
				"perPage":    perPage,
				"totalItems": total,
				"totalPages": (total + perPage - 1) / perPage,
				"debug": map[string]any{
					"internalCount":  len(items),
					"externalCount":  len(externalItems),
					"orphanCount":    len(orphanItems),
					"combinedCount":  len(combined),
					"filteredCount":  len(filtered),
					"referencedKeys": len(referenced),
					"sampleItems":    sampleItems,
					"storage":        storageInfo,
				},
				"stats": map[string]any{
					"referencedFiles": len(items) + len(externalItems),
					"referencedSize":  referencedSize,
					"orphanFiles":     len(orphanItems),
					"orphanSize":      orphanSize,
					"totalFiles":      len(items) + len(externalItems) + len(orphanItems),
					"totalSize":       referencedSize + orphanSize,
					"storageFiles":    storageFiles,
					"storageSize":     storageSize,
				},
			}

			return e.JSON(http.StatusOK, response)
		}).Bind(apis.RequireAuth())

		se.Router.POST("/api/media/external", func(e *core.RequestEvent) error {
			var req struct {
				URL          string `json:"url"`
				Title        string `json:"title"`
				Mime         string `json:"mime"`
				ThumbnailURL string `json:"thumbnail_url"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}
			if req.URL == "" {
				return apis.NewBadRequestError("url is required", nil)
			}
			if _, err := validateURL(req.URL); err != nil {
				return apis.NewBadRequestError("invalid url", err)
			}
			if req.ThumbnailURL != "" {
				if _, err := validateURL(req.ThumbnailURL); err != nil {
					return apis.NewBadRequestError("invalid thumbnail_url", err)
				}
			}

			collection, err := app.FindCollectionByNameOrId("external_media")
			if err != nil {
				return apis.NewBadRequestError("external media not configured", err)
			}

			record := core.NewRecord(collection)
			record.Set("url", req.URL)
			if req.Title != "" {
				record.Set("title", req.Title)
			}
			if req.Mime != "" {
				record.Set("mime", req.Mime)
			}
			if req.ThumbnailURL != "" {
				record.Set("thumbnail_url", req.ThumbnailURL)
			}
			if err := app.Save(record); err != nil {
				return apis.NewBadRequestError("failed to save external media", err)
			}
			return e.JSON(http.StatusOK, map[string]string{
				"id":  record.Id,
				"url": req.URL,
			})
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/api/media/external/{id}", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			if id == "" {
				return apis.NewBadRequestError("missing id", nil)
			}

			// Check for force parameter
			query := e.Request.URL.Query()
			forceDelete := strings.TrimSpace(strings.ToLower(query.Get("force"))) == "1"

			collection, err := app.FindCollectionByNameOrId("external_media")
			if err != nil {
				return apis.NewBadRequestError("external media not configured", err)
			}
			record, err := app.FindRecordById(collection.Name, id)
			if err != nil {
				return apis.NewNotFoundError("not found", err)
			}

			// Check if media is referenced by any content
			usage, err := services.FindMediaUsage(app, id)
			if err != nil {
				app.Logger().Warn("media: failed to check usage", "id", id, "error", err)
				// Continue with deletion check even if usage check fails
			}

			// If media is referenced and force is not set, return usage info
			if usage.UsageCount > 0 && !forceDelete {
				return e.JSON(http.StatusConflict, map[string]any{
					"status":      "referenced",
					"message":     fmt.Sprintf("Media is referenced by %d content item(s)", usage.UsageCount),
					"usage_count": usage.UsageCount,
					"used_by":     usage.UsedBy,
				})
			}

			// If force delete, remove references from all content first
			if forceDelete && usage.UsageCount > 0 {
				for _, item := range usage.UsedBy {
					if err := services.RemoveMediaRefFromRecord(app, item.Collection, item.RecordID, id); err != nil {
						app.Logger().Warn("media: failed to remove reference", "collection", item.Collection, "record", item.RecordID, "error", err)
						// Continue trying to remove other references
					}
				}
				app.Logger().Info("media: removed references before delete", "id", id, "count", usage.UsageCount)
			}

			if err := app.Delete(record); err != nil {
				return apis.NewBadRequestError("failed to delete external media", err)
			}

			response := map[string]any{
				"status": "deleted",
			}
			if forceDelete && usage.UsageCount > 0 {
				response["references_removed"] = usage.UsageCount
			}

			return e.JSON(http.StatusOK, response)
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/api/media", func(e *core.RequestEvent) error {
			var req struct {
				CollectionID string `json:"collection_id"`
				RecordID     string `json:"record_id"`
				Field        string `json:"field"`
				Filename     string `json:"filename"`
				RelativePath string `json:"relative_path"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			// Orphan deletion path: delete by relative path
			if req.RelativePath != "" && (req.CollectionID == "" || req.Field == "") {
				target, err := resolveStoragePathMulti(storage, req.RelativePath)
				if err != nil {
					// Return specific error messages based on error type
					switch err {
					case ErrSymlink:
						app.Logger().Warn("media: rejected symlink deletion attempt", "path", req.RelativePath)
						return apis.NewBadRequestError("symbolic links cannot be deleted", err)
					case ErrPathEscapes:
						app.Logger().Warn("media: rejected path escaping storage", "path", req.RelativePath)
						return apis.NewBadRequestError("path escapes storage directory", err)
					case ErrAbsolutePath:
						app.Logger().Warn("media: rejected absolute path", "path", req.RelativePath)
						return apis.NewBadRequestError("absolute paths not allowed", err)
					default:
						app.Logger().Warn("media: invalid path", "path", req.RelativePath, "error", err)
						return apis.NewBadRequestError("invalid path", err)
					}
				}
				if err := os.Remove(target); err != nil {
					app.Logger().Warn("media: failed to delete orphan file", "path", target, "error", err)
					return apis.NewBadRequestError("failed to delete file", err)
				}
				app.Logger().Info("media: deleted orphan file", "path", target, "relative", req.RelativePath)
				return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
			}

			if req.CollectionID == "" || req.RecordID == "" || req.Field == "" || req.Filename == "" {
				return apis.NewBadRequestError("collection_id, record_id, field, and filename are required", nil)
			}

			collection, err := app.FindCollectionByNameOrId(req.CollectionID)
			if err != nil {
				return apis.NewBadRequestError("collection not found", err)
			}

			record, err := app.FindRecordById(collection.Name, req.RecordID)
			if err != nil {
				return apis.NewBadRequestError("record not found", err)
			}

			current := record.Get(req.Field)
			_, removed := services.RemoveFileFromValue(current, req.Filename)
			if !removed {
				return apis.NewBadRequestError("file not found on record", nil)
			}

			// For uploads collection, delete the entire record since the file field is required
			// and clearing it would fail validation. For other collections, just clear the file field.
			if collection.Name == "uploads" {
				// Before deleting, clean up any mirror entries in external_media
				// that point to this upload's file URL
				fileURL := fmt.Sprintf("/api/files/%s/%s/%s", collection.Id, record.Id, req.Filename)
				cleanupMirrorEntries(app, fileURL)

				if err := app.Delete(record); err != nil {
					app.Logger().Error("media delete failed to delete uploads record", "error", err)
					return apis.NewBadRequestError("failed to delete record", err)
				}
			} else {
				updated, _ := services.RemoveFileFromValue(current, req.Filename)
				record.Set(req.Field, updated)
				if err := app.Save(record); err != nil {
					app.Logger().Error("media delete failed to update record", "error", err)
					return apis.NewBadRequestError("failed to update record", err)
				}

				// Remove file from storage using StorageService (handles multiple locations)
				if err := storage.DeleteFile(collection.Id, record.Id, req.Filename); err != nil {
					app.Logger().Warn("media: failed to delete file from storage", "error", err)
					// Don't fail the request - file might already be gone
				}
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		}).Bind(apis.RequireAuth())

		// Update display name for any media file (legacy endpoint, still works)
		se.Router.PATCH("/api/media/display-name", func(e *core.RequestEvent) error {
			var req struct {
				CollectionID string `json:"collection_id"`
				RecordID     string `json:"record_id"`
				Filename     string `json:"filename"`
				DisplayName  string `json:"display_name"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			if req.CollectionID == "" || req.RecordID == "" || req.Filename == "" {
				return apis.NewBadRequestError("collection_id, record_id, and filename are required", nil)
			}

			if req.DisplayName == "" {
				// If display name is empty, delete any custom name
				if err := services.DeleteMediaDisplayName(app, req.CollectionID, req.RecordID, req.Filename); err != nil {
					return apis.NewBadRequestError("failed to delete display name", err)
				}
			} else {
				if err := services.SetMediaDisplayName(app, req.CollectionID, req.RecordID, req.Filename, req.DisplayName); err != nil {
					return apis.NewBadRequestError("failed to set display name", err)
				}
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		}).Bind(apis.RequireAuth())

		// Update all metadata for any media file (display_name, alt_text, description, tags)
		se.Router.PATCH("/api/media/metadata", func(e *core.RequestEvent) error {
			var req struct {
				CollectionID string   `json:"collection_id"`
				RecordID     string   `json:"record_id"`
				Filename     string   `json:"filename"`
				DisplayName  string   `json:"display_name"`
				AltText      string   `json:"alt_text"`
				Description  string   `json:"description"`
				TagIDs       []string `json:"tag_ids"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			if req.CollectionID == "" || req.RecordID == "" || req.Filename == "" {
				return apis.NewBadRequestError("collection_id, record_id, and filename are required", nil)
			}

			metadata := services.MediaMetadata{
				DisplayName: req.DisplayName,
				AltText:     req.AltText,
				Description: req.Description,
				TagIDs:      req.TagIDs,
			}

			if err := services.SetMediaMetadata(app, req.CollectionID, req.RecordID, req.Filename, metadata); err != nil {
				return apis.NewBadRequestError("failed to set metadata", err)
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		}).Bind(apis.RequireAuth())

		se.Router.POST("/api/media/bulk-delete", func(e *core.RequestEvent) error {
			var req struct {
				Orphans []string `json:"orphans"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			if len(req.Orphans) == 0 {
				return apis.NewBadRequestError("no orphans specified", nil)
			}

			if len(req.Orphans) > 100 {
				return apis.NewBadRequestError("maximum 100 files per request", nil)
			}

			deleted := 0
			failed := 0
			var errors []map[string]string

			for _, relativePath := range req.Orphans {
				target, err := resolveStoragePathMulti(storage, relativePath)
				if err != nil {
					failed++
					errorMsg := "invalid path"
					switch err {
					case ErrSymlink:
						errorMsg = "symbolic link"
						app.Logger().Warn("bulk delete: rejected symlink", "path", relativePath)
					case ErrPathEscapes:
						errorMsg = "path escapes storage"
						app.Logger().Warn("bulk delete: rejected path escape", "path", relativePath)
					case ErrAbsolutePath:
						errorMsg = "absolute path not allowed"
						app.Logger().Warn("bulk delete: rejected absolute path", "path", relativePath)
					default:
						app.Logger().Warn("bulk delete: invalid path", "path", relativePath, "error", err)
					}

					errors = append(errors, map[string]string{
						"path":  relativePath,
						"error": errorMsg,
					})
					continue
				}

				if err := os.Remove(target); err != nil {
					app.Logger().Warn("bulk delete: failed to delete file", "path", target, "error", err)
					failed++
					errors = append(errors, map[string]string{
						"path":  relativePath,
						"error": err.Error(),
					})
				} else {
					deleted++
					app.Logger().Info("bulk delete: deleted orphan", "path", target, "relative", relativePath)
				}
			}

			response := map[string]any{
				"deleted": deleted,
				"failed":  failed,
				"errors":  errors,
			}

			return e.JSON(http.StatusOK, response)
		}).Bind(apis.RequireAuth())

		// Storage info endpoint for diagnostics
		se.Router.GET("/api/media/storage-info", func(e *core.RequestEvent) error {
			info := storage.GetStorageInfo()
			return e.JSON(http.StatusOK, info)
		}).Bind(apis.RequireAuth())

		// Fetch metadata from external URLs (YouTube, Vimeo, etc.)
		se.Router.GET("/api/media/fetch-metadata", func(e *core.RequestEvent) error {
			rawURL := e.Request.URL.Query().Get("url")
			if rawURL == "" {
				return apis.NewBadRequestError("url parameter is required", nil)
			}

			if !mediaembed.IsSupportedProvider(rawURL) {
				return e.JSON(http.StatusOK, map[string]any{
					"supported": false,
					"message":   "URL is not from a supported provider",
				})
			}

			metadata, err := mediaembed.FetchMetadata(rawURL)
			if err != nil {
				app.Logger().Warn("media: failed to fetch metadata", "url", rawURL, "error", err)
				return e.JSON(http.StatusOK, map[string]any{
					"supported": true,
					"error":     err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"supported":     true,
				"title":         metadata.Title,
				"thumbnail_url": metadata.ThumbnailURL,
				"provider":      metadata.Provider,
				"duration":      metadata.Duration,
			})
		}).Bind(apis.RequireAuth())

		// Mark media as recently used (updates last_used_at timestamp)
		// Now supports all collections with file fields (not just uploads and external_media)
		se.Router.POST("/api/media/mark-used", func(e *core.RequestEvent) error {
			var req struct {
				ID         string `json:"id"`
				Collection string `json:"collection"` // any collection with last_used_at field
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}
			if req.ID == "" {
				return apis.NewBadRequestError("id is required", nil)
			}
			if req.Collection == "" {
				req.Collection = "external_media" // default for backwards compatibility
			}

			// Find the collection and verify it has a last_used_at field
			collection, err := app.FindCollectionByNameOrId(req.Collection)
			if err != nil {
				return apis.NewBadRequestError("invalid collection", err)
			}

			// Check if collection has last_used_at field
			if collection.Fields.GetByName("last_used_at") == nil {
				return apis.NewBadRequestError("collection does not support usage tracking", nil)
			}

			record, err := app.FindRecordById(req.Collection, req.ID)
			if err != nil {
				return apis.NewNotFoundError("media not found", err)
			}

			record.Set("last_used_at", time.Now().UTC())
			if err := app.Save(record); err != nil {
				return apis.NewBadRequestError("failed to update media", err)
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		}).Bind(apis.RequireAuth())

		// Bulk tag media items
		se.Router.POST("/api/media/bulk-tag", func(e *core.RequestEvent) error {
			var req struct {
				Items []struct {
					ID         string `json:"id"`
					Collection string `json:"collection"`
				} `json:"items"`
				AddTags    []string `json:"add_tags"`
				RemoveTags []string `json:"remove_tags"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			if len(req.Items) == 0 {
				return apis.NewBadRequestError("no items specified", nil)
			}
			if len(req.Items) > 100 {
				return apis.NewBadRequestError("maximum 100 items per request", nil)
			}
			if len(req.AddTags) == 0 && len(req.RemoveTags) == 0 {
				return apis.NewBadRequestError("must specify add_tags or remove_tags", nil)
			}

			// Build sets for efficient lookup
			addSet := make(map[string]struct{})
			for _, id := range req.AddTags {
				addSet[id] = struct{}{}
			}
			removeSet := make(map[string]struct{})
			for _, id := range req.RemoveTags {
				removeSet[id] = struct{}{}
			}

			updated := 0
			failed := 0
			var errors []map[string]string

			for _, item := range req.Items {
				// Validate collection exists and has admin_tags field
				collection, colErr := app.FindCollectionByNameOrId(item.Collection)
				if colErr != nil {
					failed++
					errors = append(errors, map[string]string{
						"id":    item.ID,
						"error": "invalid collection",
					})
					continue
				}

				// Check if collection has admin_tags field
				if collection.Fields.GetByName("admin_tags") == nil {
					failed++
					errors = append(errors, map[string]string{
						"id":    item.ID,
						"error": "collection does not support tags",
					})
					continue
				}

				record, err := app.FindRecordById(item.Collection, item.ID)
				if err != nil {
					failed++
					errors = append(errors, map[string]string{
						"id":    item.ID,
						"error": "not found",
					})
					continue
				}

				// Get current tags
				currentTags := record.GetStringSlice("admin_tags")
				tagSet := make(map[string]struct{})
				for _, id := range currentTags {
					tagSet[id] = struct{}{}
				}

				// Add new tags
				for id := range addSet {
					tagSet[id] = struct{}{}
				}

				// Remove tags
				for id := range removeSet {
					delete(tagSet, id)
				}

				// Convert back to slice
				newTags := make([]string, 0, len(tagSet))
				for id := range tagSet {
					newTags = append(newTags, id)
				}

				// Enforce max 10 tags
				if len(newTags) > 10 {
					newTags = newTags[:10]
				}

				record.Set("admin_tags", newTags)
				if err := app.Save(record); err != nil {
					failed++
					errors = append(errors, map[string]string{
						"id":    item.ID,
						"error": err.Error(),
					})
					continue
				}

				updated++
			}

			return e.JSON(http.StatusOK, map[string]any{
				"updated": updated,
				"failed":  failed,
				"errors":  errors,
			})
		}).Bind(apis.RequireAuth())

		// Serve thumbnail files directly (PocketBase doesn't serve unregistered files)
		se.Router.GET("/api/media/thumb/{collection}/{record}/{filename}", func(e *core.RequestEvent) error {
			collectionID := e.Request.PathValue("collection")
			recordID := e.Request.PathValue("record")
			filename := e.Request.PathValue("filename")

			if collectionID == "" || recordID == "" || filename == "" {
				return apis.NewNotFoundError("invalid path", nil)
			}

			// Security: ensure filename contains thumbnail suffix
			if !strings.Contains(filename, services.ThumbnailSuffix) {
				return apis.NewBadRequestError("not a thumbnail file", nil)
			}

			// Find the thumbnail file
			fullPath, found := storage.FindFile(collectionID, recordID, filename)
			if !found {
				return apis.NewNotFoundError("thumbnail not found", nil)
			}

			// Detect MIME type
			mimeType := detectMimeType(fullPath, filename)

			// Set caching headers (thumbnails are immutable - cache for 1 year)
			e.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			e.Response.Header().Set("Content-Type", mimeType)

			// Serve the file
			return e.FileFS(os.DirFS(filepath.Dir(fullPath)), filepath.Base(fullPath))
		}) // No auth required - thumbnails are public like regular files

		// Get recently used media (with fallback to recently created)
		se.Router.GET("/api/media/recent", func(e *core.RequestEvent) error {
			query := e.Request.URL.Query()
			limit := parseIntDefault(query.Get("limit"), 10)
			if limit > 20 {
				limit = 20
			}

			var items []services.MediaItem
			seenIDs := make(map[string]struct{})

			// Helper to add upload items
			addUploadItems := func(records []*core.Record, uploadsCollection *core.Collection) {
				for _, record := range records {
					if _, seen := seenIDs[record.Id]; seen {
						continue
					}
					created := record.GetDateTime("created").Time()
					filename := record.GetString("file")
					if filename == "" {
						continue
					}
					item, err := services.BuildMediaItem(storage, uploadsCollection.Name, uploadsCollection.Id, record.Id, "file", filename, created)
					if err != nil {
						continue
					}
					item.DisplayName = record.GetString("title")
					if item.DisplayName == "" {
						item.DisplayName = filename
					}
					item.AltText = record.GetString("alt_text")
					item.Description = record.GetString("description")
					item.Tags = fetchMediaTags(app, record)
					items = append(items, item)
					seenIDs[record.Id] = struct{}{}
				}
			}

			// Helper to add external media items
			addExternalItems := func(records []*core.Record, externalCollection *core.Collection) {
				for _, record := range records {
					if _, seen := seenIDs[record.Id]; seen {
						continue
					}
					// Skip "mirror" entries that point to internal PocketBase files
					recordURL := record.GetString("url")
					if strings.Contains(recordURL, "/api/files/") {
						continue
					}
					created := record.GetDateTime("created").Time()
					title := record.GetString("title")
					if title == "" {
						title = record.GetString("url")
					}
					normalized := mediaembed.Normalize(record.GetString("url"), record.GetString("mime"), record.GetString("thumbnail_url"))
					item := services.MediaItem{
						Collection:    externalCollection.Name,
						CollectionID:  externalCollection.Id,
						CollectionKey: "external",
						RecordID:      record.Id,
						Field:         "external",
						Filename:      title,
						DisplayName:   title,
						URL:           record.GetString("url"),
						Mime:          normalized.Mime,
						ThumbnailURL:  normalized.ThumbnailURL,
						EmbedURL:      normalized.EmbedURL,
						Provider:      normalized.Provider,
						UploadedAt:    created,
						External:      true,
						AltText:       record.GetString("alt_text"),
						Description:   record.GetString("description"),
						Tags:          fetchMediaTags(app, record),
					}
					items = append(items, item)
					seenIDs[record.Id] = struct{}{}
				}
			}

			// First pass: get items with last_used_at set (recently selected)
			uploadsCollection, uploadsErr := app.FindCollectionByNameOrId("uploads")
			externalCollection, externalErr := app.FindCollectionByNameOrId("external_media")

			if uploadsErr == nil {
				records, err := app.FindRecordsByFilter(uploadsCollection.Name, "last_used_at != ''", "-last_used_at", limit, 0, nil)
				if err == nil {
					addUploadItems(records, uploadsCollection)
				}
			}

			if externalErr == nil {
				records, err := app.FindRecordsByFilter(externalCollection.Name, "last_used_at != ''", "-last_used_at", limit, 0, nil)
				if err == nil {
					addExternalItems(records, externalCollection)
				}
			}

			// Second pass: if we don't have enough items, fill with recently created
			remaining := limit - len(items)
			if remaining > 0 {
				if uploadsErr == nil {
					records, err := app.FindRecordsByFilter(uploadsCollection.Name, "", "-created", remaining, 0, nil)
					if err == nil {
						addUploadItems(records, uploadsCollection)
					}
				}
			}

			remaining = limit - len(items)
			if remaining > 0 {
				if externalErr == nil {
					records, err := app.FindRecordsByFilter(externalCollection.Name, "", "-created", remaining, 0, nil)
					if err == nil {
						addExternalItems(records, externalCollection)
					}
				}
			}

			// Trim to limit
			if len(items) > limit {
				items = items[:limit]
			}

			return e.JSON(http.StatusOK, map[string]any{
				"items": items,
			})
		}).Bind(apis.RequireAuth())

		// Get usage details for a specific external_media item (for delete warning)
		se.Router.GET("/api/media/usage/{id}", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			if id == "" {
				return apis.NewBadRequestError("missing id", nil)
			}

			// Verify the external_media record exists
			_, err := app.FindRecordById("external_media", id)
			if err != nil {
				return apis.NewNotFoundError("external_media not found", err)
			}

			usage, err := services.FindMediaUsage(app, id)
			if err != nil {
				return apis.NewBadRequestError("failed to find usage", err)
			}

			return e.JSON(http.StatusOK, usage)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

func parseIntDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// cleanupMirrorEntries removes external_media records that are mirrors pointing to internal files.
// This is called when an upload is deleted to clean up orphaned mirror entries.
func cleanupMirrorEntries(app *pocketbase.PocketBase, fileURL string) {
	collection, err := app.FindCollectionByNameOrId("external_media")
	if err != nil {
		return
	}

	// Find mirror entries that contain this file URL (could be relative or absolute)
	records, err := app.FindAllRecords(collection.Name)
	if err != nil {
		return
	}

	for _, record := range records {
		recordURL := record.GetString("url")
		// Check if this external_media points to the deleted file
		if strings.Contains(recordURL, fileURL) {
			if err := app.Delete(record); err != nil {
				app.Logger().Warn("cleanupMirrorEntries: failed to delete mirror", "id", record.Id, "error", err)
			} else {
				app.Logger().Debug("cleanupMirrorEntries: deleted mirror", "id", record.Id, "url", recordURL)
			}
		}
	}
}

// fetchMediaTags retrieves admin_tags for a record and returns them as MediaTag slice.
func fetchMediaTags(app *pocketbase.PocketBase, record *core.Record) []services.MediaTag {
	tagIDs := record.GetStringSlice("admin_tags")
	if len(tagIDs) == 0 {
		return nil
	}

	tags := make([]services.MediaTag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		tagRecord, err := app.FindRecordById("admin_tags", tagID)
		if err != nil {
			continue
		}
		tags = append(tags, services.MediaTag{
			ID:    tagRecord.Id,
			Name:  tagRecord.GetString("name"),
			Color: tagRecord.GetString("color"),
		})
	}
	return tags
}

// findLibraryURLUsage checks all *_library_url fields across collections to find
// content that references the given URL (for uploads collection files).
func findLibraryURLUsage(app *pocketbase.PocketBase, fileURL string) services.MediaUsage {
	usage := services.MediaUsage{
		UsageCount: 0,
		UsedBy:     []services.MediaUsageItem{},
	}

	// Collections and their library URL fields
	// Updated to include all collections with library URL fields
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
		"testimonials":   {"avatar_library_url"},
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
			filter := fmt.Sprintf("%s = '%s'", fieldName, fileURL)
			records, err := app.FindRecordsByFilter(collName, filter, "", 100, 0, nil)
			if err != nil {
				continue
			}

			for _, record := range records {
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

				usage.UsedBy = append(usage.UsedBy, services.MediaUsageItem{
					Collection: collName,
					RecordID:   record.Id,
					Title:      title,
					Slug:       record.GetString("slug"),
				})
				usage.UsageCount++
			}
		}
	}

	return usage
}

func collectMediaItems(app *pocketbase.PocketBase, storage *services.StorageService) ([]services.MediaItem, map[string]struct{}, int64, error) {
	collections := []string{
		"profile",
		"experience",
		"projects",
		"education",
		"certifications",
		"posts",
		"talks",
		"views",
		"uploads",
		"media_library",
		"view_exports",
		"resume_imports",
		"custom_content",
		"site_settings",
		"testimonials",
	}

	var all []services.MediaItem
	referenced := make(map[string]struct{})
	var totalSize int64

	for _, name := range collections {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			continue
		}

		fileFields := fileFieldNames(collection)
		if len(fileFields) == 0 {
			continue
		}

		// Avoid relying on created/updated columns because older seeded data may not include them.
		records, err := app.FindRecordsByFilter(collection.Name, "", "", 500, 0, nil)
		if err != nil {
			continue
		}

		for _, record := range records {
			created := record.GetDateTime("created")
			createdAt := created.Time()
			for _, field := range fileFields {
				values := services.FlattenFileValue(record.Get(field))
				for _, filename := range values {
					item, err := services.BuildMediaItem(storage, collection.Name, collection.Id, record.Id, field, filename, createdAt)
					if err != nil {
						app.Logger().Warn("media: failed to build media item",
							"collection", collection.Name,
							"record_id", record.Id,
							"field", field,
							"filename", filename,
							"error", err)
						continue
					}

					// Set record label (for "used by" display)
					recordLabel := record.GetString("title")
					if recordLabel == "" {
						recordLabel = record.GetString("name")
					}
					if recordLabel == "" {
						recordLabel = record.Id
					}
					item.RecordLabel = recordLabel

					// For uploads collection, check actual usage via library URL fields
					// For other collections, the file is inherently "used" by its owning record
					if collection.Name == "uploads" {
						usage := findLibraryURLUsage(app, item.URL)
						item.UsageCount = usage.UsageCount
						item.UsedBy = usage.UsedBy
						// Get alt_text, description, display_name for uploads from record
						item.AltText = record.GetString("alt_text")
						item.Description = record.GetString("description")
						customName := record.GetString("title")
						if customName != "" {
							item.DisplayName = customName
						}
					} else {
						item.UsageCount = 1
						item.UsedBy = []services.MediaUsageItem{{
							Collection: collection.Name,
							RecordID:   record.Id,
							Title:      recordLabel,
							Slug:       record.GetString("slug"),
						}}
						// For other collections, get metadata from media_display_names
						if metadata := services.GetMediaMetadata(app, collection.Id, record.Id, filename); metadata != nil {
							if metadata.DisplayName != "" {
								item.DisplayName = metadata.DisplayName
							}
							item.AltText = metadata.AltText
							item.Description = metadata.Description
							// Fetch tags from media_display_names
							if len(metadata.TagIDs) > 0 {
								item.Tags = make([]services.MediaTag, 0, len(metadata.TagIDs))
								for _, tagID := range metadata.TagIDs {
									tagRecord, err := app.FindRecordById("admin_tags", tagID)
									if err != nil {
										continue
									}
									item.Tags = append(item.Tags, services.MediaTag{
										ID:    tagRecord.Id,
										Name:  tagRecord.GetString("name"),
										Color: tagRecord.GetString("color"),
									})
								}
							}
						}
						// Fall back to record title if no custom display name
						if item.DisplayName == "" {
							if recordTitle := record.GetString("title"); recordTitle != "" {
								item.DisplayName = recordTitle
							} else if recordName := record.GetString("name"); recordName != "" {
								item.DisplayName = recordName
							}
						}
					}

					// Fetch tags for uploads collection (from record's admin_tags field)
					if collection.Name == "uploads" && collection.Fields.GetByName("admin_tags") != nil {
						item.Tags = fetchMediaTags(app, record)
					}
					all = append(all, item)
					key := filepath.ToSlash(filepath.Join(collection.Id, record.Id, filename))
					referenced[key] = struct{}{}
					totalSize += item.Size
				}
			}
		}
	}

	return all, referenced, totalSize, nil
}

func collectExternalMediaItems(app *pocketbase.PocketBase) ([]services.MediaItem, error) {
	collection, err := app.FindCollectionByNameOrId("external_media")
	if err != nil {
		// Collection might not exist yet - this is not an error condition
		return []services.MediaItem{}, nil
	}

	// Get all records from external_media collection
	records, err := app.FindAllRecords(collection.Name)
	if err != nil {
		return []services.MediaItem{}, err
	}

	items := make([]services.MediaItem, 0, len(records))
	for _, record := range records {
		recordURL := record.GetString("url")

		// Skip "mirror" entries that point to internal PocketBase files
		// These are created when attaching uploads to posts/projects/talks via media_refs
		// and appear as duplicates in the media library
		if strings.Contains(recordURL, "/api/files/") {
			continue
		}

		created := record.GetDateTime("created").Time()
		title := record.GetString("title")
		if title == "" {
			title = recordURL
		}
		normalized := mediaembed.Normalize(recordURL, record.GetString("mime"), record.GetString("thumbnail_url"))

		// Get usage info for this external media item
		usage, _ := services.FindMediaUsage(app, record.Id)

		item := services.MediaItem{
			Collection:    collection.Name,
			CollectionID:  collection.Id,
			CollectionKey: "external",
			RecordID:      record.Id,
			Field:         "external",
			Filename:      title,
			DisplayName:   title,
			RecordLabel:   title,
			URL:           record.GetString("url"),
			Mime:          normalized.Mime,
			ThumbnailURL:  normalized.ThumbnailURL,
			EmbedURL:      normalized.EmbedURL,
			Provider:      normalized.Provider,
			UploadedAt:    created,
			External:      true,
			AltText:       record.GetString("alt_text"),
			Description:   record.GetString("description"),
			UsageCount:    usage.UsageCount,
			UsedBy:        usage.UsedBy,
			Tags:          fetchMediaTags(app, record),
		}
		items = append(items, item)
	}
	return items, nil
}

// collectOrphanMediaItems scans all storage locations for files not referenced by any record.
func collectOrphanMediaItems(app *pocketbase.PocketBase, storage *services.StorageService, referenced map[string]struct{}) ([]services.MediaItem, int64, int64, int, error) {
	orphans := make([]services.MediaItem, 0)
	var totalOrphanSize int64
	var totalStorageSize int64
	var totalStorageFiles int

	// Scan all storage roots (primary and fallback)
	for _, storageRoot := range storage.GetStorageRoots() {
		// Check if storage directory exists
		if _, err := os.Stat(storageRoot); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(storageRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// Skip inaccessible files/directories but continue walking
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".attrs") {
				return nil
			}
			// Skip thumbnail files - they are generated artifacts, not orphans
			if strings.Contains(d.Name(), services.ThumbnailSuffix) {
				return nil
			}
			info, infoErr := d.Info()
			if infoErr == nil {
				totalStorageSize += info.Size()
				totalStorageFiles++
			}
			rel, err := filepath.Rel(storageRoot, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if _, ok := referenced[rel]; ok {
				return nil
			}

			parts := strings.Split(rel, "/")
			if len(parts) < 3 {
				return nil
			}
			collectionID := parts[0]
			recordID := parts[1]
			filename := strings.Join(parts[2:], "/")
			collectionName := collectionID
			if c, err := app.FindCollectionByNameOrId(collectionID); err == nil && c != nil {
				collectionName = c.Name
			}

			item, buildErr := services.BuildMediaItem(storage, collectionName, collectionID, recordID, "orphan", filename, time.Time{})
			if buildErr != nil {
				return nil
			}
			item.Orphan = true
			item.Field = "orphan"
			orphans = append(orphans, item)
			totalOrphanSize += item.Size
			return nil
		})
		if err != nil {
			// Log but continue with other roots
			continue
		}
	}

	return orphans, totalOrphanSize, totalStorageSize, totalStorageFiles, nil
}

func fileFieldNames(c *core.Collection) []string {
	var names []string
	for _, f := range c.Fields {
		if f.Type() == core.FieldTypeFile {
			names = append(names, f.GetName())
		}
	}
	return names
}

// resolveStoragePathMulti securely resolves a user-provided relative path across multiple storage roots.
// It finds the file in any configured storage location and validates the path for security.
func resolveStoragePathMulti(storage *services.StorageService, rel string) (string, error) {
	// 1. Validate input
	if rel == "" {
		return "", ErrInvalidPath
	}

	// 2. Clean the user input (handles .., //, platform separators)
	clean := filepath.Clean(rel)

	// 3. Reject absolute paths (prevents /etc/passwd injection)
	if filepath.IsAbs(clean) {
		return "", ErrAbsolutePath
	}

	// 4. Remove leading slashes and backslashes
	clean = strings.TrimPrefix(clean, "/")
	clean = strings.TrimPrefix(clean, "\\")

	// 5. Remove "storage" prefix if present (backward compatibility)
	clean = strings.TrimPrefix(clean, "storage/")
	clean = strings.TrimPrefix(clean, "storage\\")

	// 6. Defense in depth: reject if still contains .. after cleaning
	if strings.Contains(clean, "..") {
		return "", ErrInvalidPath
	}

	// 7. Try to find the file in any storage root
	for _, storageRoot := range storage.GetStorageRoots() {
		target := filepath.Join(storageRoot, clean)

		// Validate containment using filepath.Rel
		relPath, err := filepath.Rel(storageRoot, target)
		if err != nil {
			continue
		}

		// Check if relative path tries to escape
		if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
			continue
		}

		// Additional containment check
		if !strings.HasPrefix(target, storageRoot+string(filepath.Separator)) && target != storageRoot {
			continue
		}

		// Check if path exists
		info, err := os.Lstat(target)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Try next root
			}
			continue
		}

		// Reject symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			return "", ErrSymlink
		}

		return target, nil
	}

	return "", fmt.Errorf("file not found in any storage location: %s", rel)
}

// resolveStoragePath is the legacy single-root version for backward compatibility.
// Deprecated: Use resolveStoragePathMulti with StorageService instead.
func resolveStoragePath(storageRoot, rel string) (string, error) {
	// 1. Validate inputs
	if storageRoot == "" || rel == "" {
		return "", ErrInvalidPath
	}

	// 2. Clean the user input (handles .., //, platform separators)
	clean := filepath.Clean(rel)

	// 3. Reject absolute paths (prevents /etc/passwd injection)
	if filepath.IsAbs(clean) {
		return "", ErrAbsolutePath
	}

	// 4. Remove leading slashes and backslashes
	clean = strings.TrimPrefix(clean, "/")
	clean = strings.TrimPrefix(clean, "\\")

	// 5. Remove "storage" prefix if present (backward compatibility)
	clean = strings.TrimPrefix(clean, "storage/")
	clean = strings.TrimPrefix(clean, "storage\\")

	// 6. Defense in depth: reject if still contains .. after cleaning
	// Protects against theoretical filepath.Clean bypasses (e.g., CVE-2022-41722)
	if strings.Contains(clean, "..") {
		return "", ErrInvalidPath
	}

	// 7. Join with storage root
	target := filepath.Join(storageRoot, clean)

	// 8. Validate containment using filepath.Rel (robust approach)
	relPath, err := filepath.Rel(storageRoot, target)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Check if relative path tries to escape
	if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
		return "", ErrPathEscapes
	}

	// 9. Additional containment check using prefix (defense in depth)
	if !strings.HasPrefix(target, storageRoot+string(filepath.Separator)) && target != storageRoot {
		return "", ErrPathEscapes
	}

	// 10. CRITICAL: Check if path is a symlink
	// Use Lstat (not Stat) to avoid following the symlink
	info, err := os.Lstat(target)
	if err != nil {
		// File doesn't exist - this is okay for some operations
		// (os.Remove will fail gracefully if file doesn't exist)
		if os.IsNotExist(err) {
			return target, nil
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	// 11. Reject symbolic links entirely
	// This prevents symlink attacks where a link points outside storage
	if info.Mode()&os.ModeSymlink != 0 {
		return "", ErrSymlink
	}

	return target, nil
}

func validateURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, os.ErrInvalid
	}
	return parsed, nil
}

// detectMimeType determines the MIME type of a file using extension and content sniffing.
func detectMimeType(fullPath, filename string) string {
	// Try extension first
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if mimeType != "" {
		return mimeType
	}

	// Fallback to content sniffing
	f, err := os.Open(fullPath)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	sniff := make([]byte, 512)
	n, _ := f.Read(sniff)
	if n > 0 {
		return http.DetectContentType(sniff[:n])
	}

	return "application/octet-stream"
}

// registerThumbnailHook registers thumbnail generation hooks for a collection.
// Generates thumbnails on both create and update (when new files are added).
func registerThumbnailHook(app *pocketbase.PocketBase, collectionName string, uploadsDir string) {
	// Hook for new records
	app.OnRecordAfterCreateSuccess(collectionName).BindFunc(func(e *core.RecordEvent) error {
		go generateThumbnailsForRecord(app, e.Record, uploadsDir)
		return e.Next()
	})

	// Hook for updated records (new files added)
	app.OnRecordAfterUpdateSuccess(collectionName).BindFunc(func(e *core.RecordEvent) error {
		// Only generate thumbnails for newly added files
		go generateThumbnailsForNewFiles(app, e.Record, uploadsDir)
		return e.Next()
	})
}

// generateThumbnailsForRecord generates thumbnails for all image files in a record.
func generateThumbnailsForRecord(app *pocketbase.PocketBase, record *core.Record, uploadsDir string) {
	storage := services.NewStorageService(services.StorageConfig{
		UploadsDir: uploadsDir,
		DataDir:    app.DataDir(),
	})
	thumbService := services.NewThumbnailService(storage)

	collection := record.Collection()
	fileFields := fileFieldNames(collection)

	for _, fieldName := range fileFields {
		filenames := services.FlattenFileValue(record.Get(fieldName))
		for _, filename := range filenames {
			generateThumbnailIfSupported(app, thumbService, storage, collection.Id, record.Id, filename)
		}
	}
}

// generateThumbnailsForNewFiles generates thumbnails only for newly added files.
func generateThumbnailsForNewFiles(app *pocketbase.PocketBase, record *core.Record, uploadsDir string) {
	storage := services.NewStorageService(services.StorageConfig{
		UploadsDir: uploadsDir,
		DataDir:    app.DataDir(),
	})
	thumbService := services.NewThumbnailService(storage)

	collection := record.Collection()
	fileFields := fileFieldNames(collection)

	for _, fieldName := range fileFields {
		oldFilenames := services.FlattenFileValue(record.Original().Get(fieldName))
		newFilenames := services.FlattenFileValue(record.Get(fieldName))

		// Build set of old filenames
		oldSet := make(map[string]struct{})
		for _, fn := range oldFilenames {
			oldSet[fn] = struct{}{}
		}

		// Generate thumbnails for newly added files only
		for _, filename := range newFilenames {
			if _, exists := oldSet[filename]; !exists {
				generateThumbnailIfSupported(app, thumbService, storage, collection.Id, record.Id, filename)
			}
		}
	}
}

// generateThumbnailIfSupported generates a thumbnail if the file is a supported image format.
func generateThumbnailIfSupported(app *pocketbase.PocketBase, thumbService *services.ThumbnailService, storage *services.StorageService, collectionID, recordID, filename string) {
	// Get the file path to determine mime type
	fullPath, found := storage.FindFile(collectionID, recordID, filename)
	if !found {
		app.Logger().Debug("thumbnail: file not found", "collection", collectionID, "record", recordID, "filename", filename)
		return
	}

	// Determine mime type
	mimeType := detectMimeType(fullPath, filename)

	// Only generate thumbnails for supported image formats
	if !services.IsSupportedFormat(mimeType) {
		return
	}

	// Generate thumbnail
	_, err := thumbService.GenerateThumbnail(collectionID, recordID, filename, services.ThumbnailSize)
	if err != nil {
		app.Logger().Warn("thumbnail: generation failed", "collection", collectionID, "record", recordID, "filename", filename, "error", err)
		return
	}

	app.Logger().Debug("thumbnail: generated", "collection", collectionID, "record", recordID, "filename", filename)
}
