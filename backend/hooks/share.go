package hooks

import (
	"net/http"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterShareHooks registers share token related endpoints
func RegisterShareHooks(app *pocketbase.PocketBase, share *services.ShareService, crypto *services.CryptoService, rl *services.RateLimitService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Validate a share token
		// NOTE: All failure cases return the same generic error to prevent oracle attacks.
		// Specific error details (expired, wrong view, etc.) are not exposed to callers.
		// Rate limited: moderate tier (10/min) to prevent token enumeration
		se.Router.POST("/api/share/validate", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			// Generic error response - same for all failure modes to prevent information leakage
			invalidResponse := services.ShareTokenValidation{
				Valid: false,
				Error: "invalid token",
			}

			var req struct {
				Token  string `json:"token"`
				ViewID string `json:"view_id"` // Optional: validate token is for specific view
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
			}

			// O(1) lookup using token_prefix index
			prefix := share.TokenPrefix(req.Token)

			// Query by prefix for efficient lookup (indexed)
			candidates, err := app.FindRecordsByFilter(
				"share_tokens",
				"token_prefix = {:prefix} && is_active = true",
				"-created",
				10, // Prefix collisions are rare; limit for safety
				0,
				map[string]interface{}{"prefix": prefix},
			)

			// Fallback to legacy lookup if no prefix-based results (for tokens created before migration)
			if err != nil || len(candidates) == 0 {
				// Try legacy O(n) scan for old tokens without prefix
				candidates, err = app.FindRecordsByFilter(
					"share_tokens",
					"(token_prefix = '' || token_prefix IS NULL) && is_active = true",
					"-created",
					100,
					0,
					nil,
				)
			}

			// Find the matching token using constant-time HMAC comparison
			var tokenRecord *core.Record
			for _, record := range candidates {
				storedHMAC := record.GetString("token_hash")
				if share.ValidateTokenHMAC(req.Token, storedHMAC) {
					tokenRecord = record
					break
				}
			}

			if err != nil || tokenRecord == nil {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			// Check expiration - return same error to prevent timing oracle
			expiresAt := tokenRecord.GetDateTime("expires_at")
			if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			// Check max uses - return same error to prevent oracle
			useCount := tokenRecord.GetInt("use_count")
			maxUses := tokenRecord.GetInt("max_uses")
			if maxUses > 0 && useCount >= maxUses {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			// Get view info
			viewID := tokenRecord.GetString("view_id")
			viewRecord, err := app.FindRecordById("views", viewID)
			if err != nil {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			if !viewRecord.GetBool("is_active") {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			// If a specific view_id was requested, verify the token is for that view
			if req.ViewID != "" && req.ViewID != viewID {
				return e.JSON(http.StatusOK, invalidResponse)
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
		}))

		// Generate a new share token
		se.Router.POST("/api/share/generate", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				ViewID    string  `json:"view_id"`
				Name      string  `json:"name"`
				ExpiresAt *string `json:"expires_at"` // Accept as string, parse below
				MaxUses   int     `json:"max_uses"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request", "details": err.Error()})
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
			tokenPrefix := share.TokenPrefix(rawToken)

			// Create token record
			collection, err := app.FindCollectionByNameOrId("share_tokens")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("view_id", req.ViewID)
			record.Set("token_hash", tokenHMAC)
			record.Set("token_prefix", tokenPrefix) // For O(1) indexed lookup
			record.Set("name", req.Name)
			record.Set("is_active", true)
			record.Set("use_count", 0)

			if req.ExpiresAt != nil && *req.ExpiresAt != "" {
				// Parse datetime-local format (e.g., "2024-01-15T14:30")
				// Try multiple formats
				expiresAt, err := time.Parse("2006-01-02T15:04", *req.ExpiresAt)
				if err != nil {
					// Try with seconds
					expiresAt, err = time.Parse("2006-01-02T15:04:05", *req.ExpiresAt)
				}
				if err != nil {
					// Try RFC3339
					expiresAt, err = time.Parse(time.RFC3339, *req.ExpiresAt)
				}
				if err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expiration date format"})
				}
				record.Set("expires_at", expiresAt)
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
