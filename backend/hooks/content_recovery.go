package hooks

import (
	"os"
	"path/filepath"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// contentRecoveryMarker lives in the mapped /data volume so the one-time recovery
// runs once per instance and never again (survives restarts and image updates).
const contentRecoveryMarker = ".facet_content_recovery_443.done"

// RegisterContentRecovery wires up a one-time, best-effort content recovery for
// issue #443 (editor flattened paragraph spacing). It runs in the background on
// the first boot after the fix lands, is guarded by a marker file, and can never
// block startup or crash the process (SafeGo recovers panics; all errors are
// logged, never fatal). Set CONTENT_RECOVERY_DISABLED=1 to skip entirely.
func RegisterContentRecovery(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		services.SafeGo(app, "content-recovery", func() {
			runContentRecoveryOnce(app)
		})
		return se.Next()
	})
}

func runContentRecoveryOnce(app *pocketbase.PocketBase) {
	if os.Getenv("CONTENT_RECOVERY_DISABLED") == "1" {
		return
	}

	marker := filepath.Join(app.DataDir(), contentRecoveryMarker)
	if _, err := os.Stat(marker); err == nil {
		return // already completed on a previous boot
	}

	// Let startup settle before doing backup I/O.
	time.Sleep(30 * time.Second)

	report, err := services.RunContentRecovery(app, true)
	if err != nil {
		// Non-fatal: log and leave the marker absent so it retries next boot.
		app.Logger().Error("content-recovery: pass failed, will retry next boot", "error", err)
		return
	}

	if report.Restored > 0 {
		app.Logger().Info("content-recovery: restored flattened content (#443)",
			"restored", report.Restored,
			"scanned", report.Scanned,
			"skipped_live_edits", report.Skipped,
			"safety_backup", report.BackupUsed,
		)
	} else {
		app.Logger().Info("content-recovery: nothing to restore", "scanned", report.Scanned, "skipped_live_edits", report.Skipped)
	}

	if err := os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600); err != nil {
		// If we can't persist the marker the worst case is re-running next boot,
		// which is idempotent (already-structured content is never touched).
		app.Logger().Warn("content-recovery: could not write completion marker", "error", err)
	}
}
