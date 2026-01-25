package hooks

import (
	"net/http"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterSiteSettingsHooks exposes site settings for homepage/privacy control.
func RegisterSiteSettingsHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Public: fetch site settings (sanitized)
		se.Router.GET("/api/site-settings", func(e *core.RequestEvent) error {
			settings, err := services.LoadSiteSettings(app)
			if err != nil {
				app.Logger().Error("Failed to load site settings", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load site settings"})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"homepage_enabled":        settings.HomepageEnabled,
				"landing_page_message":   settings.LandingPageMessage,
				"custom_css":             settings.CustomCSS,
				"ga_measurement_id":      settings.GAMeasurementID,
				"hide_login_button":      settings.HideLoginButton,
				"hide_demo_toggle":       settings.HideDemoToggle,
				"homepage_custom_content": settings.HomepageCustomContent,
			})
		})

		// Authenticated: update site settings
		se.Router.PUT("/api/site-settings", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return apis.NewUnauthorizedError("authentication required", nil)
			}

			var req struct {
				HomepageEnabled       *bool                              `json:"homepage_enabled"`
				LandingPageMessage    string                             `json:"landing_page_message"`
				CustomCSS             string                             `json:"custom_css"`
				GAMeasurementID       string                             `json:"ga_measurement_id"`
				HideLoginButton       *bool                              `json:"hide_login_button"`
				HideDemoToggle        *bool                              `json:"hide_demo_toggle"`
				HomepageCustomContent []services.HomepageCustomContentItem `json:"homepage_custom_content"`
			}

			if err := e.BindBody(&req); err != nil {
				return apis.NewBadRequestError("invalid request body", err)
			}

			updates := make(map[string]any)
			if req.HomepageEnabled != nil {
				updates["homepage_enabled"] = *req.HomepageEnabled
			}
			if req.LandingPageMessage != "" || req.LandingPageMessage == "" {
				// Always allow clearing the message
				msg := strings.TrimSpace(req.LandingPageMessage)
				if len(msg) > 2000 {
					msg = msg[:2000]
				}
				updates["landing_page_message"] = msg
			}
			if req.CustomCSS != "" || req.CustomCSS == "" {
				css := strings.TrimSpace(req.CustomCSS)
				if len(css) > 20000 {
					css = css[:20000]
				}
				updates["custom_css"] = css
			}
			if req.GAMeasurementID != "" || req.GAMeasurementID == "" {
				id := strings.TrimSpace(req.GAMeasurementID)
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

			settings, err := services.UpdateSiteSettings(app, updates, app.Logger())
			if err != nil {
				return apis.NewBadRequestError("failed to update site settings", err)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"homepage_enabled":        settings.HomepageEnabled,
				"landing_page_message":   settings.LandingPageMessage,
				"custom_css":             settings.CustomCSS,
				"ga_measurement_id":      settings.GAMeasurementID,
				"hide_login_button":      settings.HideLoginButton,
				"hide_demo_toggle":       settings.HideDemoToggle,
				"homepage_custom_content": settings.HomepageCustomContent,
			})
		})

		return se.Next()
	})
}
