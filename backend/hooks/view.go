package hooks

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterViewHooks registers view-related API endpoints
func RegisterViewHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Get view access info (for frontend to determine access)
		se.Router.GET("/api/view/{slug}/access", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			records, err := app.FindRecordsByFilter(
				"views",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(records) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}

			view := records[0]

			if !view.GetBool("is_active") {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}

			visibility := view.GetString("visibility")

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":                 view.Id,
				"slug":               slug,
				"visibility":         visibility,
				"requires_password":  visibility == "password",
				"requires_token":     visibility == "unlisted",
			})
		})

		// Get full view data (with content filtering based on sections config)
		se.Router.GET("/api/view/{slug}/data", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			records, err := app.FindRecordsByFilter(
				"views",
				"slug = {:slug} && is_active = true",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(records) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}

			view := records[0]
			visibility := view.GetString("visibility")

			// Check access based on visibility
			// For public views, allow access
			// For unlisted, check for valid token in header or query
			// For password, check for access token in header
			// For private, require admin auth

			if visibility == "private" {
				if e.Auth == nil {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				}
			}

			// Build view response
			response := map[string]interface{}{
				"id":           view.Id,
				"slug":         slug,
				"name":         view.GetString("name"),
				"visibility":   visibility,
			}

			// Apply overrides if present
			if headline := view.GetString("hero_headline"); headline != "" {
				response["hero_headline"] = headline
			}
			if summary := view.GetString("hero_summary"); summary != "" {
				response["hero_summary"] = summary
			}
			if ctaText := view.GetString("cta_text"); ctaText != "" {
				response["cta_text"] = ctaText
			}
			if ctaURL := view.GetString("cta_url"); ctaURL != "" {
				response["cta_url"] = ctaURL
			}

			// Get sections configuration
			sectionsJSON := view.GetString("sections")
			var sections []map[string]interface{}
			if sectionsJSON != "" {
				json.Unmarshal([]byte(sectionsJSON), &sections)
			}

			// Fetch content for each enabled section
			sectionData := make(map[string]interface{})

			for _, section := range sections {
				sectionName, ok := section["section"].(string)
				if !ok {
					continue
				}
				enabled, ok := section["enabled"].(bool)
				if !ok || !enabled {
					continue
				}

				items, ok := section["items"].([]interface{})
				collectionName := getCollectionName(sectionName)
				if collectionName == "" {
					continue
				}

				if ok && len(items) > 0 {
					// Fetch specific items
					var itemRecords []*core.Record
					for _, itemID := range items {
						if id, ok := itemID.(string); ok {
							record, err := app.FindRecordById(collectionName, id)
							if err == nil && isRecordVisible(record) {
								itemRecords = append(itemRecords, record)
							}
						}
					}
					sectionData[sectionName] = serializeRecords(itemRecords)
				} else {
					// Fetch all visible items from section
					records, err := app.FindRecordsByFilter(
						collectionName,
						"visibility != 'private' && is_draft = false",
						"sort_order",
						100,
						0,
						nil,
					)
					if err == nil {
						sectionData[sectionName] = serializeRecords(records)
					}
				}
			}

			response["sections"] = sectionData

			return e.JSON(http.StatusOK, response)
		})

		// Apply import proposal
		se.Router.POST("/api/proposals/{id}/apply", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			proposalID := e.Request.PathValue("id")
			proposal, err := app.FindRecordById("import_proposals", proposalID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "proposal not found"})
			}

			if proposal.GetString("status") != "pending" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "proposal already processed"})
			}

			var req struct {
				AppliedFields map[string]bool `json:"applied_fields"` // field -> should apply
				LockedFields  []string        `json:"locked_fields"`  // fields to lock
				Edits         map[string]interface{} `json:"edits"` // manual edits
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Get proposed data
			var proposedData map[string]interface{}
			if err := json.Unmarshal([]byte(proposal.GetString("proposed_data")), &proposedData); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid proposal data"})
			}

			// Apply edits
			for field, value := range req.Edits {
				proposedData[field] = value
			}

			// Get or create project
			projectID := proposal.GetString("project_id")
			var project *core.Record

			if projectID != "" {
				project, err = app.FindRecordById("projects", projectID)
				if err != nil {
					return e.JSON(http.StatusNotFound, map[string]string{"error": "project not found"})
				}
			} else {
				collection, err := app.FindCollectionByNameOrId("projects")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "projects collection not found"})
				}
				project = core.NewRecord(collection)
			}

			// Get existing field locks
			var fieldLocks map[string]bool
			existingLocks := project.GetString("field_locks")
			if existingLocks != "" {
				json.Unmarshal([]byte(existingLocks), &fieldLocks)
			}
			if fieldLocks == nil {
				fieldLocks = make(map[string]bool)
			}

			// Apply fields that are approved and not locked
			for field, shouldApply := range req.AppliedFields {
				if shouldApply && !fieldLocks[field] {
					if value, exists := proposedData[field]; exists {
						project.Set(field, value)
					}
				}
			}

			// Update field locks
			for _, field := range req.LockedFields {
				fieldLocks[field] = true
			}
			fieldLocksJSON, _ := json.Marshal(fieldLocks)
			project.Set("field_locks", string(fieldLocksJSON))

			// Link to source
			sourceID := proposal.GetString("source_id")
			if sourceID != "" {
				project.Set("source_id", sourceID)
			}

			if err := app.Save(project); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save project"})
			}

			// Update source with project link
			if sourceID != "" {
				source, _ := app.FindRecordById("sources", sourceID)
				if source != nil {
					source.Set("project_id", project.Id)
					source.Set("sync_status", "success")
					app.Save(source)
				}
			}

			// Mark proposal as applied
			appliedJSON, _ := json.Marshal(req.AppliedFields)
			proposal.Set("status", "applied")
			proposal.Set("applied_fields", string(appliedJSON))
			app.Save(proposal)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"project_id": project.Id,
				"status":     "applied",
			})
		}).Bind(apis.RequireAuth())

		// Reject import proposal
		se.Router.POST("/api/proposals/{id}/reject", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			proposalID := e.Request.PathValue("id")
			proposal, err := app.FindRecordById("import_proposals", proposalID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "proposal not found"})
			}

			proposal.Set("status", "rejected")
			app.Save(proposal)

			return e.JSON(http.StatusOK, map[string]string{"status": "rejected"})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

func getCollectionName(section string) string {
	switch section {
	case "experience":
		return "experience"
	case "projects":
		return "projects"
	case "education":
		return "education"
	case "certifications":
		return "certifications"
	case "skills":
		return "skills"
	case "posts":
		return "posts"
	case "talks":
		return "talks"
	default:
		return ""
	}
}

func isRecordVisible(record *core.Record) bool {
	visibility := record.GetString("visibility")
	isDraft := record.GetBool("is_draft")
	return visibility != "private" && !isDraft
}

func serializeRecords(records []*core.Record) []map[string]interface{} {
	var result []map[string]interface{}
	for _, record := range records {
		item := make(map[string]interface{})
		for key, value := range record.ColumnValueMap() {
			// Skip sensitive fields
			if key == "password_hash" {
				continue
			}
			item[key] = value
		}
		item["id"] = record.Id
		result = append(result, item)
	}
	return result
}
