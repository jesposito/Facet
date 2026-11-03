package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// BackupConfig holds backup configuration.
type BackupConfig struct {
	// MaxBackups is the maximum number of backups to retain (0 = unlimited)
	MaxBackups int
}

// BackupResult holds the result of a backup operation.
type BackupResult struct {
	Filename string
	Size     int64
	Duration time.Duration
}

// BackupInfo describes an existing backup file.
type BackupInfo struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
}

// RunBackup creates a backup using PocketBase's built-in backup API.
// This runs a WAL checkpoint inside a transaction for a consistent snapshot.
// Backups are stored in {dataDir}/backups/ (inside the already-mapped /data volume).
func RunBackup(app core.App, name string) (*BackupResult, error) {
	start := time.Now()

	if err := app.CreateBackup(context.Background(), name); err != nil {
		return nil, fmt.Errorf("CreateBackup(%q): %w", name, err)
	}

	// Get the size of the created backup
	fsys, err := app.NewBackupsFilesystem()
	if err != nil {
		return &BackupResult{
			Filename: name,
			Duration: time.Since(start),
		}, nil
	}
	defer fsys.Close()

	files, err := fsys.List(name)
	if err != nil || len(files) == 0 {
		return &BackupResult{
			Filename: name,
			Duration: time.Since(start),
		}, nil
	}

	return &BackupResult{
		Filename: name,
		Size:     files[0].Size,
		Duration: time.Since(start),
	}, nil
}

// ListBackups returns all backup files sorted by modification time (newest first).
// Backups are stored in {dataDir}/backups/ by PocketBase. An unreadable backups
// directory is an error, not an empty list: callers such as content recovery
// must not mistake "could not look" for "nothing there".
func ListBackups(app core.App) ([]BackupInfo, error) {
	fsys, err := app.NewBackupsFilesystem()
	if err != nil {
		return nil, fmt.Errorf("open backups filesystem: %w", err)
	}
	defer fsys.Close()

	files, err := fsys.List("")
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	var backups []BackupInfo
	for _, f := range files {
		if f.IsDir {
			continue
		}
		backups = append(backups, BackupInfo{
			Filename: f.Key,
			Size:     f.Size,
			Created:  f.ModTime,
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Created.After(backups[j].Created)
	})

	return backups, nil
}

// EnforceRetention deletes the oldest backups to stay within the limit.
//
// It keeps going after a failed delete so one stuck file doesn't stop the rest
// being pruned, and returns every failure joined together. A retention error
// says nothing about the backup that was just written: callers should report
// the two separately.
func EnforceRetention(app core.App, maxBackups int) error {
	if maxBackups <= 0 {
		return nil
	}

	backups, err := ListBackups(app)
	if err != nil {
		return err
	}
	if len(backups) <= maxBackups {
		return nil
	}

	fsys, err := app.NewBackupsFilesystem()
	if err != nil {
		return fmt.Errorf("open backups filesystem: %w", err)
	}
	defer fsys.Close()

	// Delete oldest backups beyond the limit
	var errs []error
	for _, b := range backups[maxBackups:] {
		if err := fsys.Delete(b.Filename); err != nil {
			errs = append(errs, fmt.Errorf("delete %s: %w", b.Filename, err))
		}
	}
	return errors.Join(errs...)
}
