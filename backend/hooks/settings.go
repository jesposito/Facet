package hooks

import (
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/stripe/stripe-go/v82/client"
)

// RegisterSiteSettingsHooks exposes site settings for homepage/privacy control.
func RegisterSiteSettingsHooks(app *pocketbase.PocketBase, crypto *services.CryptoService) {
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
				"site_nav_mode":           settings.SiteNavMode,
				"site_nav_position":       settings.SiteNavPosition,
				"site_nav_items":          settings.SiteNavItems,
				"show_avatar":             settings.ShowAvatar,
				"skills_category_order":   settings.SkillsCategoryOrder,
				"site_cta_enabled":        settings.SiteCtaEnabled,
				"footer_cta_enabled":      settings.FooterCtaEnabled,
				"favicon":                 settings.Favicon,
				"default_locale":          settings.DefaultLocale,
				"default_theme_mode":      settings.DefaultThemeMode,
				"design":                  settings.Design,
				"homepage_view_count":     settings.HomepageViewCount,
				"homepage_last_viewed_at": settings.HomepageLastViewedAt,
				"enabled_features":        settings.EnabledFeatures,
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
				SiteNavMode           *string                                   `json:"site_nav_mode"`
				SiteNavPosition       *string                                   `json:"site_nav_position"`
				SiteNavItems          []services.SiteNavItem                    `json:"site_nav_items"`
				ShowAvatar            *bool                                     `json:"show_avatar"`
				SkillsCategoryOrder   []string                                  `json:"skills_category_order"`
				SiteCtaEnabled        *bool                                     `json:"site_cta_enabled"`
				FooterCtaEnabled      *bool                                     `json:"footer_cta_enabled"`
				DefaultLocale         *string                                   `json:"default_locale"`
				DefaultThemeMode      *string                                   `json:"default_theme_mode"`
				Design                *string                                   `json:"design"`
				EnabledFeatures       map[string]bool                           `json:"enabled_features"`
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
			if req.SiteNavMode != nil {
				mode := strings.TrimSpace(*req.SiteNavMode)
				if mode != "bar" && mode != "chips" {
					mode = "bar"
				}
				updates["site_nav_mode"] = mode
			}
			if req.SiteNavPosition != nil {
				pos := strings.TrimSpace(*req.SiteNavPosition)
				if pos != "above" && pos != "below" {
					pos = "below"
				}
				updates["site_nav_position"] = pos
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
			if req.FooterCtaEnabled != nil {
				updates["footer_cta_enabled"] = *req.FooterCtaEnabled
			}
			if req.ShowAvatar != nil {
				updates["show_avatar"] = *req.ShowAvatar
			}
			if req.DefaultLocale != nil {
				locale := strings.TrimSpace(*req.DefaultLocale)
				updates["default_locale"] = locale
			}
			if req.DefaultThemeMode != nil {
				mode := strings.TrimSpace(*req.DefaultThemeMode)
				if mode != "system" && mode != "light" && mode != "dark" {
					mode = "system"
				}
				updates["default_theme_mode"] = mode
			}
			if req.Design != nil {
				design := strings.TrimSpace(*req.Design)
				if design != "soft-premium" {
					design = "classic"
				}
				updates["design"] = design
			}
			if req.EnabledFeatures != nil {
				updates["enabled_features"] = req.EnabledFeatures
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
				"site_nav_mode":           settings.SiteNavMode,
				"site_nav_position":       settings.SiteNavPosition,
				"site_nav_items":          settings.SiteNavItems,
				"show_avatar":             settings.ShowAvatar,
				"skills_category_order":   settings.SkillsCategoryOrder,
				"site_cta_enabled":        settings.SiteCtaEnabled,
				"footer_cta_enabled":      settings.FooterCtaEnabled,
				"favicon":                 settings.Favicon,
				"default_locale":          settings.DefaultLocale,
				"default_theme_mode":      settings.DefaultThemeMode,
				"design":                  settings.Design,
				"enabled_features":        settings.EnabledFeatures,
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

		// ── SMTP Settings API ──────────────────────────────────────

		// GET /api/smtp-settings — read current SMTP config (password masked)
		se.Router.GET("/api/smtp-settings", func(e *core.RequestEvent) error {
			smtp := app.Settings().SMTP
			meta := app.Settings().Meta

			return e.JSON(http.StatusOK, map[string]any{
				"enabled":        smtp.Enabled,
				"host":           smtp.Host,
				"port":           smtp.Port,
				"username":       smtp.Username,
				"password_set":   smtp.Password != "",
				"auth_method":    smtp.AuthMethod,
				"tls":            smtp.TLS,
				"sender_name":    meta.SenderName,
				"sender_address": meta.SenderAddress,
			})
		}).Bind(apis.RequireAuth())

		// PUT /api/smtp-settings — save SMTP config
		se.Router.PUT("/api/smtp-settings", func(e *core.RequestEvent) error {
			var req struct {
				Enabled       *bool   `json:"enabled"`
				Host          *string `json:"host"`
				Port          *int    `json:"port"`
				Username      *string `json:"username"`
				Password      *string `json:"password"`
				AuthMethod    *string `json:"auth_method"`
				TLS           *bool   `json:"tls"`
				SenderName    *string `json:"sender_name"`
				SenderAddress *string `json:"sender_address"`
			}

			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			settings := app.Settings()

			if req.Enabled != nil {
				settings.SMTP.Enabled = *req.Enabled
			}
			if req.Host != nil {
				settings.SMTP.Host = strings.TrimSpace(*req.Host)
			}
			if req.Port != nil {
				settings.SMTP.Port = *req.Port
			}
			if req.Username != nil {
				settings.SMTP.Username = strings.TrimSpace(*req.Username)
			}
			// Only update password if explicitly provided (non-nil and non-empty)
			if req.Password != nil && *req.Password != "" {
				settings.SMTP.Password = *req.Password
			}
			if req.AuthMethod != nil {
				settings.SMTP.AuthMethod = strings.ToUpper(strings.TrimSpace(*req.AuthMethod))
			}
			if req.TLS != nil {
				settings.SMTP.TLS = *req.TLS
			}
			if req.SenderName != nil {
				settings.Meta.SenderName = strings.TrimSpace(*req.SenderName)
			}
			if req.SenderAddress != nil {
				settings.Meta.SenderAddress = strings.TrimSpace(*req.SenderAddress)
			}

			if err := app.Save(settings); err != nil {
				app.Logger().Error("Failed to save SMTP settings", "error", err)
				return apis.NewBadRequestError("failed to save SMTP settings", err)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"enabled":        settings.SMTP.Enabled,
				"host":           settings.SMTP.Host,
				"port":           settings.SMTP.Port,
				"username":       settings.SMTP.Username,
				"password_set":   settings.SMTP.Password != "",
				"auth_method":    settings.SMTP.AuthMethod,
				"tls":            settings.SMTP.TLS,
				"sender_name":    settings.Meta.SenderName,
				"sender_address": settings.Meta.SenderAddress,
			})
		}).Bind(apis.RequireAuth())

		// ── Stripe Settings API ─────────────────────────────────────

		// GET /api/stripe-settings — returns status (never returns actual keys)
		se.Router.GET("/api/stripe-settings", func(e *core.RequestEvent) error {
			// Check DB first
			dbCfg := services.GetStripeConfig(app, crypto)
			if dbCfg != nil {
				return e.JSON(http.StatusOK, map[string]any{
					"configured":         dbCfg.SecretKey != "" || dbCfg.WebhookSecret != "",
					"secret_key_set":     dbCfg.SecretKey != "",
					"webhook_secret_set": dbCfg.WebhookSecret != "",
					"source":             "database",
				})
			}

			// Fall back to env vars
			envKey := os.Getenv("STRIPE_SECRET_KEY")
			envWebhook := os.Getenv("STRIPE_WEBHOOK_SECRET")
			if envKey != "" || envWebhook != "" {
				return e.JSON(http.StatusOK, map[string]any{
					"configured":         true,
					"secret_key_set":     envKey != "",
					"webhook_secret_set": envWebhook != "",
					"source":             "environment",
				})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"configured":         false,
				"secret_key_set":     false,
				"webhook_secret_set": false,
				"source":             "none",
			})
		}).Bind(apis.RequireAuth())

		// PUT /api/stripe-settings — encrypt and store Stripe keys
		se.Router.PUT("/api/stripe-settings", func(e *core.RequestEvent) error {
			var req struct {
				SecretKey     *string `json:"secret_key"`
				WebhookSecret *string `json:"webhook_secret"`
			}

			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			secretKey := ""
			webhookSecret := ""
			if req.SecretKey != nil {
				secretKey = strings.TrimSpace(*req.SecretKey)
			}
			if req.WebhookSecret != nil {
				webhookSecret = strings.TrimSpace(*req.WebhookSecret)
			}

			if secretKey == "" && webhookSecret == "" {
				return apis.NewBadRequestError("at least one of secret_key or webhook_secret is required", nil)
			}

			if err := services.SaveStripeConfig(app, crypto, secretKey, webhookSecret); err != nil {
				app.Logger().Error("Failed to save Stripe config", "error", err)
				return apis.NewBadRequestError("failed to save Stripe settings", err)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"success":            true,
				"secret_key_set":     secretKey != "",
				"webhook_secret_set": webhookSecret != "",
				"source":             "database",
			})
		}).Bind(apis.RequireAuth())

		// DELETE /api/stripe-settings — clear stored keys (reverts to env var fallback)
		se.Router.DELETE("/api/stripe-settings", func(e *core.RequestEvent) error {
			if err := services.ClearStripeConfig(app); err != nil {
				app.Logger().Error("Failed to clear Stripe config", "error", err)
				return apis.NewBadRequestError("failed to clear Stripe settings", err)
			}

			// Report what's available after clearing
			envKey := os.Getenv("STRIPE_SECRET_KEY")
			envWebhook := os.Getenv("STRIPE_WEBHOOK_SECRET")
			source := "none"
			if envKey != "" || envWebhook != "" {
				source = "environment"
			}

			return e.JSON(http.StatusOK, map[string]any{
				"success":            true,
				"configured":         envKey != "" || envWebhook != "",
				"secret_key_set":     envKey != "",
				"webhook_secret_set": envWebhook != "",
				"source":             source,
			})
		}).Bind(apis.RequireAuth())

		// POST /api/stripe-settings/test — test the configured Stripe key
		se.Router.POST("/api/stripe-settings/test", func(e *core.RequestEvent) error {
			// Resolve key: DB first, then env
			stripeKey := services.GetStripeSecretKey(app, crypto)
			if stripeKey == "" {
				stripeKey = os.Getenv("STRIPE_SECRET_KEY")
			}

			if stripeKey == "" {
				return e.JSON(http.StatusOK, map[string]any{
					"success": false,
					"error":   "Stripe secret key is not configured.",
				})
			}

			// Test the key by calling stripe.Account.Get()
			sc := &client.API{}
			sc.Init(stripeKey, nil)

			_, err := sc.Accounts.Get()
			if err != nil {
				app.Logger().Error("Stripe key test failed", "error", err)
				return e.JSON(http.StatusOK, map[string]any{
					"success": false,
					"error":   err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"success": true,
			})
		}).Bind(apis.RequireAuth())

		// POST /api/smtp-settings/test — send a test email
		se.Router.POST("/api/smtp-settings/test", func(e *core.RequestEvent) error {
			var req struct {
				Recipient string `json:"recipient"`
			}
			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			recipient := strings.TrimSpace(req.Recipient)
			if recipient == "" {
				// Default to the authenticated user's email
				if e.Auth != nil {
					recipient = e.Auth.Email()
				}
			}
			if recipient == "" {
				return apis.NewBadRequestError("recipient email is required", nil)
			}

			if !app.Settings().SMTP.Enabled {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"success": false,
					"error":   "SMTP is not enabled. Save your SMTP settings first.",
				})
			}

			message := &mailer.Message{
				From: mail.Address{
					Name:    app.Settings().Meta.SenderName,
					Address: app.Settings().Meta.SenderAddress,
				},
				To:      []mail.Address{{Address: recipient}},
				Subject: "Facet — Test Email",
				HTML:    "<h2>SMTP Configuration Test</h2><p>If you're reading this, your SMTP settings are working correctly!</p><p>Sent from your Facet instance.</p>",
				Text:    "SMTP Configuration Test\n\nIf you're reading this, your SMTP settings are working correctly!\n\nSent from your Facet instance.",
			}

			client := app.NewMailClient()
			if err := client.Send(message); err != nil {
				app.Logger().Error("SMTP test email failed", "error", err, "recipient", recipient)
				return e.JSON(http.StatusOK, map[string]any{
					"success": false,
					"error":   err.Error(),
				})
			}

			app.Logger().Info("SMTP test email sent", "recipient", recipient)
			return e.JSON(http.StatusOK, map[string]any{
				"success": true,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
