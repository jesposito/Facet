package hooks

import (
	"fmt"
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
				fmt.Printf("[API /api/site-settings] failed to load: %v\n", err)
				app.Logger().Error("Failed to load site settings", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load site settings"})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"homepage_enabled":     settings.HomepageEnabled,
				"landing_page_message": settings.LandingPageMessage,
			})
		})

		// Authenticated: update site settings
		se.Router.PUT("/api/site-settings", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return apis.NewUnauthorizedError("authentication required", nil)
			}

			var req struct {
				HomepageEnabled    *bool  `json:"homepage_enabled"`
				LandingPageMessage string `json:"landing_page_message"`
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

			settings, err := services.UpdateSiteSettings(app, updates, app.Logger())
			if err != nil {
				return apis.NewBadRequestError("failed to update site settings", err)
			}

			return e.JSON(http.StatusOK, map[string]any{
				"homepage_enabled":     settings.HomepageEnabled,
				"landing_page_message": settings.LandingPageMessage,
			})
		})

		return se.Next()
	})
}
