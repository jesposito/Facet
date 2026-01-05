package hooks

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// Simple rate limiter for AI resume generation
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int           // max requests per window
	window   time.Duration // time window
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old requests
	var valid []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.requests[key] = valid

	// Check limit
	if len(valid) >= rl.limit {
		return false
	}

	// Add this request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// RegisterResumeHooks registers AI Print (resume generation) endpoints
func RegisterResumeHooks(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	ai := services.NewAIService(crypto)
	resume := services.NewResumeService(ai)
	limiter := newRateLimiter(5, time.Hour) // 5 generations per hour per IP

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// Check if AI Print is available (Pandoc installed + AI provider configured)
		// GET /api/ai-print/status
		// Public endpoint - just returns capability info, no sensitive data
		se.Router.GET("/api/ai-print/status", func(e *core.RequestEvent) error {
			pandocAvailable := resume.CheckPandocAvailable()

			// Check if any AI provider is configured
			providers, err := app.FindRecordsByFilter("ai_providers", "is_active = true", "", 1, 0, nil)
			aiAvailable := err == nil && len(providers) > 0

			return e.JSON(http.StatusOK, map[string]interface{}{
				"available":         pandocAvailable && aiAvailable,
				"pandoc_installed":  pandocAvailable,
				"ai_configured":     aiAvailable,
				"supported_formats": []string{"pdf", "docx"},
			})
		}) // No auth required - public capability check

		// Generate a resume from a view
		// POST /api/view/{slug}/generate
		// Public endpoint - allows recruiters to generate resumes from public views
		se.Router.POST("/api/view/{slug}/generate", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			// Rate limit by IP (skip for authenticated users)
			if e.Auth == nil {
				clientIP := e.Request.RemoteAddr
				if !limiter.allow(clientIP) {
					log.Printf("[AI-PRINT] Rate limit exceeded for IP: %s", clientIP)
					return e.JSON(http.StatusTooManyRequests, map[string]string{
						"error": "Rate limit exceeded. Please try again later (max 5 generations per hour).",
					})
				}
			}

			// Find the view
			views, err := app.FindRecordsByFilter("views", "slug = {:slug}", "", 1, 0, map[string]interface{}{"slug": slug})
			if err != nil || len(views) == 0 {
				log.Printf("[AI-PRINT] View not found: %s", slug)
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}
			view := views[0]

			// Check view visibility - only allow public/unlisted views for unauthenticated users
			visibility := view.GetString("visibility")
			isAuthenticated := e.Auth != nil
			if !isAuthenticated && visibility != "public" && visibility != "unlisted" {
				log.Printf("[AI-PRINT] Unauthorized access to non-public view: %s", slug)
				return e.JSON(http.StatusForbidden, map[string]string{"error": "this view requires authentication"})
			}

			// Parse request body
			var req struct {
				Format     string   `json:"format"`
				ProviderID string   `json:"provider_id"`
				TargetRole string   `json:"target_role"`
				Style      string   `json:"style"`
				Length     string   `json:"length"`
				Emphasis   []string `json:"emphasis"`
			}
			if err := e.BindBody(&req); err != nil {
				log.Printf("[AI-PRINT] Invalid request body: %v", err)
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Defaults
			if req.Format == "" {
				req.Format = "pdf"
			}
			if req.Style == "" {
				req.Style = "chronological"
			}
			if req.Length == "" {
				req.Length = "two-page"
			}


			// Check Pandoc availability
			if !resume.CheckPandocAvailable() {
				log.Printf("[AI-PRINT] Pandoc not available")
				return e.JSON(http.StatusServiceUnavailable, map[string]string{
					"error": "PDF generation is not available. Pandoc is not installed.",
				})
			}

			// Get AI provider
			provider, err := getActiveProvider(app, crypto, req.ProviderID)
			if err != nil {
				log.Printf("[AI-PRINT] No AI provider available: %v", err)
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Create export record with pending status
			exportsCollection, err := app.FindCollectionByNameOrId("view_exports")
			if err != nil {
				log.Printf("[AI-PRINT] view_exports collection not found: %v", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "export collection not configured"})
			}

			exportRecord := core.NewRecord(exportsCollection)
			exportRecord.Set("view", view.Id)
			exportRecord.Set("format", req.Format)
			exportRecord.Set("status", "processing")
			exportRecord.Set("ai_provider", provider.ID)
			exportRecord.Set("generation_config", map[string]interface{}{
				"target_role": req.TargetRole,
				"style":       req.Style,
				"length":      req.Length,
				"emphasis":    req.Emphasis,
			})

			if err := app.Save(exportRecord); err != nil {
				log.Printf("[AI-PRINT] Failed to create export record: %v", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create export"})
			}

			// Fetch view data
			viewData, err := collectViewData(app, view)
			if err != nil {
				log.Printf("[AI-PRINT] Failed to collect view data: %v", err)
				exportRecord.Set("status", "failed")
				exportRecord.Set("error_message", "Failed to collect view data")
				app.Save(exportRecord)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to collect view data"})
			}

			// Generate resume
			config := &services.GenerationConfig{
				TargetRole: req.TargetRole,
				Style:      req.Style,
				Length:     req.Length,
				Emphasis:   req.Emphasis,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			fileBytes, err := resume.GenerateResume(ctx, provider, viewData, config, req.Format)
			if err != nil {
				log.Printf("[AI-PRINT] Resume generation failed: %v", err)
				exportRecord.Set("status", "failed")
				exportRecord.Set("error_message", err.Error())
				app.Save(exportRecord)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			// Save file to record
			filename := "resume." + req.Format
			f, err := filesystem.NewFileFromBytes(fileBytes, filename)
			if err != nil {
				log.Printf("[AI-PRINT] Failed to create file object: %v", err)
				exportRecord.Set("status", "failed")
				exportRecord.Set("error_message", "Failed to save file")
				app.Save(exportRecord)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file"})
			}

			exportRecord.Set("file", f)
			exportRecord.Set("status", "completed")
			exportRecord.Set("generated_at", time.Now())

			if err := app.Save(exportRecord); err != nil {
				log.Printf("[AI-PRINT] Failed to save export record with file: %v", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save export"})
			}


			// Build download URL
			downloadURL := "/api/files/" + exportsCollection.Id + "/" + exportRecord.Id + "/" + exportRecord.GetString("file")

			return e.JSON(http.StatusOK, map[string]interface{}{
				"export_id":    exportRecord.Id,
				"status":       "completed",
				"format":       req.Format,
				"download_url": downloadURL,
				"generated_at": exportRecord.Get("generated_at"),
			})
		}) // Public - visibility check above handles authorization

		// List exports for a view
		// GET /api/view/{slug}/exports
		se.Router.GET("/api/view/{slug}/exports", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			// Find the view
			views, err := app.FindRecordsByFilter("views", "slug = {:slug}", "", 1, 0, map[string]interface{}{"slug": slug})
			if err != nil || len(views) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}
			view := views[0]

			// Find exports for this view
			exports, err := app.FindRecordsByFilter(
				"view_exports",
				"view = {:viewId}",
				"-generated_at",
				50,
				0,
				map[string]interface{}{"viewId": view.Id},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch exports"})
			}

			exportsCollection, _ := app.FindCollectionByNameOrId("view_exports")

			var result []map[string]interface{}
			for _, exp := range exports {
				item := map[string]interface{}{
					"id":           exp.Id,
					"format":       exp.GetString("format"),
					"status":       exp.GetString("status"),
					"generated_at": exp.Get("generated_at"),
				}

				if exp.GetString("status") == "completed" && exp.GetString("file") != "" {
					item["download_url"] = "/api/files/" + exportsCollection.Id + "/" + exp.Id + "/" + exp.GetString("file")
				}

				if exp.GetString("status") == "failed" {
					item["error_message"] = exp.GetString("error_message")
				}

				// Include generation config
				var config map[string]interface{}
				if err := json.Unmarshal([]byte(exp.GetString("generation_config")), &config); err == nil {
					item["config"] = config
				}

				result = append(result, item)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"exports": result,
				"count":   len(result),
			})
		}).Bind(apis.RequireAuth())

		// Delete an export
		// DELETE /api/view/{slug}/exports/{exportId}
		se.Router.DELETE("/api/view/{slug}/exports/{exportId}", func(e *core.RequestEvent) error {
			exportId := e.Request.PathValue("exportId")

			exportRecord, err := app.FindRecordById("view_exports", exportId)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "export not found"})
			}

			if err := app.Delete(exportRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete export"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// collectViewData gathers all view data for resume generation
func collectViewData(app *pocketbase.PocketBase, view *core.Record) (*services.ViewData, error) {
	viewData := &services.ViewData{
		Profile:      make(map[string]interface{}),
		Sections:     make(map[string][]map[string]interface{}),
		SectionOrder: []string{},
	}

	// Get hero overrides from view
	if headline := view.GetString("hero_headline"); headline != "" {
		viewData.HeroHeadline = headline
	}
	if summary := view.GetString("hero_summary"); summary != "" {
		viewData.HeroSummary = summary
	}

	// Get profile
	profileRecords, err := app.FindRecordsByFilter("profile", "", "", 1, 0, nil)
	if err == nil && len(profileRecords) > 0 {
		profile := profileRecords[0]
		viewData.Profile["name"] = profile.GetString("name")
		viewData.Profile["headline"] = profile.GetString("headline")
		viewData.Profile["location"] = profile.GetString("location")
		viewData.Profile["summary"] = profile.GetString("summary")
		viewData.Profile["contact_email"] = profile.GetString("contact_email")
	}

	// Parse sections configuration
	sectionsJSON := view.GetString("sections")
	var sections []map[string]interface{}
	if sectionsJSON != "" {
		json.Unmarshal([]byte(sectionsJSON), &sections)
	}

	// Collect data for each enabled section
	for _, section := range sections {
		sectionName, ok := section["section"].(string)
		if !ok {
			continue
		}
		enabled, ok := section["enabled"].(bool)
		if !ok || !enabled {
			continue
		}

		viewData.SectionOrder = append(viewData.SectionOrder, sectionName)
		collectionName := getCollectionName(sectionName)
		if collectionName == "" {
			continue
		}

		// Check if specific items are selected
		items, hasItems := section["items"].([]interface{})
		var records []*core.Record

		if hasItems && len(items) > 0 {
			// Fetch specific items in order
			for _, itemID := range items {
				if id, ok := itemID.(string); ok {
					record, err := app.FindRecordById(collectionName, id)
					if err == nil && isRecordVisible(record) {
						records = append(records, record)
					}
				}
			}
		} else {
			// Fetch all visible items
			var fetchErr error
			records, fetchErr = app.FindRecordsByFilter(
				collectionName,
				"visibility != 'private' && is_draft = false",
				"sort_order",
				100,
				0,
				nil,
			)
			if fetchErr != nil {
				continue
			}
		}

		// Extract item configs for overrides
		itemConfig := make(map[string]map[string]interface{})
		if itemConfigRaw, ok := section["itemConfig"].(map[string]interface{}); ok {
			for itemID, config := range itemConfigRaw {
				if configMap, ok := config.(map[string]interface{}); ok {
					itemConfig[itemID] = configMap
				}
			}
		}

		// Convert records to maps with overrides applied
		var sectionItems []map[string]interface{}
		for _, record := range records {
			item := make(map[string]interface{})

			// Copy all fields
			for key, value := range record.FieldsData() {
				item[key] = value
			}
			item["id"] = record.Id

			// Apply overrides if present
			if overrides, hasOverrides := itemConfig[record.Id]; hasOverrides {
				for key, value := range overrides {
					if key != "id" && value != nil && value != "" {
						item[key] = value
					}
				}
			}

			sectionItems = append(sectionItems, item)
		}

		viewData.Sections[sectionName] = sectionItems
	}

	return viewData, nil
}
