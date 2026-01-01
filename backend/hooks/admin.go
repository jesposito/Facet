package hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterAdminHooks registers admin-only API endpoints
func RegisterAdminHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Load demo data (Merlin Ambrosius)
		// POST /api/admin/demo/load
		se.Router.POST("/api/admin/demo/load", func(e *core.RequestEvent) error {
			// Check if profile data already exists
			count, _ := app.CountRecords("profile")
			if count > 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Profile data already exists. Clear data first.",
				})
			}

			// Seed demo data
			if err := SeedDemoData(app); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Demo data loaded (Merlin Ambrosius)",
			})
		}).Bind(apis.RequireAuth())

		// Clear all data
		// POST /api/admin/demo/clear
		se.Router.POST("/api/admin/demo/clear", func(e *core.RequestEvent) error {
			if err := ClearAllData(app); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "All data cleared",
			})
		}).Bind(apis.RequireAuth())

		// Check if profile has data (for UI state)
		// GET /api/admin/demo/status
		se.Router.GET("/api/admin/demo/status", func(e *core.RequestEvent) error {
			profileCount, _ := app.CountRecords("profile")
			experienceCount, _ := app.CountRecords("experience")
			projectsCount, _ := app.CountRecords("projects")

			return e.JSON(http.StatusOK, map[string]interface{}{
				"has_data":    profileCount > 0,
				"profile":     profileCount,
				"experience":  experienceCount,
				"projects":    projectsCount,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
