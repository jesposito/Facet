package hooks

import (
	"net/http"
	"time"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterShareHooks registers share token related endpoints
func RegisterShareHooks(app *pocketbase.PocketBase, share *services.ShareService, crypto *services.CryptoService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Validate a share token
		se.Router.POST("/api/share/validate", func(e *core.RequestEvent) error {
			var req struct {
				Token string `json:"token"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
			}

			// Find token record by iterating and comparing HMACs
			// We iterate because HMAC lookup isn't direct - we need constant-time comparison
			allTokens, err := app.FindRecordsByFilter(
				"share_tokens",
				"is_active = true",
				"-created",
				1000,
				0,
				nil,
			)

			var tokenRecord *core.Record
			for _, record := range allTokens {
				storedHMAC := record.GetString("token_hash")
				if share.ValidateTokenHMAC(req.Token, storedHMAC) {
					tokenRecord = record
					break
				}
			}

			if err != nil || tokenRecord == nil {
				return e.JSON(http.StatusOK, services.ShareTokenValidation{
					Valid: false,
					Error: "invalid token",
				})
			}

			// Check expiration
			expiresAt := tokenRecord.GetDateTime("expires_at")
			if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
				return e.JSON(http.StatusOK, services.ShareTokenValidation{
					Valid: false,
					Error: "token expired",
				})
			}

			// Check max uses
			useCount := tokenRecord.GetInt("use_count")
			maxUses := tokenRecord.GetInt("max_uses")
			if maxUses > 0 && useCount >= maxUses {
				return e.JSON(http.StatusOK, services.ShareTokenValidation{
					Valid: false,
					Error: "token usage limit reached",
				})
			}

			// Get view info
			viewID := tokenRecord.GetString("view_id")
			viewRecord, err := app.FindRecordById("views", viewID)
			if err != nil {
				return e.JSON(http.StatusOK, services.ShareTokenValidation{
					Valid: false,
					Error: "view not found",
				})
			}

			if !viewRecord.GetBool("is_active") {
				return e.JSON(http.StatusOK, services.ShareTokenValidation{
					Valid: false,
					Error: "view is not active",
				})
			}

			// Update usage
			tokenRecord.Set("use_count", useCount+1)
			tokenRecord.Set("last_used_at", time.Now())
			app.Save(tokenRecord)

			return e.JSON(http.StatusOK, services.ShareTokenValidation{
				Valid:    true,
				ViewID:   viewID,
				ViewSlug: viewRecord.GetString("slug"),
			})
		})

		// Generate a new share token
		se.Router.POST("/api/share/generate", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				ViewID    string     `json:"view_id"`
				Name      string     `json:"name"`
				ExpiresAt *time.Time `json:"expires_at"`
				MaxUses   int        `json:"max_uses"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Verify view exists
			_, err := app.FindRecordById("views", req.ViewID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}

			// Generate token
			rawToken, err := share.GenerateToken()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			tokenHMAC := share.HMACToken(rawToken)

			// Create token record
			collection, err := app.FindCollectionByNameOrId("share_tokens")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("view_id", req.ViewID)
			record.Set("token_hash", tokenHMAC)
			record.Set("name", req.Name)
			record.Set("is_active", true)
			record.Set("use_count", 0)

			if req.ExpiresAt != nil {
				record.Set("expires_at", *req.ExpiresAt)
			}
			if req.MaxUses > 0 {
				record.Set("max_uses", req.MaxUses)
			}

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save token"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":    record.Id,
				"token": rawToken, // Only returned once!
				"name":  req.Name,
			})
		}).Bind(apis.RequireAuth())

		// Revoke a share token
		se.Router.POST("/api/share/revoke/{id}", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			tokenID := e.Request.PathValue("id")
			record, err := app.FindRecordById("share_tokens", tokenID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "token not found"})
			}

			record.Set("is_active", false)
			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to revoke token"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "revoked"})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
