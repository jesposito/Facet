package hooks

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Reserved slugs that cannot be used for views
// These correspond to existing routes or system paths
// SYNC WITH: frontend/src/params/slug.ts
var ReservedSlugs = map[string]bool{
	// Existing routes
	"admin":    true,
	"api":      true,
	"s":        true,
	"v":        true,
	"projects": true,
	"posts":    true,
	"talks":    true,
	// SvelteKit internal
	"_app": true,
	"_":    true,
	// Static assets
	"assets": true,
	"static": true,
	// Standard web files
	"favicon.ico": true,
	"robots.txt":  true,
	"sitemap.xml": true,
	// System endpoints
	"health":  true,
	"healthz": true,
	"ready":   true,
	// Common reserved paths
	"login":    true,
	"logout":   true,
	"auth":     true,
	"oauth":    true,
	"callback": true,
	// Prevent confusion
	"home":    true,
	"index":   true,
	"default": true,
	"profile": true,
}

// isValidSlug checks if a slug is valid (not reserved and proper format)
func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}
	// Check reserved
	if ReservedSlugs[strings.ToLower(slug)] {
		return false
	}
	// Check format (alphanumeric, hyphens, underscores, starts with letter/number)
	if len(slug) > 100 {
		return false
	}
	if slug[0] == '_' || slug[0] == '-' {
		return false
	}
	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// RegisterViewHooks registers view-related API endpoints
func RegisterViewHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, rl *services.RateLimitService) {
	// Register views collection hooks for validation
	registerViewsValidation(app)

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
				"view_id":           view.Id,
				"view_name":         view.GetString("name"),
				"slug":              slug,
				"visibility":        visibility,
				"requires_password": visibility == "password",
				"requires_token":    visibility == "unlisted",
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
				// Private views return 404 to prevent leaking existence
				// Only authenticated admin users can access private views
				if e.Auth == nil {
					return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
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
				"id":         view.Id,
				"slug":       slug,
				"name":       view.GetString("name"),
				"visibility": visibility,
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

			// Include view-specific accent color (null/empty means inherit from profile)
			if accentColor := view.GetString("accent_color"); accentColor != "" {
				response["accent_color"] = accentColor
			}

			// Get sections configuration
			sectionsJSON := view.GetString("sections")
			var sections []map[string]interface{}
			if sectionsJSON != "" {
				json.Unmarshal([]byte(sectionsJSON), &sections)
			}

			// Fetch content for each enabled section
			sectionData := make(map[string]interface{})
			// Track section order for frontend rendering
			var sectionOrder []string
			// Track layouts for each section
			sectionLayouts := make(map[string]string)
			// Track widths for each section (Phase 6.3)
			sectionWidths := make(map[string]string)

			for _, section := range sections {
				sectionName, ok := section["section"].(string)
				if !ok {
					continue
				}
				enabled, ok := section["enabled"].(bool)
				if !ok || !enabled {
					continue
				}
				// Add to order list
				sectionOrder = append(sectionOrder, sectionName)

				// Extract layout (default to "default" if not specified)
				if layout, ok := section["layout"].(string); ok && layout != "" {
					sectionLayouts[sectionName] = layout
				} else {
					sectionLayouts[sectionName] = getDefaultLayout(sectionName)
				}

				// Extract width (default to "full" if not specified)
				if width, ok := section["width"].(string); ok && width != "" {
					sectionWidths[sectionName] = width
				} else {
					sectionWidths[sectionName] = "full"
				}

				items, ok := section["items"].([]interface{})
				collectionName := getCollectionName(sectionName)
				if collectionName == "" {
					continue
				}

				// Extract itemConfig for overrides
				itemConfig := make(map[string]map[string]interface{})
				if itemConfigRaw, ok := section["itemConfig"].(map[string]interface{}); ok {
					for itemID, config := range itemConfigRaw {
						if configMap, ok := config.(map[string]interface{}); ok {
							itemConfig[itemID] = configMap
						}
					}
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
					sectionData[sectionName] = serializeRecordsWithOverrides(itemRecords, itemConfig, sectionName)
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
						sectionData[sectionName] = serializeRecordsWithOverrides(records, itemConfig, sectionName)
					}
				}
			}

			response["sections"] = sectionData
			response["section_order"] = sectionOrder
			response["section_layouts"] = sectionLayouts
			response["section_widths"] = sectionWidths

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
					"accent_color":  profile.GetString("accent_color"),
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

		// Get default view slug/data
		// This endpoint returns the slug of the default view for the homepage
		// Rate limited: normal tier (60/min)
		se.Router.GET("/api/default-view", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			settings, err := services.LoadSiteSettings(app)
			if err != nil {
				app.Logger().Warn("Failed to load site settings", "error", err)
			}

			if settings != nil && !settings.HomepageEnabled {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"has_default":          false,
					"fallback":             "homepage",
					"homepage_enabled":     false,
					"landing_page_message": settings.LandingPageMessage,
				})
			}

			// Find the default view (is_default = true, is_active = true, visibility = public)
			records, err := app.FindRecordsByFilter(
				"views",
				"is_default = true && is_active = true && visibility = 'public'",
				"",
				1,
				0,
				nil,
			)

			if err != nil || len(records) == 0 {
				// Fallback: find the first public active view by creation date
				records, err = app.FindRecordsByFilter(
					"views",
					"is_active = true && visibility = 'public'",
					"created",
					1,
					0,
					nil,
				)
			}

			if err != nil || len(records) == 0 {
				// No default view configured - return indicator
				return e.JSON(http.StatusOK, map[string]interface{}{
					"has_default": false,
					"fallback":    "homepage",
				})
			}

			view := records[0]
			return e.JSON(http.StatusOK, map[string]interface{}{
				"has_default":          true,
				"slug":                 view.GetString("slug"),
				"view_id":              view.Id,
				"name":                 view.GetString("name"),
				"homepage_enabled":     true,
				"landing_page_message": settings.LandingPageMessage,
			})
		}))

		// Get homepage data (public content aggregation)
		// DEPRECATED: Use /api/default-view + /api/view/{slug}/data instead
		// Kept for backwards compatibility during migration
		// Rate limited: normal tier (60/min) to prevent scraping
		se.Router.GET("/api/homepage", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			fmt.Println("[API /api/homepage] ========== REQUEST START ==========")
			response := make(map[string]interface{})

			settings, err := services.LoadSiteSettings(app)
			if err != nil {
				app.Logger().Warn("Failed to load site settings", "error", err)
			}
			if settings != nil && !settings.HomepageEnabled {
				response["homepage_enabled"] = false
				response["landing_page_message"] = settings.LandingPageMessage
				fmt.Println("[API /api/homepage] Homepage disabled via settings")
				fmt.Println("[API /api/homepage] ========== REQUEST END ==========")
				return e.JSON(http.StatusOK, response)
			}

			// Fetch profile - only public profiles appear on homepage
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility = 'public'",
				"",
				1,
				0,
				nil,
			)
			fmt.Printf("[API /api/homepage] Profile query: found %d records, err=%v\n", len(profileRecords), err)
			if err == nil && len(profileRecords) > 0 {
				profile := profileRecords[0]
				fmt.Printf("[API /api/homepage] Profile: id=%s name=%q visibility=%q\n",
					profile.Id, profile.GetString("name"), profile.GetString("visibility"))
				profileData := map[string]interface{}{
					"id":            profile.Id,
					"name":          profile.GetString("name"),
					"headline":      profile.GetString("headline"),
					"location":      profile.GetString("location"),
					"summary":       profile.GetString("summary"),
					"contact_email": profile.GetString("contact_email"),
					"contact_links": profile.Get("contact_links"),
					"visibility":    profile.GetString("visibility"),
					"accent_color":  profile.GetString("accent_color"),
				}

				// Include file URLs if present
				if heroImage := profile.GetString("hero_image"); heroImage != "" {
					profileData["hero_image_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + heroImage
				}
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileData["avatar_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + avatar
				}

				response["profile"] = profileData
			} else {
				fmt.Println("[API /api/homepage] No public profile found!")
			}

			// Fetch experience - only public items appear on homepage
			experienceRecords, err := app.FindRecordsByFilter(
				"experience",
				"visibility = 'public' && is_draft = false",
				"-sort_order,-start_date",
				100,
				0,
				nil,
			)
			if err == nil {
				response["experience"] = serializeRecords(experienceRecords)
			}

			// Fetch projects - only public items appear on homepage
			projectRecords, err := app.FindRecordsByFilter(
				"projects",
				"visibility = 'public' && is_draft = false",
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

			// Fetch education - only public items appear on homepage
			educationRecords, err := app.FindRecordsByFilter(
				"education",
				"visibility = 'public' && is_draft = false",
				"-sort_order,-end_date",
				100,
				0,
				nil,
			)
			if err == nil {
				response["education"] = serializeRecords(educationRecords)
			}

			// Fetch skills - only public items appear on homepage
			skillRecords, err := app.FindRecordsByFilter(
				"skills",
				"visibility = 'public'",
				"category,sort_order",
				200,
				0,
				nil,
			)
			if err == nil {
				response["skills"] = serializeRecords(skillRecords)
			}

			// Fetch posts - only public items appear on homepage
			postRecords, err := app.FindRecordsByFilter(
				"posts",
				"visibility = 'public' && is_draft = false",
				"-published_at",
				100,
				0,
				nil,
			)
			fmt.Printf("[API /api/homepage] Posts query: found %d records, err=%v\n", len(postRecords), err)
			if err == nil {
				for i, r := range postRecords {
					fmt.Printf("[API /api/homepage]   Post[%d] id=%s title=%q visibility=%q is_draft=%v\n",
						i, r.Id, r.GetString("title"), r.GetString("visibility"), r.GetBool("is_draft"))
				}
				posts := serializeRecords(postRecords)
				// Add file URLs for cover images
				for i, p := range posts {
					if coverImage, ok := p["cover_image"].(string); ok && coverImage != "" {
						if id, ok := p["id"].(string); ok {
							posts[i]["cover_image_url"] = "/api/files/posts/" + id + "/" + coverImage
						}
					}
				}
				response["posts"] = posts
			} else {
				fmt.Printf("[API /api/homepage] Posts query ERROR: %v\n", err)
			}

			// Fetch talks - only public items appear on homepage
			talkRecords, err := app.FindRecordsByFilter(
				"talks",
				"visibility = 'public' && is_draft = false",
				"-sort_order,-date",
				100,
				0,
				nil,
			)
			fmt.Printf("[API /api/homepage] Talks query: found %d records, err=%v\n", len(talkRecords), err)
			if err == nil {
				for i, r := range talkRecords {
					fmt.Printf("[API /api/homepage]   Talk[%d] id=%s title=%q visibility=%q is_draft=%v\n",
						i, r.Id, r.GetString("title"), r.GetString("visibility"), r.GetBool("is_draft"))
				}
				response["talks"] = serializeRecords(talkRecords)
			} else {
				fmt.Printf("[API /api/homepage] Talks query ERROR: %v\n", err)
			}

			// Fetch certifications - only public items appear on homepage
			certRecords, err := app.FindRecordsByFilter(
				"certifications",
				"visibility = 'public' && is_draft = false",
				"issuer,sort_order,-issue_date",
				100,
				0,
				nil,
			)
			if err == nil {
				response["certifications"] = serializeRecords(certRecords)
			}

			// Log final response summary
			expLen := 0
			projLen := 0
			eduLen := 0
			skillsLen := 0
			postsLen := 0
			talksLen := 0
			certsLen := 0
			if exp, ok := response["experience"].([]map[string]interface{}); ok {
				expLen = len(exp)
			}
			if proj, ok := response["projects"].([]map[string]interface{}); ok {
				projLen = len(proj)
			}
			if edu, ok := response["education"].([]map[string]interface{}); ok {
				eduLen = len(edu)
			}
			if skills, ok := response["skills"].([]map[string]interface{}); ok {
				skillsLen = len(skills)
			}
			if posts, ok := response["posts"].([]map[string]interface{}); ok {
				postsLen = len(posts)
			}
			if talks, ok := response["talks"].([]map[string]interface{}); ok {
				talksLen = len(talks)
			}
			if certs, ok := response["certifications"].([]map[string]interface{}); ok {
				certsLen = len(certs)
			}
			fmt.Printf("[API /api/homepage] Response summary: profile=%v exp=%d proj=%d edu=%d skills=%d posts=%d talks=%d certs=%d\n",
				response["profile"] != nil, expLen, projLen, eduLen, skillsLen, postsLen, talksLen, certsLen)
			fmt.Println("[API /api/homepage] ========== REQUEST END ==========")

			return e.JSON(http.StatusOK, response)
		}))

		// Public posts listing
		// Rate limited: normal tier (60/min)
		// Returns all non-private, non-draft posts for the index page
		se.Router.GET("/api/posts", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			fmt.Println("[API /api/posts] ========== REQUEST START ==========")

			settings, err := services.LoadSiteSettings(app)
			if err != nil {
				app.Logger().Warn("Failed to load site settings", "error", err)
			}
			if settings != nil && !settings.HomepageEnabled {
				fmt.Println("[API /api/posts] Homepage disabled via settings")
				return e.JSON(http.StatusForbidden, map[string]interface{}{
					"homepage_enabled":     false,
					"landing_page_message": settings.LandingPageMessage,
				})
			}

			// Fetch non-private, non-draft posts (public and unlisted)
			// Use explicit OR to handle NULL visibility values
			filter := "(visibility = 'public' || visibility = 'unlisted') && is_draft = false"
			fmt.Printf("[API /api/posts] Using filter: %s\n", filter)

			postRecords, err := app.FindRecordsByFilter(
				"posts",
				filter,
				"-published_at",
				100,
				0,
				nil,
			)
			if err != nil {
				fmt.Printf("[API /api/posts] ERROR: FindRecordsByFilter failed: %v\n", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch posts: " + err.Error()})
			}

			fmt.Printf("[API /api/posts] Found %d post records\n", len(postRecords))
			for i, r := range postRecords {
				fmt.Printf("[API /api/posts]   [%d] id=%s title=%q visibility=%q is_draft=%v\n",
					i, r.Id, r.GetString("title"), r.GetString("visibility"), r.GetBool("is_draft"))
			}

			posts := serializeRecords(postRecords)
			// Add file URLs for cover images
			for i, p := range posts {
				if coverImage, ok := p["cover_image"].(string); ok && coverImage != "" {
					if id, ok := p["id"].(string); ok {
						posts[i]["cover_image_url"] = "/api/files/posts/" + id + "/" + coverImage
					}
				}
			}

			// Fetch profile data for page context
			var profile map[string]interface{}
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility = 'public'",
				"",
				1,
				0,
				nil,
			)
			fmt.Printf("[API /api/posts] Profile query: found %d records, err=%v\n", len(profileRecords), err)
			if err == nil && len(profileRecords) > 0 {
				p := profileRecords[0]
				profile = map[string]interface{}{
					"id":       p.Id,
					"name":     p.GetString("name"),
					"headline": p.GetString("headline"),
				}
				fmt.Printf("[API /api/posts] Profile: id=%s name=%q\n", p.Id, p.GetString("name"))
			} else {
				fmt.Println("[API /api/posts] No public profile found")
			}

			fmt.Printf("[API /api/posts] Returning: posts=%d profile_exists=%v\n", len(posts), profile != nil)
			fmt.Println("[API /api/posts] ========== REQUEST END ==========")
			return e.JSON(http.StatusOK, map[string]interface{}{
				"posts":   posts,
				"profile": profile,
			})
		}))

		// RSS feed for public posts
		se.Router.GET("/rss.xml", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			fmt.Println("[API /rss.xml] ========== REQUEST START ==========")

			// Fetch profile for channel metadata
			channelTitle := "Facet - Latest Posts"
			channelDescription := "Recent posts"
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility != 'private'",
				"",
				1,
				0,
				nil,
			)
			if err == nil && len(profileRecords) > 0 {
				p := profileRecords[0]
				if name := p.GetString("name"); name != "" {
					channelTitle = name + " — Recent Posts"
				}
				if headline := p.GetString("headline"); headline != "" {
					channelDescription = headline
				}
			}

			// Fetch latest public posts
			postRecords, err := app.FindRecordsByFilter(
				"posts",
				"visibility = 'public' && is_draft = false",
				"-published_at",
				50,
				0,
				nil,
			)
			if err != nil {
				fmt.Printf("[API /rss.xml] ERROR: FindRecordsByFilter failed: %v\n", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch posts"})
			}

			type rssItem struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				GUID        string `xml:"guid"`
				Description string `xml:"description,omitempty"`
				PubDate     string `xml:"pubDate,omitempty"`
			}

			type rssChannel struct {
				Title       string    `xml:"title"`
				Link        string    `xml:"link"`
				Description string    `xml:"description"`
				Items       []rssItem `xml:"item"`
			}

			type rssFeed struct {
				XMLName xml.Name   `xml:"rss"`
				Version string     `xml:"version,attr"`
				Channel rssChannel `xml:"channel"`
			}

			baseURL := strings.TrimSuffix(resolveBaseURL(e), "/")
			items := make([]rssItem, 0, len(postRecords))

			for _, record := range postRecords {
				slug := record.GetString("slug")
				if slug == "" {
					continue
				}

				link := fmt.Sprintf("%s/posts/%s", baseURL, slug)
				pub := record.GetDateTime("published_at")
				pubDate := ""
				if !pub.IsZero() {
					pubDate = pub.Time().UTC().Format(time.RFC1123Z)
				}

				item := rssItem{
					Title:       record.GetString("title"),
					Link:        link,
					GUID:        link,
					Description: record.GetString("excerpt"),
					PubDate:     pubDate,
				}
				items = append(items, item)
			}

			feed := rssFeed{
				Version: "2.0",
				Channel: rssChannel{
					Title:       channelTitle,
					Link:        baseURL,
					Description: channelDescription,
					Items:       items,
				},
			}

			data, err := xml.MarshalIndent(feed, "", "  ")
			if err != nil {
				fmt.Printf("[API /rss.xml] ERROR: xml.MarshalIndent failed: %v\n", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to render feed"})
			}

			data = append([]byte(xml.Header), data...)
			e.Response.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
			_, _ = e.Response.Write(data)
			fmt.Printf("[API /rss.xml] Returned %d items\n", len(items))
			fmt.Println("[API /rss.xml] ========== REQUEST END ==========")
			return nil
		}))

		// Public talks listing
		// Rate limited: normal tier (60/min)
		// Returns all non-private, non-draft talks for the index page
		se.Router.GET("/api/talks", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			fmt.Println("[API /api/talks] ========== REQUEST START ==========")

			settings, err := services.LoadSiteSettings(app)
			if err != nil {
				app.Logger().Warn("Failed to load site settings", "error", err)
			}
			if settings != nil && !settings.HomepageEnabled {
				fmt.Println("[API /api/talks] Homepage disabled via settings")
				return e.JSON(http.StatusForbidden, map[string]interface{}{
					"homepage_enabled":     false,
					"landing_page_message": settings.LandingPageMessage,
				})
			}

			// Fetch non-private, non-draft talks (public and unlisted)
			// Use explicit OR to handle NULL visibility values
			filter := "(visibility = 'public' || visibility = 'unlisted') && is_draft = false"
			fmt.Printf("[API /api/talks] Using filter: %s\n", filter)

			talkRecords, err := app.FindRecordsByFilter(
				"talks",
				filter,
				"-date,-sort_order",
				100,
				0,
				nil,
			)
			if err != nil {
				fmt.Printf("[API /api/talks] ERROR: FindRecordsByFilter failed: %v\n", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch talks: " + err.Error()})
			}

			fmt.Printf("[API /api/talks] Found %d talk records\n", len(talkRecords))
			for i, r := range talkRecords {
				fmt.Printf("[API /api/talks]   [%d] id=%s title=%q visibility=%q is_draft=%v\n",
					i, r.Id, r.GetString("title"), r.GetString("visibility"), r.GetBool("is_draft"))
			}

			talks := serializeRecords(talkRecords)

			// Fetch profile data for page context
			var profile map[string]interface{}
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility = 'public'",
				"",
				1,
				0,
				nil,
			)
			fmt.Printf("[API /api/talks] Profile query: found %d records, err=%v\n", len(profileRecords), err)
			if err == nil && len(profileRecords) > 0 {
				p := profileRecords[0]
				profile = map[string]interface{}{
					"id":       p.Id,
					"name":     p.GetString("name"),
					"headline": p.GetString("headline"),
				}
				fmt.Printf("[API /api/talks] Profile: id=%s name=%q\n", p.Id, p.GetString("name"))
			} else {
				fmt.Println("[API /api/talks] No public profile found")
			}

			fmt.Printf("[API /api/talks] Returning: talks=%d profile_exists=%v\n", len(talks), profile != nil)
			fmt.Println("[API /api/talks] ========== REQUEST END ==========")
			return e.JSON(http.StatusOK, map[string]interface{}{
				"talks":   talks,
				"profile": profile,
			})
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
				AppliedFields map[string]bool        `json:"applied_fields"` // field -> should apply
				LockedFields  []string               `json:"locked_fields"`  // fields to lock
				Edits         map[string]interface{} `json:"edits"`          // manual edits
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

		// Get public post by slug
		// Rate limited: normal tier (60/min)
		// Returns 404 for private, unlisted, draft, or non-existent posts
		se.Router.GET("/api/post/{slug}", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			if slug == "" {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "post not found"})
			}

			// Find post by slug
			records, err := app.FindRecordsByFilter(
				"posts",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(records) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "post not found"})
			}

			post := records[0]

			// Check visibility - only public, non-draft posts are accessible
			visibility := post.GetString("visibility")
			isDraft := post.GetBool("is_draft")

			if visibility != "public" || isDraft {
				// Return 404 to prevent discovery of private/unlisted/draft posts
				return e.JSON(http.StatusNotFound, map[string]string{"error": "post not found"})
			}

			// Build response with resolved file URLs
			response := map[string]interface{}{
				"id":           post.Id,
				"title":        post.GetString("title"),
				"slug":         post.GetString("slug"),
				"excerpt":      post.GetString("excerpt"),
				"content":      post.GetString("content"),
				"tags":         post.Get("tags"),
				"published_at": post.GetDateTime("published_at"),
				"created":      post.GetDateTime("created"),
				"updated":      post.GetDateTime("updated"),
			}

			// Resolve cover image URL
			if coverImage := post.GetString("cover_image"); coverImage != "" {
				response["cover_image_url"] = "/api/files/" + post.Collection().Id + "/" + post.Id + "/" + coverImage
			}

			// Fetch profile data for navigation context
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility = 'public'",
				"",
				1,
				0,
				nil,
			)
			if err == nil && len(profileRecords) > 0 {
				profile := profileRecords[0]
				profileData := map[string]interface{}{
					"id":       profile.Id,
					"name":     profile.GetString("name"),
					"headline": profile.GetString("headline"),
				}
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileData["avatar_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + avatar
				}
				response["profile"] = profileData
			}

			// Fetch previous and next posts for navigation
			// Previous post (published before this one)
			prevRecords, err := app.FindRecordsByFilter(
				"posts",
				"visibility = 'public' && is_draft = false && published_at < {:published_at}",
				"-published_at",
				1,
				0,
				map[string]interface{}{"published_at": post.GetDateTime("published_at").String()},
			)
			if err == nil && len(prevRecords) > 0 {
				prev := prevRecords[0]
				response["prev_post"] = map[string]interface{}{
					"slug":  prev.GetString("slug"),
					"title": prev.GetString("title"),
				}
			}

			// Next post (published after this one)
			nextRecords, err := app.FindRecordsByFilter(
				"posts",
				"visibility = 'public' && is_draft = false && published_at > {:published_at}",
				"published_at",
				1,
				0,
				map[string]interface{}{"published_at": post.GetDateTime("published_at").String()},
			)
			if err == nil && len(nextRecords) > 0 {
				next := nextRecords[0]
				response["next_post"] = map[string]interface{}{
					"slug":  next.GetString("slug"),
					"title": next.GetString("title"),
				}
			}

			return e.JSON(http.StatusOK, response)
		}))

		// Get public project by slug
		// Rate limited: normal tier (60/min)
		// Returns 404 for private, unlisted, draft, or non-existent projects
		se.Router.GET("/api/project/{slug}", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			if slug == "" {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "project not found"})
			}

			// Find project by slug
			records, err := app.FindRecordsByFilter(
				"projects",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(records) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "project not found"})
			}

			project := records[0]

			// Check visibility - only public, non-draft projects are accessible
			visibility := project.GetString("visibility")
			isDraft := project.GetBool("is_draft")

			if visibility != "public" || isDraft {
				// Return 404 to prevent discovery of private/unlisted/draft projects
				return e.JSON(http.StatusNotFound, map[string]string{"error": "project not found"})
			}

			// Build response with resolved file URLs
			response := map[string]interface{}{
				"id":          project.Id,
				"title":       project.GetString("title"),
				"slug":        project.GetString("slug"),
				"summary":     project.GetString("summary"),
				"description": project.GetString("description"),
				"tech_stack":  project.Get("tech_stack"),
				"links":       project.Get("links"),
				"categories":  project.Get("categories"),
				"is_featured": project.GetBool("is_featured"),
			}

			// Resolve cover image URL
			if coverImage := project.GetString("cover_image"); coverImage != "" {
				response["cover_image_url"] = "/api/files/" + project.Collection().Id + "/" + project.Id + "/" + coverImage
			}

			// Resolve media URLs
			if mediaField := project.Get("media"); mediaField != nil {
				if mediaFiles, ok := mediaField.([]string); ok && len(mediaFiles) > 0 {
					var mediaURLs []string
					for _, file := range mediaFiles {
						mediaURLs = append(mediaURLs, "/api/files/"+project.Collection().Id+"/"+project.Id+"/"+file)
					}
					response["media_urls"] = mediaURLs
				}
			}

			// Fetch profile data for navigation context
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility = 'public'",
				"",
				1,
				0,
				nil,
			)
			if err == nil && len(profileRecords) > 0 {
				profile := profileRecords[0]
				profileData := map[string]interface{}{
					"id":       profile.Id,
					"name":     profile.GetString("name"),
					"headline": profile.GetString("headline"),
				}
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileData["avatar_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + avatar
				}
				response["profile"] = profileData
			}

			return e.JSON(http.StatusOK, response)
		}))

		// Get public talk by slug
		// Rate limited: normal tier (60/min)
		// Returns 404 for private, unlisted, draft, or non-existent talks
		se.Router.GET("/api/talk/{slug}", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			if slug == "" {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "talk not found"})
			}

			// Find talk by slug
			records, err := app.FindRecordsByFilter(
				"talks",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(records) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "talk not found"})
			}

			talk := records[0]

			// Check visibility - only public, non-draft talks are accessible
			visibility := talk.GetString("visibility")
			isDraft := talk.GetBool("is_draft")

			if visibility != "public" || isDraft {
				// Return 404 to prevent discovery of private/unlisted/draft talks
				return e.JSON(http.StatusNotFound, map[string]string{"error": "talk not found"})
			}

			// Build response
			response := map[string]interface{}{
				"id":          talk.Id,
				"title":       talk.GetString("title"),
				"slug":        talk.GetString("slug"),
				"event":       talk.GetString("event"),
				"event_url":   talk.GetString("event_url"),
				"date":        talk.GetDateTime("date"),
				"location":    talk.GetString("location"),
				"description": talk.GetString("description"),
				"slides_url":  talk.GetString("slides_url"),
				"video_url":   talk.GetString("video_url"),
				"created":     talk.GetDateTime("created"),
				"updated":     talk.GetDateTime("updated"),
			}

			// Fetch profile data for navigation context
			profileRecords, err := app.FindRecordsByFilter(
				"profile",
				"visibility = 'public'",
				"",
				1,
				0,
				nil,
			)
			if err == nil && len(profileRecords) > 0 {
				profile := profileRecords[0]
				profileData := map[string]interface{}{
					"id":       profile.Id,
					"name":     profile.GetString("name"),
					"headline": profile.GetString("headline"),
				}
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileData["avatar_url"] = "/api/files/" + profile.Collection().Id + "/" + profile.Id + "/" + avatar
				}
				response["profile"] = profileData
			}

			// Fetch previous and next talks for navigation
			// Previous talk (before this one by date)
			if talkDate := talk.GetDateTime("date"); !talkDate.IsZero() {
				prevRecords, err := app.FindRecordsByFilter(
					"talks",
					"visibility = 'public' && is_draft = false && date < {:date}",
					"-date",
					1,
					0,
					map[string]interface{}{"date": talkDate.String()},
				)
				if err == nil && len(prevRecords) > 0 {
					prev := prevRecords[0]
					if prevSlug := prev.GetString("slug"); prevSlug != "" {
						response["prev_talk"] = map[string]interface{}{
							"slug":  prevSlug,
							"title": prev.GetString("title"),
						}
					}
				}

				// Next talk (after this one by date)
				nextRecords, err := app.FindRecordsByFilter(
					"talks",
					"visibility = 'public' && is_draft = false && date > {:date}",
					"date",
					1,
					0,
					map[string]interface{}{"date": talkDate.String()},
				)
				if err == nil && len(nextRecords) > 0 {
					next := nextRecords[0]
					if nextSlug := next.GetString("slug"); nextSlug != "" {
						response["next_talk"] = map[string]interface{}{
							"slug":  nextSlug,
							"title": next.GetString("title"),
						}
					}
				}
			}

			return e.JSON(http.StatusOK, response)
		}))

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

