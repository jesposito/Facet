package hooks

import (
	"net/http"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterCleanupHooks starts a background goroutine that periodically deletes
// expired share tokens, used verification tokens, and failed view exports.
// It also registers an admin endpoint for on-demand cleanup.
func RegisterCleanupHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		done := make(chan struct{})

		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			close(done)
			return te.Next()
		})

		// Run cleanup once on startup (after a short delay to let migrations finish),
		// then periodically every 24 hours.
		go func() {
			select {
			case <-time.After(30 * time.Second):
			case <-done:
				return
			}

			result := services.RunCleanup(app)
			if result.Total() > 0 {
				app.Logger().Info("cleanup: startup cleanup complete",
					"expired_share_tokens", result.ExpiredShareTokens,
					"revoked_share_tokens", result.RevokedShareTokens,
					"expired_verification_tokens", result.ExpiredVerificationTokens,
					"verified_tokens", result.VerifiedTokens,
					"failed_exports", result.FailedExports,
					"stuck_exports", result.StuckExports,
					"total", result.Total(),
				)
			}

			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					result := services.RunCleanup(app)
					if result.Total() > 0 {
						app.Logger().Info("cleanup: periodic cleanup complete",
							"expired_share_tokens", result.ExpiredShareTokens,
							"revoked_share_tokens", result.RevokedShareTokens,
							"expired_verification_tokens", result.ExpiredVerificationTokens,
							"verified_tokens", result.VerifiedTokens,
							"failed_exports", result.FailedExports,
							"stuck_exports", result.StuckExports,
							"total", result.Total(),
						)
					}
				case <-done:
					return
				}
			}
		}()

		// Admin endpoint for on-demand cleanup
		se.Router.POST("/api/admin/cleanup-stale-data", func(e *core.RequestEvent) error {
			result := services.RunCleanup(app)
			return e.JSON(http.StatusOK, map[string]any{
				"status":                      "ok",
				"expired_share_tokens":        result.ExpiredShareTokens,
				"revoked_share_tokens":        result.RevokedShareTokens,
				"expired_verification_tokens": result.ExpiredVerificationTokens,
				"verified_tokens":             result.VerifiedTokens,
				"failed_exports":              result.FailedExports,
				"stuck_exports":               result.StuckExports,
				"total":                        result.Total(),
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}
