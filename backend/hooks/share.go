package hooks

import (
	"errors"
	"net/http"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// maxShareTokenUses caps a share link's use limit. Zero means unlimited.
const maxShareTokenUses = 1_000_000

// shareError is a refusal the owner can act on. A transaction returns one to
// roll back and tell the handler what to answer.
type shareError struct {
	status  int
	message string
}

func (e *shareError) Error() string { return e.message }

// checkShareableView refuses a view that a new share link could not open. A
// private view is owner-only and an inactive one is offline.
func checkShareableView(app core.App, viewID string) *shareError {
	view, err := app.FindRecordById("views", viewID)
	if err != nil {
		return &shareError{http.StatusNotFound, "view not found"}
	}
	if view.GetString("visibility") == "private" {
		return &shareError{http.StatusConflict, "private facets are owner-only and can't have share links"}
	}
	if !view.GetBool("is_active") {
		return &shareError{http.StatusConflict, "inactive facets can't have share links"}
	}
	return nil
}

// parseShareExpiry reads an expires_at value. Browsers send an RFC 3339 instant
// and PocketBase stores "2026-01-15 14:30:00.000Z". A value with no zone, such
// as a bare datetime-local, is read as UTC.
func parseShareExpiry(value string) (time.Time, error) {
	var parsed time.Time
	var err error
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05Z07:00", "2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if parsed, err = time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, err
}

// newShareTokenRecord builds an unsaved token record. The raw token it returns
// is shown to the owner once and never stored.
func newShareTokenRecord(app core.App, share *services.ShareService, viewID, name string) (*core.Record, string, error) {
	rawToken, err := share.GenerateToken()
	if err != nil {
		return nil, "", err
	}
	collection, err := app.FindCollectionByNameOrId("share_tokens")
	if err != nil {
		return nil, "", err
	}

	record := core.NewRecord(collection)
	record.Set("view_id", viewID)
	record.Set("token_hash", share.HMACToken(rawToken))
	record.Set("token_prefix", share.TokenPrefix(rawToken)) // For O(1) indexed lookup
	record.Set("name", name)
	record.Set("is_active", true)
	record.Set("use_count", 0)
	return record, rawToken, nil
}

// replaceShareToken swaps an active token for a new one in one transaction, so
// any failure leaves the old link working. It reads the old token inside the
// transaction: transactions run one at a time, so the second of two
// simultaneous requests finds the token already inactive.
func replaceShareToken(app core.App, share *services.ShareService, tokenID string) (*core.Record, string, error) {
	var replacement *core.Record
	var rawToken string
	err := app.RunInTransaction(func(txApp core.App) error {
		old, err := txApp.FindRecordById("share_tokens", tokenID)
		if err != nil {
			return &shareError{http.StatusNotFound, "token not found"}
		}
		if !old.GetBool("is_active") {
			return &shareError{http.StatusConflict, "this link has already been replaced or revoked"}
		}
		expiresAt := old.GetDateTime("expires_at")
		if !expiresAt.IsZero() && !time.Now().Before(expiresAt.Time()) {
			return &shareError{http.StatusConflict, "this link has expired, so its replacement could not be used"}
		}
		viewID := old.GetString("view_id")
		if refused := checkShareableView(txApp, viewID); refused != nil {
			return refused
		}

		replacement, rawToken, err = newShareTokenRecord(txApp, share, viewID, old.GetString("name"))
		if err != nil {
			return err
		}
		if !expiresAt.IsZero() {
			replacement.Set("expires_at", expiresAt)
		}
		if maxUses := old.GetInt("max_uses"); maxUses > 0 {
			replacement.Set("max_uses", maxUses)
		}
		if err := txApp.Save(replacement); err != nil {
			return err
		}

		old.Set("is_active", false)
		return txApp.Save(old)
	})
	if err != nil {
		return nil, "", err
	}
	return replacement, rawToken, nil
}

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

			// Direct lookup by HMAC (unique index) avoids any prefix mismatches
			tokenHMAC := share.HMACToken(req.Token)
			tokenRecord, err := app.FindFirstRecordByFilter(
				"share_tokens",
				"token_hash = {:hash} && is_active = true",
				map[string]interface{}{"hash": tokenHMAC},
			)
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

			// A private view is owner-only, so no token opens it or spends a use on it
			if !viewRecord.GetBool("is_active") || viewRecord.GetString("visibility") == "private" {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			// If a specific view_id was requested, verify the token is for that view
			if req.ViewID != "" && req.ViewID != viewID {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			// Count the use; a concurrent request may have taken the last one
			if !consumeShareTokenUse(app, tokenRecord.Id) {
				return e.JSON(http.StatusOK, invalidResponse)
			}
			// The visit this use admits continues on the view without
			// spending another use.
			issueShareVisitProof(e, crypto, tokenRecord)

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

			if req.MaxUses < 0 || req.MaxUses > maxShareTokenUses {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "max_uses must be between 0 and 1000000"})
			}

			if refused := checkShareableView(app, req.ViewID); refused != nil {
				return e.JSON(refused.status, map[string]string{"error": refused.message})
			}

			record, rawToken, err := newShareTokenRecord(app, share, req.ViewID, req.Name)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			if req.ExpiresAt != nil && *req.ExpiresAt != "" {
				expiresAt, err := parseShareExpiry(*req.ExpiresAt)
				if err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expiration date format"})
				}
				if !expiresAt.After(time.Now()) {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "expiration date must be in the future"})
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

		// Replace a share token, keeping its name, view, expiry and use limit
		se.Router.POST("/api/share/regenerate/{id}", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			tokenID := e.Request.PathValue("id")
			record, rawToken, err := replaceShareToken(app, share, tokenID)
			if err != nil {
				var refused *shareError
				if errors.As(err, &refused) {
					return e.JSON(refused.status, map[string]string{"error": refused.message})
				}
				app.Logger().Warn("Failed to regenerate share token", "error", err, "token_id", tokenID)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to regenerate token"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":    record.Id,
				"token": rawToken, // Only returned once!
				"name":  record.GetString("name"),
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
