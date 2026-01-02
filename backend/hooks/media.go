package hooks

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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

			items, err := collectMediaItems(app)
			if err != nil {
				app.Logger().Error("media list failed", "error", err)
				return apis.NewBadRequestError("failed to enumerate media", err)
			}

			query := e.Request.URL.Query()
			search := strings.TrimSpace(strings.ToLower(query.Get("q")))
			typeFilter := strings.ToLower(strings.TrimSpace(query.Get("type"))) // "image" or ""
			collectionFilter := strings.TrimSpace(strings.ToLower(query.Get("collection")))

			filtered := make([]services.MediaItem, 0, len(items))
			for _, item := range items {
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
			}

			return e.JSON(http.StatusOK, response)
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/api/media", func(e *core.RequestEvent) error {
			var req struct {
				CollectionID string `json:"collection_id"`
				RecordID     string `json:"record_id"`
				Field        string `json:"field"`
				Filename     string `json:"filename"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
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

func collectMediaItems(app *pocketbase.PocketBase) ([]services.MediaItem, error) {
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
		"view_exports",
	}

	var all []services.MediaItem

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

		records, err := app.FindRecordsByFilter(collection.Name, "", "-created", 500, 0, nil)
		if err != nil {
			app.Logger().Warn("media: failed to load records", "collection", collection.Name, "error", err)
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
						app.Logger().Warn("media: failed to build item", "collection", collection.Name, "record", record.Id, "field", field, "file", filename, "error", err)
						continue
					}
					all = append(all, item)
				}
			}
		}
	}

	return all, nil
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
