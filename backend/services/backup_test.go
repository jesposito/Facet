package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunBackup(t *testing.T) {
	// Create temp directories
	dataDir := t.TempDir()
	backupDir := t.TempDir()

	// Create a fake database file
	dbPath := filepath.Join(dataDir, "data.db")
	if err := os.WriteFile(dbPath, []byte("test database content"), 0644); err != nil {
		t.Fatal(err)
	}

	config := BackupConfig{
		DataDir:    dataDir,
		BackupDir:  backupDir,
		MaxBackups: 5,
	}

	result, err := RunBackup(config)
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	if result.Filename == "" {
		t.Error("expected non-empty filename")
	}
	if result.Size == 0 {
		t.Error("expected non-zero size")
	}

	// Verify file exists
	if _, err := os.Stat(result.Path); os.IsNotExist(err) {
		t.Error("backup file does not exist")
	}

	// Verify content matches
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "test database content" {
		t.Error("backup content does not match source")
	}
}

func TestRunBackup_MissingDB(t *testing.T) {
	dataDir := t.TempDir()
	backupDir := t.TempDir()

	config := BackupConfig{
		DataDir:   dataDir,
		BackupDir: backupDir,
	}

	_, err := RunBackup(config)
	if err == nil {
		t.Error("expected error for missing database")
	}
}

func TestEnforceRetention(t *testing.T) {
	backupDir := t.TempDir()

	// Create 5 backup files
	for i := 0; i < 5; i++ {
		name := filepath.Join(backupDir, "facet-backup-2026010"+string(rune('1'+i))+"-120000.db")
		if err := os.WriteFile(name, []byte("backup"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	enforceRetention(backupDir, 3)

	backups, err := ListBackups(backupDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(backups) != 3 {
		t.Errorf("expected 3 backups after retention, got %d", len(backups))
	}
}

func TestListBackups_Empty(t *testing.T) {
	backupDir := t.TempDir()

	backups, err := ListBackups(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestListBackups_NonExistentDir(t *testing.T) {
	backups, err := ListBackups("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if backups != nil {
		t.Errorf("expected nil backups for nonexistent dir, got %v", backups)
	}
}

func TestListBackups_IgnoresNonBackupFiles(t *testing.T) {
	backupDir := t.TempDir()

	// Create a backup file and a non-backup file
	os.WriteFile(filepath.Join(backupDir, "facet-backup-20260101-120000.db"), []byte("backup"), 0644)
	os.WriteFile(filepath.Join(backupDir, "random-file.txt"), []byte("not a backup"), 0644)
	os.WriteFile(filepath.Join(backupDir, "facet-backup-readme.md"), []byte("readme"), 0644)

	backups, err := ListBackups(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(backups))
	}
}
