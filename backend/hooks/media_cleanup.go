package hooks

import (
	"net/http"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// collectionsWithFileFields lists all collections that have file fields
// and may have entries in the media_display_names table.
var collectionsWithFileFields = []string{
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

// RegisterMediaCleanupHooks registers hooks for cleaning up media_display_names
// when records or files are deleted from collections with file fields.
func RegisterMediaCleanupHooks(app *pocketbase.PocketBase) {
	// Register post-delete hooks for all collections with file fields
	// When a record is deleted, clean up any media_display_names entries for its files
	for _, collName := range collectionsWithFileFields {
		registerDeleteHook(app, collName)
	}

	// Register post-update hooks to detect file removals
	// When a file is removed from a record, clean up its media_display_names entry
	for _, collName := range collectionsWithFileFields {
		registerUpdateHook(app, collName)
	}

	// Admin endpoint to clean up orphaned media_display_names entries
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/admin/cleanup-media-metadata", func(e *core.RequestEvent) error {
			cleaned := cleanupOrphanedMediaMetadata(app)
			return e.JSON(http.StatusOK, map[string]any{
				"status":  "ok",
				"cleaned": cleaned,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// registerDeleteHook registers a post-delete hook for a collection
func registerDeleteHook(app *pocketbase.PocketBase, collectionName string) {
	app.OnRecordAfterDeleteSuccess(collectionName).BindFunc(func(e *core.RecordEvent) error {
		collection := e.Record.Collection()
		recordID := e.Record.Id

		// Get all file field names from this collection
		fileFields := getFileFieldNames(collection)
		if len(fileFields) == 0 {
			return e.Next()
		}

		// For each file field, get the filenames and delete their metadata
		for _, fieldName := range fileFields {
			filenames := services.FlattenFileValue(e.Record.Get(fieldName))
			for _, filename := range filenames {
				if err := services.DeleteMediaDisplayName(app, collection.Id, recordID, filename); err != nil {
					app.Logger().Warn("media_cleanup: failed to delete metadata",
						"collection", collectionName,
						"record_id", recordID,
						"filename", filename,
						"error", err)
					// Continue with other files
				}
			}
		}

		app.Logger().Debug("media_cleanup: cleaned up metadata for deleted record",
			"collection", collectionName,
			"record_id", recordID)

		return e.Next()
	})
}

// registerUpdateHook registers a post-update hook for a collection to detect file removals
func registerUpdateHook(app *pocketbase.PocketBase, collectionName string) {
	app.OnRecordAfterUpdateSuccess(collectionName).BindFunc(func(e *core.RecordEvent) error {
		collection := e.Record.Collection()
		recordID := e.Record.Id

		// Get all file field names from this collection
		fileFields := getFileFieldNames(collection)
		if len(fileFields) == 0 {
			return e.Next()
		}

		// For each file field, compare old vs new values to find removed files
		for _, fieldName := range fileFields {
			oldFilenames := services.FlattenFileValue(e.Record.Original().Get(fieldName))
			newFilenames := services.FlattenFileValue(e.Record.Get(fieldName))

			// Build a set of new filenames for efficient lookup
			newSet := make(map[string]struct{})
			for _, fn := range newFilenames {
				newSet[fn] = struct{}{}
			}

			// Find removed files and delete their metadata
			for _, oldFn := range oldFilenames {
				if _, exists := newSet[oldFn]; !exists {
					// File was removed
					if err := services.DeleteMediaDisplayName(app, collection.Id, recordID, oldFn); err != nil {
						app.Logger().Warn("media_cleanup: failed to delete metadata for removed file",
							"collection", collectionName,
							"record_id", recordID,
							"filename", oldFn,
							"error", err)
						// Continue with other files
					} else {
						app.Logger().Debug("media_cleanup: deleted metadata for removed file",
							"collection", collectionName,
							"record_id", recordID,
							"filename", oldFn)
					}
				}
			}
		}

		return e.Next()
	})
}

// getFileFieldNames returns all file field names from a collection
func getFileFieldNames(c *core.Collection) []string {
	var names []string
	for _, f := range c.Fields {
		if f.Type() == core.FieldTypeFile {
			names = append(names, f.GetName())
		}
	}
	return names
}

// cleanupOrphanedMediaMetadata finds and removes media_display_names entries
// that reference files or records that no longer exist.
func cleanupOrphanedMediaMetadata(app *pocketbase.PocketBase) []string {
	var cleaned []string

	// Get the media_display_names collection
	mdnCollection, err := app.FindCollectionByNameOrId("media_display_names")
	if err != nil {
		app.Logger().Warn("media_cleanup: media_display_names collection not found")
		return cleaned
	}

	// Get all media_display_names records with pagination
	var records []*core.Record
	const pageSize = 200
	for offset := 0; ; offset += pageSize {
		page, err := app.FindRecordsByFilter(mdnCollection.Name, "", "", pageSize, offset, nil)
		if err != nil {
			app.Logger().Warn("media_cleanup: failed to load media_display_names", "error", err)
			break
		}
		records = append(records, page...)
		if len(page) < pageSize {
			break
		}
	}

	for _, mdn := range records {
		collectionID := mdn.GetString("collection_id")
		recordID := mdn.GetString("record_id")
		filename := mdn.GetString("filename")

		// Check if the source collection exists
		sourceCollection, err := app.FindCollectionByNameOrId(collectionID)
		if err != nil {
			// Collection doesn't exist - orphaned entry
			if err := app.Delete(mdn); err != nil {
				app.Logger().Warn("media_cleanup: failed to delete orphaned metadata",
					"mdn_id", mdn.Id, "error", err)
			} else {
				cleaned = append(cleaned, "collection_missing:"+collectionID+"/"+recordID+"/"+filename)
			}
			continue
		}

		// Check if the source record exists
		sourceRecord, err := app.FindRecordById(sourceCollection.Name, recordID)
		if err != nil {
			// Record doesn't exist - orphaned entry
			if err := app.Delete(mdn); err != nil {
				app.Logger().Warn("media_cleanup: failed to delete orphaned metadata",
					"mdn_id", mdn.Id, "error", err)
			} else {
				cleaned = append(cleaned, "record_missing:"+collectionID+"/"+recordID+"/"+filename)
			}
			continue
		}

		// Check if the file still exists on the record
		fileFound := false
		fileFields := getFileFieldNames(sourceCollection)
		for _, fieldName := range fileFields {
			filenames := services.FlattenFileValue(sourceRecord.Get(fieldName))
			for _, fn := range filenames {
				if fn == filename {
					fileFound = true
					break
				}
			}
			if fileFound {
				break
			}
		}

		if !fileFound {
			// File doesn't exist on record anymore - orphaned entry
			if err := app.Delete(mdn); err != nil {
				app.Logger().Warn("media_cleanup: failed to delete orphaned metadata",
					"mdn_id", mdn.Id, "error", err)
			} else {
				cleaned = append(cleaned, "file_missing:"+collectionID+"/"+recordID+"/"+filename)
			}
		}
	}

	app.Logger().Info("media_cleanup: cleanup complete", "removed", len(cleaned))
	return cleaned
}
