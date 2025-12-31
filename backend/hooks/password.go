package hooks

import (
	"net/http"
	"time"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterPasswordHooks registers password protection endpoints (view-level only)
func RegisterPasswordHooks(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Check password for protected view
		se.Router.POST("/api/password/check", func(e *core.RequestEvent) error {
			var req struct {
				ViewID   string `json:"view_id"`
				Password string `json:"password"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			record, err := app.FindRecordById("views", req.ViewID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}

			visibility := record.GetString("visibility")
			if visibility != "password" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "view is not password protected"})
			}

			passwordHash := record.GetString("password_hash")
			if passwordHash == "" {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "password not configured"})
			}

			if !crypto.CheckPassword(req.Password, passwordHash) {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "incorrect password"})
			}

			// Generate signed JWT for view access (1 hour expiry)
			accessToken, expiresAt, err := crypto.GenerateViewAccessJWT(req.ViewID, 1*time.Hour)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			// Calculate expires_in for client convenience
			expiresIn := int(time.Until(expiresAt).Seconds())

			return e.JSON(http.StatusOK, map[string]interface{}{
				"access_token": accessToken,
				"expires_in":   expiresIn,
			})
		})

		// Set password for view (admin only)
		se.Router.POST("/api/password/set", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				ViewID   string `json:"view_id"`
				Password string `json:"password"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			record, err := app.FindRecordById("views", req.ViewID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}

			hash, err := crypto.HashPassword(req.Password)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			}

			record.Set("password_hash", hash)
			record.Set("visibility", "password")

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "password set"})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
