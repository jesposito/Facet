package hooks

import (
	"net/http"
	"os"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterPlanHooks registers the plan/feature-flag API endpoint.
func RegisterPlanHooks(app *pocketbase.PocketBase, planConfig *services.PlanConfig) {
	// Public endpoint: GET /api/plan
	// Returns current plan configuration and feature flags.
	// Self-hosted instances return all features unlocked.
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/plan", func(e *core.RequestEvent) error {
			demoMode := strings.TrimSpace(os.Getenv("FACET_DEMO_MODE"))

			response := map[string]any{
				"managed":   planConfig.Managed,
				"plan":      planConfig.Plan,
				"features":  planConfig.Features,
				"demo_mode": demoMode == "readonly",
			}

			return e.JSON(http.StatusOK, response)
		})

		return se.Next()
	})
}
