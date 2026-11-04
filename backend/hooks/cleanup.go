package hooks

import (
	"fmt"
	"net/http"
	"runtime/debug"
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
		services.SafeGo(app, "cleanup-scheduler", func() {
			select {
			case <-time.After(30 * time.Second):
			case <-done:
				return
			}

			// Per-iteration panic recovery for startup cleanup
			func() {
				defer func() {
					if r := recover(); r != nil {
						app.Logger().Error("goroutine panic recovered",
							"goroutine", "cleanup-startup",
							"error", fmt.Sprint(r),
							"stack", string(debug.Stack()),
						)
					}
				}()

				logCleanupResult(app, "startup", services.RunCleanup(app))
			}()

			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					// Per-iteration panic recovery for periodic cleanup
					func() {
						defer func() {
							if r := recover(); r != nil {
								app.Logger().Error("goroutine panic recovered",
									"goroutine", "cleanup-periodic",
									"error", fmt.Sprint(r),
									"stack", string(debug.Stack()),
								)
							}
						}()

						logCleanupResult(app, "periodic", services.RunCleanup(app))
					}()
				case <-done:
					return
				}
			}
		})

		// Admin endpoint for on-demand cleanup. Counts are always returned so
		// the caller can see what was removed. If any step failed, the response
		// is a 500 naming those steps; the underlying errors stay in the log.
		se.Router.POST("/api/admin/cleanup-stale-data", func(e *core.RequestEvent) error {
			result := services.RunCleanup(app)
			body := cleanupResultFields(result)
			if !result.OK() {
				body["status"] = "error"
				body["message"] = "Some cleanup steps failed. Check the server log for details."
				body["failed_steps"] = result.Failed
				return e.JSON(http.StatusInternalServerError, body)
			}
			body["status"] = "ok"
			return e.JSON(http.StatusOK, body)
		}).Bind(apis.RequireAuth("users", core.CollectionNameSuperusers))

		return se.Next()
	})
}

func cleanupResultFields(result services.CleanupResult) map[string]any {
	return map[string]any{
		"expired_share_tokens":        result.ExpiredShareTokens,
		"revoked_share_tokens":        result.RevokedShareTokens,
		"expired_verification_tokens": result.ExpiredVerificationTokens,
		"verified_tokens":             result.VerifiedTokens,
		"failed_exports":              result.FailedExports,
		"stuck_exports":               result.StuckExports,
		"orphaned_uploads":            result.OrphanedUploads,
		"total":                       result.Total(),
	}
}

// logCleanupResult logs a scheduled run. A clean run that removed nothing
// stays quiet; a run with failed steps is always logged.
func logCleanupResult(app *pocketbase.PocketBase, run string, result services.CleanupResult) {
	if result.OK() && result.Total() == 0 {
		return
	}
	attrs := []any{
		"expired_share_tokens", result.ExpiredShareTokens,
		"revoked_share_tokens", result.RevokedShareTokens,
		"expired_verification_tokens", result.ExpiredVerificationTokens,
		"verified_tokens", result.VerifiedTokens,
		"failed_exports", result.FailedExports,
		"stuck_exports", result.StuckExports,
		"orphaned_uploads", result.OrphanedUploads,
		"total", result.Total(),
	}
	if !result.OK() {
		attrs = append(attrs, "failed_steps", result.Failed)
		app.Logger().Error("cleanup: "+run+" cleanup finished with errors", attrs...)
		return
	}
	app.Logger().Info("cleanup: "+run+" cleanup complete", attrs...)
}
