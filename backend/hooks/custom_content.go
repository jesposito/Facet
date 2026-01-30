package hooks

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterCustomContentHooks registers hooks for custom_content collection cleanup
func RegisterCustomContentHooks(app *pocketbase.PocketBase) {
	// Clean up references when custom_content is deleted
	app.OnRecordAfterDeleteSuccess("custom_content").BindFunc(func(e *core.RecordEvent) error {
		deletedId := e.Record.Id
		sectionKey := "custom:" + deletedId

		app.Logger().Info("custom_content: cleaning up references", "id", deletedId)

		// Clean up references in all views
		if err := cleanupViewReferences(app, sectionKey); err != nil {
			app.Logger().Warn("custom_content: failed to clean view references", "id", deletedId, "error", err)
			// Don't fail the deletion, just log the warning
		}

		// Clean up references in site_settings (homepage)
		if err := cleanupSiteSettingsReferences(app, sectionKey); err != nil {
			app.Logger().Warn("custom_content: failed to clean site_settings references", "id", deletedId, "error", err)
			// Don't fail the deletion, just log the warning
		}

		return e.Next()
	})

	// Admin endpoint to clean up orphaned custom content references
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/admin/cleanup-custom-content", func(e *core.RequestEvent) error {
			cleaned := cleanupOrphanedCustomContentRefs(app)
			return e.JSON(http.StatusOK, map[string]any{
				"status":  "ok",
				"cleaned": cleaned,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// cleanupViewReferences removes the custom content section from all views
func cleanupViewReferences(app *pocketbase.PocketBase, sectionKey string) error {
	viewsCollection, err := app.FindCollectionByNameOrId("views")
	if err != nil {
		return nil // Collection doesn't exist, nothing to clean
	}

	views, err := app.FindAllRecords(viewsCollection.Name)
	if err != nil {
		return err
	}

	for _, view := range views {
		sectionsRaw := view.Get("sections")
		if sectionsRaw == nil {
			continue
		}

		// Parse sections JSON
		sectionsBytes, err := json.Marshal(sectionsRaw)
		if err != nil {
			continue
		}

		var sections []map[string]any
		if err := json.Unmarshal(sectionsBytes, &sections); err != nil {
			continue
		}

		// Filter out the deleted custom content section
		modified := false
		filtered := make([]map[string]any, 0, len(sections))
		for _, section := range sections {
			if sectionName, ok := section["section"].(string); ok && sectionName == sectionKey {
				modified = true
				continue // Skip this section
			}
			filtered = append(filtered, section)
		}

		if modified {
			view.Set("sections", filtered)
			if err := app.Save(view); err != nil {
				app.Logger().Warn("custom_content: failed to save view after cleanup",
					"view_id", view.Id, "error", err)
			} else {
				app.Logger().Debug("custom_content: removed reference from view",
					"view_id", view.Id, "section", sectionKey)
			}
		}
	}

	return nil
}

// cleanupSiteSettingsReferences removes the custom content from homepage settings
func cleanupSiteSettingsReferences(app *pocketbase.PocketBase, sectionKey string) error {
	settingsCollection, err := app.FindCollectionByNameOrId("site_settings")
	if err != nil {
		return nil // Collection doesn't exist, nothing to clean
	}

	settings, err := app.FindAllRecords(settingsCollection.Name)
	if err != nil {
		return err
	}

	for _, setting := range settings {
		modified := false

		// Clean homepage_section_order
		orderRaw := setting.Get("homepage_section_order")
		if orderRaw != nil {
			orderBytes, err := json.Marshal(orderRaw)
			if err == nil {
				var order []string
				if json.Unmarshal(orderBytes, &order) == nil {
					filteredOrder := make([]string, 0, len(order))
					for _, key := range order {
						if key != sectionKey {
							filteredOrder = append(filteredOrder, key)
						} else {
							modified = true
						}
					}
					if modified {
						setting.Set("homepage_section_order", filteredOrder)
					}
				}
			}
		}

		// Clean homepage_sections
		sectionsRaw := setting.Get("homepage_sections")
		if sectionsRaw != nil {
			sectionsBytes, err := json.Marshal(sectionsRaw)
			if err == nil {
				var sections map[string]any
				if json.Unmarshal(sectionsBytes, &sections) == nil {
					if _, exists := sections[sectionKey]; exists {
						delete(sections, sectionKey)
						setting.Set("homepage_sections", sections)
						modified = true
					}
				}
			}
		}

		// Clean homepage_custom_content (array of objects with id field)
		customContentRaw := setting.Get("homepage_custom_content")
		if customContentRaw != nil {
			contentBytes, err := json.Marshal(customContentRaw)
			if err == nil {
				var items []map[string]any
				if json.Unmarshal(contentBytes, &items) == nil {
					// Extract the ID from sectionKey (remove "custom:" prefix)
					customId := strings.TrimPrefix(sectionKey, "custom:")
					filteredItems := make([]map[string]any, 0, len(items))
					for _, item := range items {
						if id, ok := item["id"].(string); ok && id == customId {
							modified = true
							continue
						}
						filteredItems = append(filteredItems, item)
					}
					if modified {
						setting.Set("homepage_custom_content", filteredItems)
					}
				}
			}
		}

		if modified {
			if err := app.Save(setting); err != nil {
				app.Logger().Warn("custom_content: failed to save site_settings after cleanup",
					"error", err)
			} else {
				app.Logger().Debug("custom_content: removed reference from site_settings",
					"section", sectionKey)
			}
		}
	}

	return nil
}

// cleanupOrphanedCustomContentRefs finds and removes references to custom content that no longer exists
func cleanupOrphanedCustomContentRefs(app *pocketbase.PocketBase) []string {
	var cleaned []string

	// Get all valid custom content IDs
	validIds := make(map[string]struct{})
	customContentCollection, err := app.FindCollectionByNameOrId("custom_content")
	if err == nil {
		records, err := app.FindAllRecords(customContentCollection.Name)
		if err == nil {
			for _, record := range records {
				validIds[record.Id] = struct{}{}
			}
		}
	}

	// Clean up views
	viewsCollection, err := app.FindCollectionByNameOrId("views")
	if err == nil {
		views, err := app.FindAllRecords(viewsCollection.Name)
		if err == nil {
			for _, view := range views {
				sectionsRaw := view.Get("sections")
				if sectionsRaw == nil {
					continue
				}

				sectionsBytes, err := json.Marshal(sectionsRaw)
				if err != nil {
					continue
				}

				var sections []map[string]any
				if err := json.Unmarshal(sectionsBytes, &sections); err != nil {
					continue
				}

				modified := false
				filtered := make([]map[string]any, 0, len(sections))
				for _, section := range sections {
					sectionName, ok := section["section"].(string)
					if !ok {
						filtered = append(filtered, section)
						continue
					}

					if strings.HasPrefix(sectionName, "custom:") {
						customId := strings.TrimPrefix(sectionName, "custom:")
						if _, exists := validIds[customId]; !exists {
							modified = true
							cleaned = append(cleaned, "view:"+view.Id+":"+sectionName)
							continue // Skip orphaned reference
						}
					}
					filtered = append(filtered, section)
				}

				if modified {
					view.Set("sections", filtered)
					if err := app.Save(view); err != nil {
						app.Logger().Warn("cleanup: failed to save view", "id", view.Id, "error", err)
					}
				}
			}
		}
	}

	// Clean up site_settings
	settingsCollection, err := app.FindCollectionByNameOrId("site_settings")
	if err == nil {
		settings, err := app.FindAllRecords(settingsCollection.Name)
		if err == nil {
			for _, setting := range settings {
				modified := false

				// Clean homepage_section_order
				orderRaw := setting.Get("homepage_section_order")
				if orderRaw != nil {
					orderBytes, _ := json.Marshal(orderRaw)
					var order []string
					if json.Unmarshal(orderBytes, &order) == nil {
						filteredOrder := make([]string, 0, len(order))
						for _, key := range order {
							if strings.HasPrefix(key, "custom:") {
								customId := strings.TrimPrefix(key, "custom:")
								if _, exists := validIds[customId]; !exists {
									modified = true
									cleaned = append(cleaned, "homepage_order:"+key)
									continue
								}
							}
							filteredOrder = append(filteredOrder, key)
						}
						if modified {
							setting.Set("homepage_section_order", filteredOrder)
						}
					}
				}

				// Clean homepage_sections
				sectionsRaw := setting.Get("homepage_sections")
				if sectionsRaw != nil {
					sectionsBytes, _ := json.Marshal(sectionsRaw)
					var sections map[string]any
					if json.Unmarshal(sectionsBytes, &sections) == nil {
						for key := range sections {
							if strings.HasPrefix(key, "custom:") {
								customId := strings.TrimPrefix(key, "custom:")
								if _, exists := validIds[customId]; !exists {
									delete(sections, key)
									modified = true
									cleaned = append(cleaned, "homepage_sections:"+key)
								}
							}
						}
						if modified {
							setting.Set("homepage_sections", sections)
						}
					}
				}

				// Clean homepage_custom_content
				customContentRaw := setting.Get("homepage_custom_content")
				if customContentRaw != nil {
					contentBytes, _ := json.Marshal(customContentRaw)
					var items []map[string]any
					if json.Unmarshal(contentBytes, &items) == nil {
						filteredItems := make([]map[string]any, 0, len(items))
						for _, item := range items {
							if id, ok := item["id"].(string); ok {
								if _, exists := validIds[id]; !exists {
									modified = true
									cleaned = append(cleaned, "homepage_custom_content:"+id)
									continue
								}
							}
							filteredItems = append(filteredItems, item)
						}
						if modified {
							setting.Set("homepage_custom_content", filteredItems)
						}
					}
				}

				if modified {
					if err := app.Save(setting); err != nil {
						app.Logger().Warn("cleanup: failed to save site_settings", "error", err)
					}
				}
			}
		}
	}

	app.Logger().Info("custom_content: cleanup complete", "removed", len(cleaned))
	return cleaned
}