// getDefaultLayout returns the default layout for a section type
// Sync with VALID_LAYOUTS in frontend/src/lib/pocketbase.ts
func getDefaultLayout(section string) string {
	switch section {
	case "experience":
		return "default"
	case "projects":
		return "grid-3"
	case "education":
		return "default"
	case "certifications":
		return "grouped"
	case "skills":
		return "grouped"
	case "posts":
		return "grid-3"
	case "talks":
		return "default"
	default:
		return "default"
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

// serializeRecordsWithOverrides serializes records and applies view-specific field overrides
func serializeRecordsWithOverrides(records []*core.Record, itemConfig map[string]map[string]interface{}, sectionName string) []map[string]interface{} {
	var result []map[string]interface{}
	overridableFields := getOverridableFields(sectionName)

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

		// Apply overrides if present for this item
		if config, exists := itemConfig[record.Id]; exists {
			if overrides, ok := config["overrides"].(map[string]interface{}); ok {
				for field, value := range overrides {
					// Only apply overrides for allowed fields
					if containsString(overridableFields, field) {
						item[field] = value
					}
				}
			}
		}

		result = append(result, item)
	}
	return result
}

// getOverridableFields returns the list of fields that can be overridden per section
func getOverridableFields(sectionName string) []string {
	switch sectionName {
	case "experience":
		return []string{"title", "description", "bullets"}
	case "projects":
		return []string{"title", "summary", "description"}
	case "education":
		return []string{"degree", "field", "description"}
	case "talks":
		return []string{"title", "description"}
	default:
		return []string{}
	}
}

// containsString checks if a string slice contains a specific string
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
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
//  1. X-Share-Token: <token> - Primary header for share tokens
//  2. ?token=<token> - LEGACY/COMPAT ONLY for shareable links
//
// NOTE: Authorization header is NOT checked here because it's reserved for
// password JWTs. This prevents conflicts when both tokens are present.
//
// SECURITY WARNING: Query parameter tokens are logged in server access logs,
// browser history, and may leak via Referer headers. Use headers when possible.
func extractShareToken(e *core.RequestEvent) string {
	// Check X-Share-Token header (primary method)
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

// registerViewsValidation registers hooks for views collection validation
// This enforces:
// 1. Reserved slug protection - prevents creating views with reserved slugs
// 2. Single default view - ensures only one view can be marked as default
func registerViewsValidation(app *pocketbase.PocketBase) {
	// Validate on create
	app.OnRecordCreate("views").BindFunc(func(e *core.RecordEvent) error {
		slug := e.Record.GetString("slug")

		// Validate slug is not reserved
		if !isValidSlug(slug) {
			return fmt.Errorf("invalid or reserved slug: slugs cannot use reserved paths like 'admin', 'api', 's', 'v', etc")
		}

		// If this view is being set as default, clear other defaults
		if e.Record.GetBool("is_default") {
			if err := clearOtherDefaults(app, ""); err != nil {
				return err
			}
		}

		return e.Next()
	})

	// Validate on update
	app.OnRecordUpdate("views").BindFunc(func(e *core.RecordEvent) error {
		slug := e.Record.GetString("slug")

		// Validate slug is not reserved
		if !isValidSlug(slug) {
			return fmt.Errorf("invalid or reserved slug: slugs cannot use reserved paths like 'admin', 'api', 's', 'v', etc")
		}

		// If this view is being set as default, clear other defaults
		if e.Record.GetBool("is_default") {
			if err := clearOtherDefaults(app, e.Record.Id); err != nil {
				return err
			}
		}

		return e.Next()
	})
}

// clearOtherDefaults removes is_default from all views except the one with excludeID
func clearOtherDefaults(app *pocketbase.PocketBase, excludeID string) error {
	filter := "is_default = true"
	if excludeID != "" {
		filter += " && id != {:id}"
	}

	records, err := app.FindRecordsByFilter(
		"views",
		filter,
		"",
		100,
		0,
		map[string]interface{}{"id": excludeID},
	)

	if err != nil {
		return err
	}

	for _, record := range records {
		record.Set("is_default", false)
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}

func resolveBaseURL(e *core.RequestEvent) string {
	if appURL := strings.TrimSpace(os.Getenv("APP_URL")); appURL != "" {
		return strings.TrimSuffix(appURL, "/")
	}

	req := e.Request
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if req.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := req.Host
	return fmt.Sprintf("%s://%s", proto, host)
}
