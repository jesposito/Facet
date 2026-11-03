package hooks

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const adminCredentialsFilename = ".admin-credentials"

func AdminCredentialsPath(dataDir string) string {
	return filepath.Join(dataDir, adminCredentialsFilename)
}

// pendingAdminCredentialsError explains why a new handoff was refused. It
// names the file but never its contents.
func pendingAdminCredentialsError(path string) error {
	return fmt.Errorf("an unread admin credentials file already exists at %s; read it, then delete it before resetting a password again (no password was changed)", path)
}

// adminCredentialsHandoff identifies the credentials file one call created.
// It keeps the file open so the filesystem cannot reuse its inode for a
// different file while cleanup is still possible. Call release when done.
type adminCredentialsHandoff struct {
	file *os.File
}

func (h *adminCredentialsHandoff) release() {
	_ = h.file.Close()
}

// writeAdminCredentials publishes a new private handoff file.
//
// It never replaces anything already at the handoff path, whatever its type:
// an unread file may hold the only copy of a working password, possibly for a
// different account. The complete file is written to a temporary name first
// and then hard-linked into place, which fails if the path already exists, so
// concurrent callers cannot overwrite each other and no reader sees a partial
// file.
func writeAdminCredentials(dataDir, email, password string) (*adminCredentialsHandoff, error) {
	path := AdminCredentialsPath(dataDir)
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("admin credentials path is a symlink: %s", path)
		}
		return nil, pendingAdminCredentialsError(path)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect admin credentials path: %w", err)
	}

	// CreateTemp opens the file with mode 0600 and O_EXCL.
	file, err := os.CreateTemp(dataDir, adminCredentialsFilename+"-*")
	if err != nil {
		return nil, fmt.Errorf("create admin credentials file: %w", err)
	}
	temporaryPath := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(temporaryPath)
	}()

	if _, err := fmt.Fprintf(file, "Email: %s\nPassword: %s\n", email, password); err != nil {
		return nil, fmt.Errorf("write admin credentials file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("sync admin credentials file: %w", err)
	}
	created, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect admin credentials file: %w", err)
	}
	held, err := os.Open(temporaryPath)
	if err != nil {
		return nil, fmt.Errorf("open admin credentials file: %w", err)
	}
	handoff := &adminCredentialsHandoff{file: held}
	heldInfo, err := held.Stat()
	if err != nil || !os.SameFile(heldInfo, created) {
		handoff.release()
		return nil, fmt.Errorf("admin credentials file changed while it was written: %s", temporaryPath)
	}
	if err := file.Close(); err != nil {
		handoff.release()
		return nil, fmt.Errorf("close admin credentials file: %w", err)
	}
	if err := os.Link(temporaryPath, path); err != nil {
		handoff.release()
		if errors.Is(err, fs.ErrExist) {
			return nil, pendingAdminCredentialsError(path)
		}
		return nil, fmt.Errorf("publish admin credentials file: %w", err)
	}
	return handoff, nil
}

// removeAdminCredentials deletes the handoff file only if it is still the one
// this handoff created, then releases the handoff. If the path now holds
// anything else, such as a file written by another reset or a symlink, it is
// left alone.
func removeAdminCredentials(dataDir string, handoff *adminCredentialsHandoff) error {
	defer handoff.release()
	created, err := handoff.file.Stat()
	if err != nil {
		return fmt.Errorf("inspect admin credentials file: %w", err)
	}
	path := AdminCredentialsPath(dataDir)
	current, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect admin credentials file: %w", err)
	}
	if !current.Mode().IsRegular() || !os.SameFile(current, created) {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove admin credentials file: %w", err)
	}
	return nil
}
