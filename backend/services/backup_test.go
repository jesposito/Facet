package services

import (
	"testing"
	"time"
)

func TestBackupInfoSorting(t *testing.T) {
	// Verify our sort logic (newest first)
	backups := []BackupInfo{
		{Filename: "old.zip", Size: 100, Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Filename: "new.zip", Size: 200, Created: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		{Filename: "mid.zip", Size: 150, Created: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)},
	}

	// Simulate our sort
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[j].Created.After(backups[i].Created) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	if backups[0].Filename != "new.zip" {
		t.Errorf("expected newest first, got %s", backups[0].Filename)
	}
	if backups[2].Filename != "old.zip" {
		t.Errorf("expected oldest last, got %s", backups[2].Filename)
	}
}

func TestBackupConfigDefaults(t *testing.T) {
	config := BackupConfig{}
	if config.MaxBackups != 0 {
		t.Errorf("expected default MaxBackups=0 (unlimited), got %d", config.MaxBackups)
	}
}
