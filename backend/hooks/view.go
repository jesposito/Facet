package hooks

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterViewHooks registers view-related API endpoints
func RegisterViewHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, rl *services.RateLimitService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Get view access info (for frontend to determine access)
		// Rate limited: normal tier (60/min) to prevent enumeration
		se.Router.GET("/api/view/{slug}/access", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
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
				"view_id":            view.Id,
				"view_name":          view.GetString("name"),
				"slug":               slug,
				"visibility":         visibility,
				"requires_password":  visibility == "password",
				"requires_token":     visibility == "unlisted",
			})
		}))

		// Get full view data (with content filtering based on sections config)
		// Rate limited: normal tier (60/min) to prevent scraping
		se.Router.GET("/api/view/{slug}/data", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
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
			switch visibility {
			case "private":
				// Private views require admin authentication
				if e.Auth == nil {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				}

			case "password":
				// Password-protected views require valid JWT
				token := extractPasswordToken(e)
				if token == "" {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "password token required"})
				}

				viewID, err := crypto.ValidateViewAccessJWT(token)
				if err != nil {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				}

				// Ensure token is for this specific view
				if viewID != view.Id {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "token not valid for this view"})
				}

			case "unlisted":
				// Unlisted views require a valid share token
				shareToken := extractShareToken(e)
				if shareToken == "" {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "share token required"})
				}

				// Validate the share token
				valid, tokenRecord := validateShareToken(app, share, shareToken, view.Id)
				if !valid {
					return e.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired share token"})
				}

				// Update token usage (fire-and-forget, don't block request)
				if tokenRecord != nil {
					useCount := tokenRecord.GetInt("use_count")
					tokenRecord.Set("use_count", useCount+1)
					tokenRecord.Set("last_used_at", time.Now())
					app.Save(tokenRecord)
				}

			case "public":
				// Public views are accessible to everyone
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

			// Fetch profile data for the view
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility != 'private'",
				"",
				1,
				0,
				nil,
			)
			if err == nil && len(profileRecords) > 0 {
				profile := profileRecords[0]
				profileData := map[string]interface{}{
					"id":            profile.Id,
					"name":          profile.GetString("name"),
					"headline":      profile.GetString("headline"),
					"location":      profile.GetString("location"),
					"summary":       profile.GetString("summary"),
					"contact_email": profile.GetString("contact_email"),
					"contact_links": profile.Get("contact_links"),
					"visibility":    profile.GetString("visibility"),
				}

				// Include file URLs if present
				if heroImage := profile.GetString("hero_image"); heroImage != "" {
					profileData["hero_image_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + heroImage
				}
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileData["avatar_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + avatar
				}

				response["profile"] = profileData
			}

			return e.JSON(http.StatusOK, response)
		}))

		// Get homepage data (public content aggregation)
		// Rate limited: normal tier (60/min) to prevent scraping
		se.Router.GET("/api/homepage", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			response := make(map[string]interface{})

			// Fetch profile
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility != 'private'",
				"",
				1,
				0,
				nil,
			)
			if err == nil && len(profileRecords) > 0 {
				profile := profileRecords[0]
				profileData := map[string]interface{}{
					"id":            profile.Id,
					"name":          profile.GetString("name"),
					"headline":      profile.GetString("headline"),
					"location":      profile.GetString("location"),
					"summary":       profile.GetString("summary"),
					"contact_email": profile.GetString("contact_email"),
					"contact_links": profile.Get("contact_links"),
					"visibility":    profile.GetString("visibility"),
				}

				// Include file URLs if present
				if heroImage := profile.GetString("hero_image"); heroImage != "" {
					profileData["hero_image_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + heroImage
				}
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileData["avatar_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + avatar
				}

				response["profile"] = profileData
			}

			// Fetch experience
			experienceRecords, err := app.FindRecordsByFilter(
				"experience",
				"visibility != 'private' && is_draft = false",
				"-sort_order,-start_date",
				100,
				0,
				nil,
			)
			if err == nil {
				response["experience"] = serializeRecords(experienceRecords)
			}

			// Fetch projects
			projectRecords, err := app.FindRecordsByFilter(
				"projects",
				"visibility != 'private' && is_draft = false",
				"-is_featured,-sort_order",
				100,
				0,
				nil,
			)
			if err == nil {
				projects := serializeRecords(projectRecords)
				// Add file URLs for cover images
				for i, p := range projects {
					if coverImage, ok := p["cover_image"].(string); ok && coverImage != "" {
						if id, ok := p["id"].(string); ok {
							projects[i]["cover_image_url"] = "/api/files/projects/" + id + "/" + coverImage
						}
					}
				}
				response["projects"] = projects
			}

			// Fetch education
			educationRecords, err := app.FindRecordsByFilter(
				"education",
				"visibility != 'private' && is_draft = false",
				"-sort_order,-end_date",
				100,
				0,
				nil,
			)
			if err == nil {
				response["education"] = serializeRecords(educationRecords)
			}

			// Fetch skills
			skillRecords, err := app.FindRecordsByFilter(
				"skills",
				"visibility != 'private'",
				"category,sort_order",
				200,
				0,
				nil,
			)
			if err == nil {
				response["skills"] = serializeRecords(skillRecords)
			}

			return e.JSON(http.StatusOK, response)
		}))

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
		for key, value := range record.FieldsData() {
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

// extractPasswordToken extracts the password access token from request headers
// Accepts: Authorization: Bearer <token> (preferred) or X-Password-Token: <token>
func extractPasswordToken(e *core.RequestEvent) string {
	// Check Authorization header first (preferred)
	authHeader := e.Request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to X-Password-Token header (legacy/UI convenience)
	return e.Request.Header.Get("X-Password-Token")
}

// extractShareToken extracts the share token from request headers or query params.
//
// Transport methods (in order of preference):
//  1. Authorization: Bearer <token> - RECOMMENDED for API clients
//  2. X-Share-Token: <token> - Alternative header for programmatic access
//  3. ?token=<token> - LEGACY/COMPAT ONLY for shareable links
//
// SECURITY WARNING: Query parameter tokens are logged in server access logs,
// browser history, and may leak via Referer headers. Use headers when possible.
func extractShareToken(e *core.RequestEvent) string {
	// Check Authorization header first (preferred, most secure)
	authHeader := e.Request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Check X-Share-Token header (alternative, also secure)
	if shareToken := e.Request.Header.Get("X-Share-Token"); shareToken != "" {
		return shareToken
	}

	// LEGACY: Query parameter for shareable links
	// WARNING: Tokens in URLs may leak via logs, Referer headers, browser history
	return e.Request.URL.Query().Get("token")
}

// validateShareToken validates a share token for a specific view
// Returns (valid, tokenRecord) - tokenRecord is returned for usage tracking
func validateShareToken(app *pocketbase.PocketBase, share *services.ShareService, token string, viewID string) (bool, *core.Record) {
	if token == "" {
		return false, nil
	}

	// O(1) lookup using token_prefix index
	prefix := share.TokenPrefix(token)

	// Query by prefix for efficient lookup (indexed)
	candidates, err := app.FindRecordsByFilter(
		"share_tokens",
		"token_prefix = {:prefix} && is_active = true",
		"-created",
		10,
		0,
		map[string]interface{}{"prefix": prefix},
	)

	// Fallback to legacy lookup if no prefix-based results
	if err != nil || len(candidates) == 0 {
		candidates, err = app.FindRecordsByFilter(
			"share_tokens",
			"(token_prefix = '' || token_prefix IS NULL) && is_active = true",
			"-created",
			100,
			0,
			nil,
		)
	}

	// Find matching token using constant-time HMAC comparison
	var tokenRecord *core.Record
	for _, record := range candidates {
		storedHMAC := record.GetString("token_hash")
		if share.ValidateTokenHMAC(token, storedHMAC) {
			tokenRecord = record
			break
		}
	}

	if err != nil || tokenRecord == nil {
		return false, nil
	}

	// Verify token is for this specific view
	if tokenRecord.GetString("view_id") != viewID {
		return false, nil
	}

	// Check expiration
	expiresAt := tokenRecord.GetDateTime("expires_at")
	if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
		return false, nil
	}

	// Check max uses
	useCount := tokenRecord.GetInt("use_count")
	maxUses := tokenRecord.GetInt("max_uses")
	if maxUses > 0 && useCount >= maxUses {
		return false, nil
	}

	return true, tokenRecord
}
