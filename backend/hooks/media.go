package hooks

import (
	"errors"
	"fmt"
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

// RegisterMediaHooks exposes admin-only media listing and deletion endpoints.
func RegisterMediaHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/media", func(e *core.RequestEvent) error {
			// Auth is enforced by middleware; log principal
			if e.Auth != nil {
				app.Logger().Debug("media list auth ok", "id", e.Auth.Id, "email", e.Auth.Email())
			}

			query := e.Request.URL.Query()
			includeOrphans := strings.TrimSpace(strings.ToLower(query.Get("includeOrphans"))) == "1"
			orphansOnly := strings.TrimSpace(strings.ToLower(query.Get("orphans"))) == "1"

			items, referenced, referencedSize, err := collectMediaItems(app)
			if err != nil {
				app.Logger().Error("media list failed", "error", err)
				return apis.NewBadRequestError("failed to enumerate media", err)
			}

			externalItems, err := collectExternalMediaItems(app)
			if err != nil {
				externalItems = []services.MediaItem{}
			}

			orphanItems, orphanSize, storageSize, storageFiles, err := collectOrphanMediaItems(app, referenced)
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
			typeFilter := strings.ToLower(strings.TrimSpace(query.Get("type"))) // "image" or ""
			collectionFilter := strings.TrimSpace(strings.ToLower(query.Get("collection")))

			filtered := make([]services.MediaItem, 0, len(combined))
			for _, item := range combined {
				if search != "" && !strings.Contains(strings.ToLower(item.Filename), search) && !strings.Contains(strings.ToLower(item.DisplayName), search) {
					continue
				}
				if collectionFilter != "" && strings.ToLower(item.Collection) != collectionFilter && strings.ToLower(item.CollectionKey) != collectionFilter {
					continue
				}
				if typeFilter == "image" && !strings.HasPrefix(item.Mime, "image/") {
					continue
				}
				filtered = append(filtered, item)
			}

			// Sort by uploaded_at desc
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].UploadedAt.After(filtered[j].UploadedAt)
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
			sampleItems := make([]map[string]interface{}, 0, 3)
			for i := 0; i < len(filtered) && i < 3; i++ {
				item := filtered[i]
				sampleItems = append(sampleItems, map[string]interface{}{
					"collection":   item.Collection,
					"filename":     item.Filename,
					"external":     item.External,
					"orphan":       item.Orphan,
					"usage_count":  item.UsageCount,
					"used_by_len":  len(item.UsedBy),
				})
			}

			response := map[string]interface{}{
				"items":      filtered[start:end],
				"page":       page,
				"perPage":    perPage,
				"totalItems": total,
				"totalPages": (total + perPage - 1) / perPage,
				"debug": map[string]interface{}{
					"internalCount":  len(items),
					"externalCount":  len(externalItems),
					"orphanCount":    len(orphanItems),
					"combinedCount":  len(combined),
					"filteredCount":  len(filtered),
					"referencedKeys": len(referenced),
					"sampleItems":    sampleItems,
				},
				"stats": map[string]interface{}{
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
				return e.JSON(http.StatusConflict, map[string]interface{}{
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

			response := map[string]interface{}{
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

			// Orphan deletion path: delete by relative path under /storage
			if req.RelativePath != "" && (req.CollectionID == "" || req.Field == "") {
				dataDir := app.DataDir()
				storageRoot := filepath.Join(dataDir, "storage")
				target, err := resolveStoragePath(storageRoot, req.RelativePath)
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
			updated, removed := services.RemoveFileFromValue(current, req.Filename)
			if !removed {
				return apis.NewBadRequestError("file not found on record", nil)
			}

			record.Set(req.Field, updated)
			if err := app.Save(record); err != nil {
				app.Logger().Error("media delete failed to update record", "error", err)
				return apis.NewBadRequestError("failed to update record", err)
			}

			// Remove file from storage (ignore errors to avoid blocking user if already missing)
			dataDir := app.DataDir()
			_ = os.Remove(filepath.Join(dataDir, "storage", collection.Id, record.Id, req.Filename))

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		}).Bind(apis.RequireAuth())

		// Update display name for any media file
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

		dataDir := app.DataDir()
		storageRoot := filepath.Join(dataDir, "storage")

		deleted := 0
		failed := 0
		var errors []map[string]string

		for _, relativePath := range req.Orphans {
			target, err := resolveStoragePath(storageRoot, relativePath)
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

		response := map[string]interface{}{
			"deleted": deleted,
			"failed":  failed,
			"errors":  errors,
		}

		return e.JSON(http.StatusOK, response)
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

func collectMediaItems(app *pocketbase.PocketBase) ([]services.MediaItem, map[string]struct{}, int64, error) {
	dataDir := app.DataDir()

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
		"view_exports",
		"resume_imports",
		"custom_content",
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
					item, err := services.BuildMediaItem(dataDir, collection.Name, collection.Id, record.Id, field, filename, createdAt)
					if err != nil {
						continue
					}

					// Check for custom display name first
					customName := services.GetMediaDisplayName(app, collection.Id, record.Id, filename)
					if customName != "" {
						item.DisplayName = customName
					} else {
						// Fall back to record title for uploads collection
						recordTitle := record.GetString("title")
						if recordTitle == "" {
							recordTitle = record.GetString("name")
						}
						if recordTitle != "" {
							item.DisplayName = recordTitle
						}
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
					// For internal media, set usage to the owning record
					item.UsageCount = 1
					item.UsedBy = []services.MediaUsageItem{{
						Collection: collection.Name,
						RecordID:   record.Id,
						Title:      recordLabel,
						Slug:       record.GetString("slug"),
					}}
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
		created := record.GetDateTime("created").Time()
		title := record.GetString("title")
		if title == "" {
			title = record.GetString("url")
		}
		normalized := mediaembed.Normalize(record.GetString("url"), record.GetString("mime"), record.GetString("thumbnail_url"))

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
			UsageCount:    usage.UsageCount,
			UsedBy:        usage.UsedBy,
		}
		items = append(items, item)
	}
	return items, nil
}

func collectOrphanMediaItems(app *pocketbase.PocketBase, referenced map[string]struct{}) ([]services.MediaItem, int64, int64, int, error) {
	dataDir := app.DataDir()
	storageRoot := filepath.Join(dataDir, "storage")
	orphans := make([]services.MediaItem, 0)
	var totalSize int64
	var storageSize int64
	var storageFiles int

	// Check if storage directory exists
	if _, err := os.Stat(storageRoot); os.IsNotExist(err) {
		// No storage directory means no files and no orphans
		return orphans, 0, 0, 0, nil
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
		info, infoErr := d.Info()
		if infoErr == nil {
			storageSize += info.Size()
			storageFiles++
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

		item, buildErr := services.BuildMediaItem(dataDir, collectionName, collectionID, recordID, "orphan", filename, time.Time{})
		if buildErr != nil {
			return nil
		}
		item.Orphan = true
		item.Field = "orphan"
		orphans = append(orphans, item)
		totalSize += item.Size
		return nil
	})

	return orphans, totalSize, storageSize, storageFiles, err
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

// resolveStoragePath securely resolves a user-provided relative path within the storage root directory.
// It prevents:
// - Path traversal attacks (../ sequences)
// - Symlink attacks (links pointing outside storage)
// - Absolute path injection
// - Platform-specific path bypass techniques
//
// This function implements defense-in-depth with multiple validation layers:
// 1. Input validation (non-empty, non-absolute)
// 2. Path cleaning (normalize separators, remove ..)
// 3. Containment validation (must remain within storage root)
// 4. Symlink detection (reject symbolic links)
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
