package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

const adminCredentialsFilename = ".admin-credentials"

func AdminCredentialsPath(dataDir string) string {
	return filepath.Join(dataDir, adminCredentialsFilename)
}

func writeAdminCredentials(dataDir, email, password string) error {
	path := AdminCredentialsPath(dataDir)
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("admin credentials path is a symlink: %s", path)
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect admin credentials path: %w", err)
	}

	file, err := os.CreateTemp(dataDir, adminCredentialsFilename+"-*")
	if err != nil {
		return fmt.Errorf("create admin credentials file: %w", err)
	}
	temporaryPath := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(temporaryPath)
	}()

	if _, err := fmt.Fprintf(file, "Email: %s\nPassword: %s\n", email, password); err != nil {
		return fmt.Errorf("write admin credentials file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync admin credentials file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close admin credentials file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace admin credentials file: %w", err)
	}
	return nil
}

func removeAdminCredentials(dataDir string) error {
	if err := os.Remove(AdminCredentialsPath(dataDir)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove admin credentials file: %w", err)
	}
	return nil
}
