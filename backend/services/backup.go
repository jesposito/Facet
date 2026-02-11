package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase"
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
func RunBackup(app *pocketbase.PocketBase, name string) (*BackupResult, error) {
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
// Backups are stored in {dataDir}/backups/ by PocketBase.
func ListBackups(app *pocketbase.PocketBase) ([]BackupInfo, error) {
	fsys, err := app.NewBackupsFilesystem()
	if err != nil {
		return nil, nil
	}
	defer fsys.Close()

	files, err := fsys.List("")
	if err != nil {
		return nil, nil
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
func EnforceRetention(app *pocketbase.PocketBase, maxBackups int) {
	if maxBackups <= 0 {
		return
	}

	backups, err := ListBackups(app)
	if err != nil || len(backups) <= maxBackups {
		return
	}

	fsys, err := app.NewBackupsFilesystem()
	if err != nil {
		return
	}
	defer fsys.Close()

	// Delete oldest backups beyond the limit
	for _, b := range backups[maxBackups:] {
		fsys.Delete(b.Filename)
	}
}
