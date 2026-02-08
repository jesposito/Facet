package hooks

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	defaultBackupDir    = "/backups"
	defaultMaxBackups   = 10
	defaultBackupHour   = 2 // 2 AM UTC
)

// RegisterBackupHooks sets up the automated backup system.
func RegisterBackupHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		config := getBackupConfig(app)

		// Start background backup scheduler
		go runBackupScheduler(app, config)

		// POST /api/admin/backup - Trigger manual backup
		se.Router.POST("/api/admin/backup", func(e *core.RequestEvent) error {
			result, err := services.RunBackup(config)
			if err != nil {
				app.Logger().Error("backup: manual backup failed", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Backup failed: " + err.Error(),
				})
			}

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

		// GET /api/admin/backup/status - Get backup status and list
		se.Router.GET("/api/admin/backup/status", func(e *core.RequestEvent) error {
			backups, err := services.ListBackups(config.BackupDir)
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
				"backup_dir":  config.BackupDir,
				"max_backups": config.MaxBackups,
				"last_backup": lastBackup,
				"backups":     backups,
				"total_count": len(backups),
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

func getBackupConfig(app *pocketbase.PocketBase) services.BackupConfig {
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = defaultBackupDir
	}

	maxBackups := defaultMaxBackups
	if v := os.Getenv("BACKUP_MAX_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxBackups = n
		}
	}

	// Determine data directory from PocketBase's configured dir
	dataDir := app.DataDir()

	return services.BackupConfig{
		DataDir:    dataDir,
		BackupDir:  backupDir,
		MaxBackups: maxBackups,
	}
}

func runBackupScheduler(app *pocketbase.PocketBase, config services.BackupConfig) {
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

		result, err := services.RunBackup(config)
		if err != nil {
			app.Logger().Error("backup: scheduled backup failed", "error", err)
			continue
		}

		app.Logger().Info("backup: scheduled backup complete",
			"filename", result.Filename,
			"size", result.Size,
			"duration", result.Duration.String(),
		)
	}
}
