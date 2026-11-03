package hooks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	defaultMaxBackups = 10
	defaultBackupHour = 2 // 2 AM UTC
)

// RegisterBackupHooks sets up the automated backup system.
// Backups are stored in {dataDir}/backups/ (inside the already-mapped /data volume).
// No additional volume mapping is required.
func RegisterBackupHooks(app *pocketbase.PocketBase) {
	// Exclude directories that contain symlinks pointing to /uploads.
	//
	// PocketBase's archive walker (zipAddFS) treats symlinks as regular files:
	// d.IsDir() returns false for symlinks, so the walker tries to Open+Read
	// the symlink target. When the target is a directory (/uploads), this fails
	// with "read <path>/storage: is a directory".
	//
	// The Exclude list matches by path prefix relative to the data dir root.
	// "storage" only matches the root-level symlink /data/storage, but NOT
	// nested symlinks like /data/pb_data/storage or /data/data/pb_data/storage.
	//
	// Excluded entries:
	//   - "storage"  : /data/storage -> /uploads (root-level symlink)
	//   - "pb_data"  : remnant from older PocketBase versions; contains storage -> /uploads
	//   - "data"     : remnant nested data dir from misconfigured installs; also contains symlinks
	//
	// Uploads are backed up separately via the /uploads volume.
	app.OnBackupCreate().BindFunc(func(e *core.BackupEvent) error {
		e.Exclude = append(e.Exclude, "storage", "pb_data", "data")
		return e.Next()
	})

	// Exclude the same entries during restore so the /data/storage -> /uploads
	// symlink survives the directory swap. PocketBase's RestoreBackup moves the
	// current data dir aside and replaces it; excluding "storage" keeps the
	// symlink in place (start.sh only creates it at container boot, not after
	// PocketBase's syscall.Exec()-based restart).
	app.OnBackupRestore().BindFunc(func(e *core.BackupEvent) error {
		e.Exclude = append(e.Exclude, "storage", "pb_data", "data")
		return e.Next()
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		maxBackups := getMaxBackups()

		// Start background backup scheduler. It stops when the app terminates
		// rather than sleeping through shutdown.
		schedCtx, stopScheduler := context.WithCancel(context.Background())
		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			stopScheduler()
			return te.Next()
		})
		services.SafeGo(app, "backup-scheduler", func() {
			newBackupScheduler(app, maxBackups).loop(schedCtx)
		})

		// POST /api/admin/backup - Trigger manual backup
		se.Router.POST("/api/admin/backup", func(e *core.RequestEvent) error {
			// Guard against concurrent backup/restore operations.
			// PocketBase sets this store key during CreateBackup/RestoreBackup;
			// without this check, the nested CreateBackup call returns an error
			// that surfaces as a 500 instead of a clear "try again" message.
			if app.Store().Has(core.StoreKeyActiveBackup) {
				return e.JSON(http.StatusConflict, map[string]string{
					"error": "Another backup or restore operation is already in progress. Try again later.",
				})
			}

			name := fmt.Sprintf("facet_manual_%s.zip", time.Now().UTC().Format("20060102_150405"))

			outcome := performBackup(app, defaultBackupOps, "manual", name, maxBackups)
			if outcome.err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Backup failed: " + outcome.err.Error(),
				})
			}

			resp := map[string]any{
				"filename": outcome.result.Filename,
				"size":     outcome.result.Size,
				"duration": outcome.result.Duration.Milliseconds(),
			}
			// The archive exists, so this is still a successful backup. Pruning
			// old archives is reported alongside it, not as a failure of it.
			if outcome.retentionErr != nil {
				resp["retention_warning"] = "Backup created, but old backups could not be pruned. See System Alerts."
			}
			return e.JSON(http.StatusOK, resp)
		}).Bind(apis.RequireAuth())

		// POST /api/admin/backup/restore - Restore from a backup
		//
		// PocketBase's RestoreBackup swaps the data directory and then
		// calls syscall.Exec() to replace the process. The HTTP response
		// must be flushed before that happens, so the actual restore runs
		// asynchronously after a 1-second delay.
		//
		// After the response, the frontend polls /api/health until the
		// new process is ready, then redirects to login (all sessions are
		// invalidated by the restore).
		se.Router.POST("/api/admin/backup/restore", func(e *core.RequestEvent) error {
			if app.Store().Has(core.StoreKeyActiveBackup) {
				return e.JSON(http.StatusConflict, map[string]string{
					"error": "Another backup or restore operation is already in progress. Try again later.",
				})
			}

			var body struct {
				Filename string `json:"filename"`
			}
			if err := e.BindBody(&body); err != nil || body.Filename == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Missing or invalid filename",
				})
			}

			// Verify backup exists
			backups, err := services.ListBackups(app)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to list backups",
				})
			}
			found := false
			for _, b := range backups {
				if b.Filename == body.Filename {
					found = true
					break
				}
			}
			if !found {
				return e.JSON(http.StatusNotFound, map[string]string{
					"error": "Backup not found",
				})
			}

			app.Logger().Info("backup: restore initiated", "filename", body.Filename)

			// Fire restore asynchronously — the response must be sent before
			// syscall.Exec() replaces the process.
			services.SafeGo(app, "backup-restore", func() {
				time.Sleep(1 * time.Second)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				if err := app.RestoreBackup(ctx, body.Filename); err != nil {
					reportRestoreFailure(app, body.Filename, err)
				}
			})

			return e.JSON(http.StatusOK, map[string]string{
				"message": "Restore initiated. The server will restart momentarily.",
			})
		}).Bind(apis.RequireAuth())

		// GET /api/admin/backup/status - Get backup status and list
		se.Router.GET("/api/admin/backup/status", func(e *core.RequestEvent) error {
			backups, err := services.ListBackups(app)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to list backups",
				})
			}

			var lastBackup *services.BackupInfo
			if len(backups) > 0 {
				lastBackup = &backups[0]
			}

			return e.JSON(http.StatusOK, map[string]any{
				"enabled":     true,
				"backup_dir":  app.DataDir() + "/backups",
				"max_backups": maxBackups,
				"last_backup": lastBackup,
				"backups":     backups,
				"total_count": len(backups),
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

func getMaxBackups() int {
	maxBackups := defaultMaxBackups
	if v := os.Getenv("BACKUP_MAX_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxBackups = n
		}
	}
	return maxBackups
}

// System alert events raised by the backup system. Each is deduplicated by
// CreateSystemAlert, so a nightly failure shows up once until acknowledged.
const (
	backupFailedEvent          = "backup_failed"
	backupRetentionFailedEvent = "backup_retention_failed"
	backupRestoreFailedEvent   = "backup_restore_failed"
)

// backupOps is the pair of storage operations a backup run needs. Production
// uses the services package; tests substitute failures.
type backupOps struct {
	run     func(app core.App, name string) (*services.BackupResult, error)
	enforce func(app core.App, maxBackups int) error
}

var defaultBackupOps = backupOps{
	run:     services.RunBackup,
	enforce: services.EnforceRetention,
}

type backupOutcome struct {
	result       *services.BackupResult
	err          error // the archive was not created
	retentionErr error // the archive was created, but pruning old ones failed
}

// performBackup creates one backup, prunes old ones, and logs and alerts on
// each failure separately. trigger is "manual" or "scheduled".
func performBackup(app core.App, ops backupOps, trigger, name string, maxBackups int) backupOutcome {
	result, err := ops.run(app, name)
	if err != nil {
		app.Logger().Error("backup: "+trigger+" backup failed",
			"error", err,
			"name", name,
			"dataDir", app.DataDir(),
		)
		CreateSystemAlert(app, SystemAlertInput{
			Severity: SeverityCritical,
			Event:    backupFailedEvent,
			Title:    "Backup failed",
			Details: fmt.Sprintf("The %s backup %s was not created. Check free disk space and permissions on the backups folder, then run a manual backup from Settings. Error: %s",
				trigger, name, alertSafeError(err)),
			Metadata: map[string]any{"trigger": trigger, "name": name},
		})
		return backupOutcome{err: err}
	}

	app.Logger().Info("backup: "+trigger+" backup complete",
		"filename", result.Filename,
		"size", result.Size,
		"duration", result.Duration.String(),
	)

	out := backupOutcome{result: result}
	if err := ops.enforce(app, maxBackups); err != nil {
		out.retentionErr = err
		app.Logger().Error("backup: retention cleanup failed",
			"error", err,
			"trigger", trigger,
			"max_backups", maxBackups,
		)
		CreateSystemAlert(app, SystemAlertInput{
			Severity: SeverityWarning,
			Event:    backupRetentionFailedEvent,
			Title:    "Old backups could not be pruned",
			Details: fmt.Sprintf("The %s backup %s was created successfully, but removing backups beyond the limit of %d failed, so the backups folder may keep growing. Check permissions on the backups folder. Error: %s",
				trigger, result.Filename, maxBackups, alertSafeError(err)),
			Metadata: map[string]any{"trigger": trigger, "max_backups": maxBackups},
		})
	}
	return out
}

// reportRestoreFailure logs and alerts on an asynchronous restore that
// returned an error. The admin got a 200 before the restore ran, so without
// this alert the only trace would be in the server log.
func reportRestoreFailure(app core.App, filename string, err error) {
	app.Logger().Error("backup: restore failed", "error", err, "filename", filename)
	CreateSystemAlert(app, SystemAlertInput{
		Severity: SeverityCritical,
		Event:    backupRestoreFailedEvent,
		Title:    "Backup restore failed",
		Details: fmt.Sprintf("Restoring %s did not complete and the server kept running on its existing data. Check that the archive is intact and there is enough free disk space, then try again. Error: %s",
			filename, alertSafeError(err)),
		Metadata: map[string]any{"filename": filename},
	})
}

var (
	// scheme://user:pass@host -> scheme://[redacted]@host
	alertURLUserinfo = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^/@\s]+@`)
	// key=value pairs whose key looks like a credential
	alertSecretPair = regexp.MustCompile(`(?i)\b((?:access[_-]?key|secret|password|passwd|token|api[_-]?key|signature)[A-Za-z_-]*)\s*[=:]\s*\S+`)
)

// alertMaxErrorLen keeps alert details well under the 2000-character field.
const alertMaxErrorLen = 500

// alertSafeError turns an error into a one-line message fit for the admin
// inbox: credentials in URLs or key=value pairs are redacted, newlines (which
// is where wrapped stack-like output starts) are collapsed, and the result is
// truncated. The full error stays in the server log.
func alertSafeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.Join(strings.Fields(err.Error()), " ")
	msg = alertURLUserinfo.ReplaceAllString(msg, "${1}[redacted]@")
	msg = alertSecretPair.ReplaceAllString(msg, "${1}=[redacted]")
	if r := []rune(msg); len(r) > alertMaxErrorLen {
		msg = string(r[:alertMaxErrorLen]) + "…"
	}
	return msg
}

// backupScheduler runs one backup a day at hour:00 UTC. Time and waiting are
// injectable so tests can drive it without real delays.
type backupScheduler struct {
	startDelay time.Duration
	hour       int
	now        func() time.Time
	after      func(time.Duration) <-chan time.Time
	runOnce    func()
	logNext    func(next time.Time, in time.Duration)
	onPanic    func(r any)
}

func newBackupScheduler(app *pocketbase.PocketBase, maxBackups int) *backupScheduler {
	backupHour := defaultBackupHour
	if v := os.Getenv("BACKUP_HOUR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < 24 {
			backupHour = n
		}
	}

	return &backupScheduler{
		startDelay: 60 * time.Second, // wait for startup to complete
		hour:       backupHour,
		now:        time.Now,
		after:      time.After,
		runOnce: func() {
			name := fmt.Sprintf("facet_auto_%s.zip", time.Now().UTC().Format("20060102_150405"))
			performBackup(app, defaultBackupOps, "scheduled", name, maxBackups)
		},
		logNext: func(next time.Time, in time.Duration) {
			app.Logger().Info("backup: next scheduled backup",
				"at", next.Format(time.RFC3339),
				"in", in.Round(time.Minute).String(),
			)
		},
		onPanic: func(r any) {
			app.Logger().Error("goroutine panic recovered",
				"goroutine", "backup-scheduler",
				"error", fmt.Sprint(r),
				"stack", string(debug.Stack()),
			)
			// The stack stays in the log; the alert only says where to look.
			CreateSystemAlert(app, SystemAlertInput{
				Severity: SeverityCritical,
				Event:    backupFailedEvent,
				Title:    "Backup failed",
				Details:  "The scheduled backup stopped with an internal error and no archive was confirmed. The server log has the details (search for backup-scheduler). Run a manual backup from Settings to check.",
				Metadata: map[string]any{"trigger": "scheduled"},
			})
		},
	}
}

// nextBackupTime returns the next hour:00 UTC strictly after now.
func nextBackupTime(now time.Time, hour int) time.Time {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// loop blocks until ctx is cancelled, running a backup at each scheduled time.
// A panic inside one run is recovered so the schedule survives it.
func (s *backupScheduler) loop(ctx context.Context) {
	select {
	case <-s.after(s.startDelay):
	case <-ctx.Done():
		return
	}

	for {
		now := s.now()
		next := nextBackupTime(now, s.hour)
		wait := next.Sub(now)
		s.logNext(next, wait)

		select {
		case <-s.after(wait):
		case <-ctx.Done():
			return
		}

		// Per-iteration panic recovery - wrap the backup execution
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.onPanic(r)
				}
			}()
			s.runOnce()
		}()
	}
}
