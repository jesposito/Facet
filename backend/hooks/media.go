package hooks

import (
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
				app.Logger().Error("media external scan failed", "error", err)
				externalItems = nil
			}

			orphanItems, orphanSize, storageSize, storageFiles, err := collectOrphanMediaItems(app, referenced)
			if err != nil {
				app.Logger().Error("media orphan scan failed", "error", err)
				orphanItems = nil
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

			response := map[string]interface{}{
				"items":      filtered[start:end],
				"page":       page,
				"perPage":    perPage,
				"totalItems": total,
				"totalPages": (total + perPage - 1) / perPage,
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
			collection, err := app.FindCollectionByNameOrId("external_media")
			if err != nil {
				return apis.NewBadRequestError("external media not configured", err)
			}
			record, err := app.FindRecordById(collection.Name, id)
			if err != nil {
				return apis.NewNotFoundError("not found", err)
			}
			if err := app.Delete(record); err != nil {
				return apis.NewBadRequestError("failed to delete external media", err)
			}
			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
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
					return apis.NewBadRequestError("invalid path", err)
				}
				if err := os.Remove(target); err != nil {
					app.Logger().Warn("media: failed to delete orphan file", "path", target, "error", err)
					return apis.NewBadRequestError("failed to delete file", err)
				}
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

	app.Logger().Info("media: collecting files")

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
	}

	var all []services.MediaItem
	referenced := make(map[string]struct{})
	var totalSize int64

	for _, name := range collections {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			app.Logger().Warn("media: collection not found", "collection", name, "error", err)
			continue
		}

		fileFields := fileFieldNames(collection)
		if len(fileFields) == 0 {
			app.Logger().Debug("media: no file fields", "collection", collection.Name)
			continue
		}

		// Avoid relying on created/updated columns because older seeded data may not include them.
		records, err := app.FindRecordsByFilter(collection.Name, "", "", 500, 0, nil)
		if err != nil {
			app.Logger().Warn("media: failed to load records", "collection", collection.Name, "error", err)
			continue
		}

		app.Logger().Info("media: collection scan", "collection", collection.Name, "records", len(records), "fileFields", fileFields)

		for _, record := range records {
			created := record.GetDateTime("created")
			createdAt := created.Time()
			for _, field := range fileFields {
				values := services.FlattenFileValue(record.Get(field))
				for _, filename := range values {
					item, err := services.BuildMediaItem(dataDir, collection.Name, collection.Id, record.Id, field, filename, createdAt)
					if err != nil {
						app.Logger().Warn("media: failed to build item", "collection", collection.Name, "record", record.Id, "field", field, "file", filename, "error", err)
						continue
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
		return nil, nil
	}

	records, err := app.FindRecordsByFilter(collection.Name, "", "-created", 500, 0, nil)
	if err != nil {
		return nil, err
	}

	var items []services.MediaItem
	for _, record := range records {
		created := record.GetDateTime("created").Time()
		title := record.GetString("title")
		if title == "" {
			title = record.GetString("url")
		}
		normalized := mediaembed.Normalize(record.GetString("url"), record.GetString("mime"), record.GetString("thumbnail_url"))
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
		}
		items = append(items, item)
	}
	return items, nil
}

func collectOrphanMediaItems(app *pocketbase.PocketBase, referenced map[string]struct{}) ([]services.MediaItem, int64, int64, int, error) {
	dataDir := app.DataDir()
	storageRoot := filepath.Join(dataDir, "storage")
	var orphans []services.MediaItem
	var totalSize int64
	var storageSize int64
	var storageFiles int

	err := filepath.WalkDir(storageRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
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
			app.Logger().Warn("media: failed to build orphan item", "path", rel, "error", buildErr)
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

func resolveStoragePath(storageRoot, rel string) (string, error) {
	clean := filepath.Clean(rel)
	if strings.Contains(clean, "..") {
		return "", os.ErrInvalid
	}
	clean = strings.TrimPrefix(clean, "/")
	clean = strings.TrimPrefix(clean, "\\")
	clean = strings.TrimPrefix(clean, "storage/")
	clean = strings.TrimPrefix(clean, "storage\\")
	target := filepath.Join(storageRoot, clean)
	if !strings.HasPrefix(target, storageRoot) {
		return "", os.ErrInvalid
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
