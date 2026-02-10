package hooks

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// RegisterSiteSettingsHooks exposes site settings for homepage/privacy control.
func RegisterSiteSettingsHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Authenticated: check encryption key source
		se.Router.GET("/api/system/encryption-status", func(e *core.RequestEvent) error {
			source := os.Getenv("ENCRYPTION_KEY_SOURCE")
			if source == "" {
				// If not set by start.sh, the key came from env directly
				source = "env"
			}
			return e.JSON(http.StatusOK, map[string]any{
				"source": source,
				// auto = generated on first run, saved to /data/.encryption_key
				// file = loaded from /data/.encryption_key (previously generated)
				// env  = explicitly set via ENCRYPTION_KEY environment variable
			})
		}).Bind(apis.RequireAuth())

		// Public: fetch site settings (sanitized)
		se.Router.GET("/api/site-settings", func(e *core.RequestEvent) error {
			settings, err := services.LoadSiteSettings(app)
			if err != nil {
				app.Logger().Error("Failed to load site settings", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load site settings"})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"homepage_enabled":        settings.HomepageEnabled,
				"landing_page_message":    settings.LandingPageMessage,
				"custom_css":              settings.CustomCSS,
				"ga_measurement_id":       settings.GAMeasurementID,
				"hide_login_button":       settings.HideLoginButton,
				"hide_demo_toggle":        settings.HideDemoToggle,
				"homepage_custom_content": settings.HomepageCustomContent,
				"homepage_section_order":  settings.HomepageSectionOrder,
				"homepage_sections":       settings.HomepageSections,
				"site_nav_enabled":        settings.SiteNavEnabled,
				"site_nav_items":          settings.SiteNavItems,
				"skills_category_order":   settings.SkillsCategoryOrder,
				"site_cta_enabled":        settings.SiteCtaEnabled,
				"favicon":                 settings.Favicon,
				"default_locale":          settings.DefaultLocale,
				"homepage_view_count":     settings.HomepageViewCount,
				"homepage_last_viewed_at": settings.HomepageLastViewedAt,
			})
		})

		// Authenticated: update site settings
		se.Router.PUT("/api/site-settings", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return apis.NewUnauthorizedError("authentication required", nil)
			}

			var req struct {
				HomepageEnabled       *bool                                     `json:"homepage_enabled"`
				LandingPageMessage    *string                                   `json:"landing_page_message"`
				CustomCSS             *string                                   `json:"custom_css"`
				GAMeasurementID       *string                                   `json:"ga_measurement_id"`
				HideLoginButton       *bool                                     `json:"hide_login_button"`
				HideDemoToggle        *bool                                     `json:"hide_demo_toggle"`
				HomepageCustomContent []services.HomepageCustomContentItem      `json:"homepage_custom_content"`
				HomepageSectionOrder  []string                                  `json:"homepage_section_order"`
				HomepageSections      map[string]services.HomepageSectionConfig `json:"homepage_sections"`
				SiteNavEnabled        *bool                                     `json:"site_nav_enabled"`
				SiteNavItems          []services.SiteNavItem                    `json:"site_nav_items"`
				SkillsCategoryOrder   []string                                  `json:"skills_category_order"`
				SiteCtaEnabled        *bool                                     `json:"site_cta_enabled"`
				DefaultLocale         *string                                   `json:"default_locale"`
			}

			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			updates := make(map[string]any)
			if req.HomepageEnabled != nil {
				updates["homepage_enabled"] = *req.HomepageEnabled
			}
			if req.LandingPageMessage != nil {
				// Allow clearing the message by sending empty string
				msg := strings.TrimSpace(*req.LandingPageMessage)
				if len(msg) > 2000 {
					msg = msg[:2000]
				}
				updates["landing_page_message"] = msg
			}
			if req.CustomCSS != nil {
				css := strings.TrimSpace(*req.CustomCSS)
				if len(css) > 20000 {
					css = css[:20000]
				}
				updates["custom_css"] = css
			}
			if req.GAMeasurementID != nil {
				id := strings.TrimSpace(*req.GAMeasurementID)
				if len(id) > 100 {
					id = id[:100]
				}
				updates["ga_measurement_id"] = id
			}
			if req.HideLoginButton != nil {
				updates["hide_login_button"] = *req.HideLoginButton
			}
			if req.HideDemoToggle != nil {
				updates["hide_demo_toggle"] = *req.HideDemoToggle
			}
			if req.HomepageCustomContent != nil {
				updates["homepage_custom_content"] = req.HomepageCustomContent
			}
			if req.HomepageSectionOrder != nil {
				updates["homepage_section_order"] = req.HomepageSectionOrder
			}
			if req.HomepageSections != nil {
				updates["homepage_sections"] = req.HomepageSections
			}
			if req.SiteNavEnabled != nil {
				updates["site_nav_enabled"] = *req.SiteNavEnabled
			}
			if req.SiteNavItems != nil {
				updates["site_nav_items"] = req.SiteNavItems
			}
			if req.SkillsCategoryOrder != nil {
				updates["skills_category_order"] = req.SkillsCategoryOrder
			}
			if req.SiteCtaEnabled != nil {
				updates["site_cta_enabled"] = *req.SiteCtaEnabled
			}
			if req.DefaultLocale != nil {
				locale := strings.TrimSpace(*req.DefaultLocale)
				updates["default_locale"] = locale
			}

			settings, err := services.UpdateSiteSettings(app, updates, app.Logger())
			if err != nil {
				return apis.NewBadRequestError("failed to update site settings", err)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"homepage_enabled":        settings.HomepageEnabled,
				"landing_page_message":    settings.LandingPageMessage,
				"custom_css":              settings.CustomCSS,
				"ga_measurement_id":       settings.GAMeasurementID,
				"hide_login_button":       settings.HideLoginButton,
				"hide_demo_toggle":        settings.HideDemoToggle,
				"homepage_custom_content": settings.HomepageCustomContent,
				"homepage_section_order":  settings.HomepageSectionOrder,
				"homepage_sections":       settings.HomepageSections,
				"site_nav_enabled":        settings.SiteNavEnabled,
				"site_nav_items":          settings.SiteNavItems,
				"skills_category_order":   settings.SkillsCategoryOrder,
				"site_cta_enabled":        settings.SiteCtaEnabled,
				"favicon":                 settings.Favicon,
				"default_locale":          settings.DefaultLocale,
			})
		})

		// Authenticated: upload favicon
		se.Router.POST("/api/site-settings/favicon", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return apis.NewUnauthorizedError("authentication required", nil)
			}

			settings, err := services.LoadSiteSettings(app)
			if err != nil || settings.Record == nil {
				return apis.NewBadRequestError("failed to load site settings", err)
			}

			// Get uploaded file
			file, fileHeader, err := e.Request.FormFile("favicon")
			if err != nil {
				return apis.NewBadRequestError("favicon file required", err)
			}
			defer file.Close()

			// Create filesystem.File from multipart file
			fsFile, err := filesystem.NewFileFromMultipart(fileHeader)
			if err != nil {
				return apis.NewBadRequestError("failed to process file", err)
			}

			// Set the file on the record
			settings.Record.Set("favicon", fsFile)

			if err := app.Save(settings.Record); err != nil {
				app.Logger().Error("favicon save failed", "error", err)
				return apis.NewBadRequestError("failed to save favicon", err)
			}

			// Verify the file was saved
			faviconFile := settings.Record.GetString("favicon")
			app.Logger().Info("favicon uploaded",
				"filename", faviconFile,
				"collection_id", settings.Record.Collection().Id,
				"record_id", settings.Record.Id)

			if faviconFile == "" {
				app.Logger().Error("favicon save succeeded but filename is empty")
				return apis.NewBadRequestError("favicon file was not saved", nil)
			}

			// Return the /api/favicon endpoint which serves with no-cache headers
			app.Logger().Info("favicon URL returned", "url", "/api/favicon")

			return e.JSON(http.StatusOK, map[string]any{
				"favicon": "/api/favicon",
			})
		})

		// Authenticated: remove favicon
		se.Router.DELETE("/api/site-settings/favicon", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return apis.NewUnauthorizedError("authentication required", nil)
			}

			settings, err := services.LoadSiteSettings(app)
			if err != nil || settings.Record == nil {
				return apis.NewBadRequestError("failed to load site settings", err)
			}

			// Clear the favicon field
			settings.Record.Set("favicon", nil)

			if err := app.Save(settings.Record); err != nil {
				return apis.NewBadRequestError("failed to remove favicon", err)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"favicon": nil,
			})
		})

		// Public: serve favicon with no-cache headers
		// This bypasses PocketBase's aggressive file caching (1 year max-age)
		se.Router.GET("/api/favicon", func(e *core.RequestEvent) error {
			settings, err := services.LoadSiteSettings(app)
			if err != nil || settings.Record == nil {
				return e.Redirect(http.StatusTemporaryRedirect, "/favicon.png")
			}

			faviconFile := settings.Record.GetString("favicon")
			if faviconFile == "" {
				return e.Redirect(http.StatusTemporaryRedirect, "/favicon.png")
			}

			// Build the storage path - PocketBase stores files in {dataDir}/storage/{collection}/{record}/{filename}
			collectionID := settings.Record.Collection().Id
			recordID := settings.Record.Id
			relPath := filepath.Join(collectionID, recordID, faviconFile)

			// Try primary storage first (UPLOADS_DIR), then fallback to pb_data/storage
			var content []byte
			var readErr error

			// Check UPLOADS_DIR environment variable
			uploadsDir := filepath.Clean(filepath.Join("/uploads"))
			primaryPath := filepath.Join(uploadsDir, relPath)
			fallbackPath := filepath.Join(app.DataDir(), "storage", relPath)

			// Try primary storage
			content, readErr = os.ReadFile(primaryPath)
			if readErr != nil {
				// Try fallback storage
				content, readErr = os.ReadFile(fallbackPath)
			}

			if readErr != nil {
				app.Logger().Error("failed to read favicon file",
					"error", readErr,
					"primaryPath", primaryPath,
					"fallbackPath", fallbackPath)
				return e.Redirect(http.StatusTemporaryRedirect, "/favicon.png")
			}

			// Determine content type from extension
			contentType := "image/x-icon"
			ext := strings.ToLower(filepath.Ext(faviconFile))
			switch ext {
			case ".png":
				contentType = "image/png"
			case ".svg":
				contentType = "image/svg+xml"
			case ".gif":
				contentType = "image/gif"
			case ".ico":
				contentType = "image/x-icon"
			}

			// Set headers to prevent caching
			e.Response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			e.Response.Header().Set("Pragma", "no-cache")
			e.Response.Header().Set("Expires", "0")

			return e.Blob(http.StatusOK, contentType, content)
		})

		return se.Next()
	})
}
