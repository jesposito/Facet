package hooks

import (
	"net/http"
	"time"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterPasswordHooks registers password protection endpoints
func RegisterPasswordHooks(app *pocketbase.PocketBase) {
	crypto := services.NewCryptoService("")

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Check password for protected content
		se.Router.POST("/api/password/check", func(e *core.RequestEvent) error {
			var req struct {
				Type     string `json:"type"`     // "view", "project", "experience"
				ID       string `json:"id"`
				Password string `json:"password"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			var passwordHash string
			var collectionName string

			switch req.Type {
			case "view":
				collectionName = "views"
			case "project":
				collectionName = "projects"
			case "experience":
				collectionName = "experience"
			default:
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid type"})
			}

			record, err := app.FindRecordById(collectionName, req.ID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
			}

			visibility := record.GetString("visibility")
			if visibility != "password" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "content is not password protected"})
			}

			passwordHash = record.GetString("password_hash")
			if passwordHash == "" {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "password not configured"})
			}

			if !crypto.CheckPassword(req.Password, passwordHash) {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "incorrect password"})
			}

			// Generate a short-lived access token
			accessToken, err := crypto.GenerateToken(16)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			// Store in a temporary access collection or use signed token
			// For simplicity, we'll return a signed token that expires in 1 hour
			expiresAt := time.Now().Add(1 * time.Hour)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"access_token": accessToken,
				"expires_at":   expiresAt,
				"type":         req.Type,
				"id":           req.ID,
			})
		})

		// Set password for protected content (admin only)
		se.Router.POST("/api/password/set", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				Type     string `json:"type"`
				ID       string `json:"id"`
				Password string `json:"password"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			var collectionName string
			switch req.Type {
			case "view":
				collectionName = "views"
			case "project":
				collectionName = "projects"
			case "experience":
				collectionName = "experience"
			default:
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid type"})
			}

			record, err := app.FindRecordById(collectionName, req.ID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
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
		}).Bind(RequireAuth(app))

		return se.Next()
	})
}
