package hooks

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"facet/services"

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

			var orphanItems []services.MediaItem
			var orphanSize int64
			if orphansOnly || includeOrphans {
				orphanItems, orphanSize, err = collectOrphanMediaItems(app, referenced)
				if err != nil {
					app.Logger().Error("media orphan scan failed", "error", err)
					return apis.NewBadRequestError("failed to enumerate media", err)
				}
			}

			combined := items
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
				if search != "" && !strings.Contains(strings.ToLower(item.Filename), search) {
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
					"referencedFiles": len(items),
					"referencedSize":  referencedSize,
					"orphanFiles":     len(orphanItems),
					"orphanSize":      orphanSize,
					"totalFiles":      len(items) + len(orphanItems),
					"totalSize":       referencedSize + orphanSize,
				},
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
				clean := filepath.Clean(req.RelativePath)
				if strings.Contains(clean, "..") {
					return apis.NewBadRequestError("invalid path", nil)
				}
				clean = strings.TrimPrefix(clean, "/")
				clean = strings.TrimPrefix(clean, "\\")
				clean = strings.TrimPrefix(clean, "storage/")
				clean = strings.TrimPrefix(clean, "storage\\")
				target := filepath.Join(storageRoot, clean)
				if !strings.HasPrefix(target, storageRoot) {
					return apis.NewBadRequestError("invalid path", nil)
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

func collectOrphanMediaItems(app *pocketbase.PocketBase, referenced map[string]struct{}) ([]services.MediaItem, int64, error) {
	dataDir := app.DataDir()
	storageRoot := filepath.Join(dataDir, "storage")
	var orphans []services.MediaItem
	var totalSize int64

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

	return orphans, totalSize, err
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
