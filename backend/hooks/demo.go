package hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterDemoHooks registers demo-related API endpoints
func RegisterDemoHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// One-click demo login endpoint
		se.Router.POST("/api/demo/login", func(e *core.RequestEvent) error {
			// Get demo user email (uses same logic as seed - defined in seed.go)
			demoEmail := getSeedAdminEmail("admin@example.com") // defined in seed.go, same package

			// Authenticate the demo user
			record, err := app.FindAuthRecordByEmail("users", demoEmail)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{
					"error": "Demo account not found. Have you seeded demo data?",
				})
			}

			// Generate auth token
			token, err := record.NewAuthToken()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to generate auth token",
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"token":  token,
				"record": record,
			})
		})

		return se.Next()
	})
}
