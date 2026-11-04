package hooks

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"facet/internal/textlimit"
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
func RegisterResumeHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService) {
	ai := services.NewAIService(crypto)
	resume := services.NewResumeService(ai)
	limiter := newRateLimiter(5, time.Hour) // 5 generations per hour per IP
	purchaseService := services.NewPurchaseService(crypto)

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
		// Visitors can generate from any view they could open: the same
		// password token or share token that /data accepts works here.
		se.Router.POST("/api/view/{slug}/generate", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")
			isOwner := isSuperuser(e)

			// Rate limit by client address (skip for the site owner). Each
			// generation costs an AI call and a LaTeX run.
			if !isOwner {
				clientIP := services.ClientIP(e.Request)
				if !limiter.allow(clientIP) {
					log.Printf("[AI-PRINT] Rate limit exceeded for IP: %s", clientIP)
					return e.JSON(http.StatusTooManyRequests, map[string]string{
						"error": "Rate limit exceeded. Please try again later (max 5 generations per hour).",
					})
				}
			}

			// Find the view. Visitors only reach active views, like /data.
			filter := "slug = {:slug}"
			if !isOwner {
				filter += " && is_active = true"
			}
			views, err := app.FindRecordsByFilter("views", filter, "", 1, 0, map[string]interface{}{"slug": slug})
			if err != nil || len(views) == 0 {
				log.Printf("[AI-PRINT] View not found: %s", slug)
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}
			view := views[0]

			// Generating doesn't use up a share token; opening the view did.
			_, denial := checkViewAccess(app, crypto, share, e, view)
			if denial = publicViewDenial(view, denial); denial != nil {
				return e.JSON(denial.status, map[string]string{"error": denial.message})
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
			if req.Format != "pdf" && req.Format != "docx" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "format must be pdf or docx"})
			}
			// Visitors get the site's default provider. Choosing one is an
			// owner setting.
			if !isOwner {
				req.ProviderID = ""
			}
			// These go into the AI prompt. Keep them to what the form needs.
			req.TargetRole = textlimit.Bytes(req.TargetRole, 200)
			req.Style = textlimit.Bytes(req.Style, 40)
			req.Length = textlimit.Bytes(req.Length, 40)
			if len(req.Emphasis) > 10 {
				req.Emphasis = req.Emphasis[:10]
			}
			for i := range req.Emphasis {
				req.Emphasis[i] = textlimit.Bytes(req.Emphasis[i], 100)
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
			viewData, err := collectViewData(app, view, newContentAccess(app, purchaseService, e))
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
		}).Bind(apis.RequireAuth("users", core.CollectionNameSuperusers))

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
		}).Bind(apis.RequireAuth("users", core.CollectionNameSuperusers))

		return se.Next()
	})
}

// collectViewData gathers the view's content for resume generation. Records
// go through the same public projection as /api/view/{slug}/data, so the
// prompt never holds a field the visitor couldn't already read on the page.
func collectViewData(app *pocketbase.PocketBase, view *core.Record, access contentAccess) (*services.ViewData, error) {
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

	// Get profile (the same record /data shows)
	profileRecords, err := app.FindRecordsByFilter("profile", "visibility != 'private'", "", 1, 0, nil)
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

		// Skip custom content sections for resume generation
		// Custom content is freeform and may not fit standard resume formats
		// The AI Print feature generates professional resumes with standardized sections
		if isCustomContentSection(sectionName) {
			continue
		}

		viewData.SectionOrder = append(viewData.SectionOrder, sectionName)

		collectionName := getCollectionName(sectionName)
		if collectionName == "" {
			continue
		}

		// Same records as /api/view/{slug}/data, so the resume matches the page.
		records, err := resolveSectionRecords(app, collectionName, sectionName, section, view.Id)
		if err != nil {
			return nil, err
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

		viewData.Sections[sectionName] = serializeRecordsWithOverrides(app, records, itemConfig, sectionName, access)
	}

	return viewData, nil
}
