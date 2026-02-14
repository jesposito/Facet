package hooks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
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

		// Start background backup scheduler
		services.SafeGo(app, "backup-scheduler", func() {
			runBackupScheduler(app, maxBackups)
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

			result, err := services.RunBackup(app, name)
			if err != nil {
				app.Logger().Error("backup: manual backup failed",
					"error", err,
					"name", name,
					"dataDir", app.DataDir(),
				)
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Backup failed: " + err.Error(),
				})
			}

			// Enforce retention after manual backup
			services.EnforceRetention(app, maxBackups)

			app.Logger().Info("backup: manual backup complete",
				"filename", result.Filename,
				"size", result.Size,
				"duration", result.Duration.String(),
			)

			return e.JSON(http.StatusOK, map[string]any{
				"filename": result.Filename,
				"size":     result.Size,
				"duration": result.Duration.Milliseconds(),
			})
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
					app.Logger().Error("backup: restore failed", "error", err)
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

func runBackupScheduler(app *pocketbase.PocketBase, maxBackups int) {
	// Wait for startup to complete
	time.Sleep(60 * time.Second)

	backupHour := defaultBackupHour
	if v := os.Getenv("BACKUP_HOUR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < 24 {
			backupHour = n
		}
	}

	for {
		now := time.Now().UTC()
		// Calculate next backup time (today or tomorrow at backupHour:00 UTC)
		next := time.Date(now.Year(), now.Month(), now.Day(), backupHour, 0, 0, 0, time.UTC)
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		sleepDuration := time.Until(next)
		app.Logger().Info("backup: next scheduled backup",
			"at", next.Format(time.RFC3339),
			"in", sleepDuration.Round(time.Minute).String(),
		)

		time.Sleep(sleepDuration)

		// Per-iteration panic recovery - wrap the backup execution
		func() {
			defer func() {
				if r := recover(); r != nil {
					app.Logger().Error("goroutine panic recovered",
						"goroutine", "backup-scheduler",
						"error", fmt.Sprint(r),
						"stack", string(debug.Stack()),
					)
				}
			}()

			name := fmt.Sprintf("facet_auto_%s.zip", time.Now().UTC().Format("20060102_150405"))

			result, err := services.RunBackup(app, name)
			if err != nil {
				app.Logger().Error("backup: scheduled backup failed", "error", err)
				return
			}

			// Enforce retention after scheduled backup
			services.EnforceRetention(app, maxBackups)

			app.Logger().Info("backup: scheduled backup complete",
				"filename", result.Filename,
				"size", result.Size,
				"duration", result.Duration.String(),
			)
		}()
	}
}
