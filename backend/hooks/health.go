package hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterHealthCheck registers the health check endpoint
func RegisterHealthCheck(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/health", func(e *core.RequestEvent) error {
			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":  "healthy",
				"version": "1.0.0",
			})
		})
		return se.Next()
	})
}
