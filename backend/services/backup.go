package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupConfig holds backup configuration.
type BackupConfig struct {
	// DataDir is the PocketBase data directory (contains data.db)
	DataDir string
	// BackupDir is where backups are stored
	BackupDir string
	// MaxBackups is the maximum number of backups to retain (0 = unlimited)
	MaxBackups int
}

// BackupResult holds the result of a backup operation.
type BackupResult struct {
	Filename  string
	Path      string
	Size      int64
	Duration  time.Duration
	Timestamp time.Time
}

// BackupInfo describes an existing backup file.
type BackupInfo struct {
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	Created   time.Time `json:"created"`
}

// RunBackup creates a backup of the PocketBase SQLite database.
// It copies data.db to the backup directory with a timestamped filename.
func RunBackup(config BackupConfig) (*BackupResult, error) {
	start := time.Now()

	// Ensure backup directory exists
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	srcPath := filepath.Join(config.DataDir, "data.db")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found at %s", srcPath)
	}

	timestamp := start.UTC().Format("20060102-150405")
	filename := fmt.Sprintf("facet-backup-%s.db", timestamp)
	dstPath := filepath.Join(config.BackupDir, filename)

	// Copy the database file
	if err := copyFile(srcPath, dstPath); err != nil {
		// Clean up partial file
		os.Remove(dstPath)
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	info, err := os.Stat(dstPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup: %w", err)
	}

	duration := time.Since(start)

	// Enforce retention policy
	if config.MaxBackups > 0 {
		enforceRetention(config.BackupDir, config.MaxBackups)
	}

	return &BackupResult{
		Filename:  filename,
		Path:      dstPath,
		Size:      info.Size(),
		Duration:  duration,
		Timestamp: start.UTC(),
	}, nil
}

// ListBackups returns all backup files sorted by creation time (newest first).
func ListBackups(backupDir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "facet-backup-") || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Filename: entry.Name(),
			Size:     info.Size(),
			Created:  info.ModTime(),
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Created.After(backups[j].Created)
	})

	return backups, nil
}

// enforceRetention deletes the oldest backups to stay within the limit.
func enforceRetention(backupDir string, maxBackups int) {
	backups, err := ListBackups(backupDir)
	if err != nil || len(backups) <= maxBackups {
		return
	}

	// Delete oldest backups beyond the limit
	for _, b := range backups[maxBackups:] {
		os.Remove(filepath.Join(backupDir, b.Filename))
	}
}

// copyFile copies src to dst using buffered I/O.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}
