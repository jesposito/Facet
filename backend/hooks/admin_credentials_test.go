package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestWriteAdminCredentialsUsesPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	path := AdminCredentialsPath(dir)

	writeTestAdminCredentials(t, dir, "owner@example.com", "test-password")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("credentials file mode = %04o, want 0600", got)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "Email: owner@example.com\nPassword: test-password\n"
	if string(contents) != want {
		t.Fatal("credentials file does not contain the expected email and password")
	}
}

// writeTestAdminCredentials writes a handoff file that the test expects to
// succeed and releases it when the test ends.
func writeTestAdminCredentials(t *testing.T, dir, email, password string) {
	t.Helper()
	handoff, err := writeAdminCredentials(dir, email, password)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handoff.release)
}

// assertOnlyEntry fails if anything other than name is left in dir, such as
// a temporary credentials file.
func assertOnlyEntry(t *testing.T, dir, name string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != name {
		t.Fatalf("data directory has %d entries, want only %s", len(entries), name)
	}
}

func TestWriteAdminCredentialsKeepsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := AdminCredentialsPath(dir)
	original := []byte("Email: earlier@example.com\nPassword: unread-password\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = writeAdminCredentials(dir, "new-owner@example.com", "replacement-password")
	if err == nil {
		t.Fatal("writing credentials replaced an unread credentials file")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "delete it") {
		t.Fatalf("error does not tell the operator what to do: %v", err)
	}
	if strings.Contains(err.Error(), "replacement-password") || strings.Contains(err.Error(), "unread-password") {
		t.Fatal("error contains a password")
	}

	after, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || after.Mode() != before.Mode() {
		t.Fatal("existing credentials file was replaced")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatal("existing credentials file was modified")
	}
	assertOnlyEntry(t, dir, filepath.Base(path))
}

func TestWriteAdminCredentialsLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	writeTestAdminCredentials(t, dir, "owner@example.com", "test-password")
	assertOnlyEntry(t, dir, adminCredentialsFilename)
}

func TestWriteAdminCredentialsAllowsOnlyOneConcurrentWriter(t *testing.T) {
	dir := t.TempDir()
	const writers = 16
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners []string
		start   = make(chan struct{})
	)
	for i := 0; i < writers; i++ {
		password := fmt.Sprintf("password-%02d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if handoff, err := writeAdminCredentials(dir, "owner@example.com", password); err == nil {
				handoff.release()
				mu.Lock()
				winners = append(winners, password)
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(winners) != 1 {
		t.Fatalf("%d concurrent writers succeeded, want exactly 1", len(winners))
	}
	contents, err := os.ReadFile(AdminCredentialsPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "Email: owner@example.com\nPassword: "+winners[0]+"\n" {
		t.Fatal("credentials file does not belong to the writer that succeeded")
	}
	assertOnlyEntry(t, dir, adminCredentialsFilename)
}

func TestRemoveAdminCredentialsRemovesOwnFile(t *testing.T) {
	dir := t.TempDir()
	created, err := writeAdminCredentials(dir, "owner@example.com", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := removeAdminCredentials(dir, created); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(AdminCredentialsPath(dir)); !os.IsNotExist(err) {
		t.Fatal("credentials file created by this write was not removed")
	}
}

func TestRemoveAdminCredentialsAcceptsAlreadyDeletedFile(t *testing.T) {
	dir := t.TempDir()
	created, err := writeAdminCredentials(dir, "owner@example.com", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(AdminCredentialsPath(dir)); err != nil {
		t.Fatal(err)
	}
	if err := removeAdminCredentials(dir, created); err != nil {
		t.Fatalf("cleanup failed after the file was already deleted: %v", err)
	}
}

func TestRemoveAdminCredentialsKeepsReplacementFile(t *testing.T) {
	dir := t.TempDir()
	path := AdminCredentialsPath(dir)
	created, err := writeAdminCredentials(dir, "owner@example.com", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	// Someone reads and deletes the file, and another reset writes a new one.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	other := []byte("Email: other@example.com\nPassword: other-password\n")
	if err := os.WriteFile(path, other, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := removeAdminCredentials(dir, created); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("another operation's credentials file was removed: %v", err)
	}
	if string(contents) != string(other) {
		t.Fatal("another operation's credentials file was modified")
	}
}

func TestRemoveAdminCredentialsKeepsSymlink(t *testing.T) {
	dir := t.TempDir()
	path := AdminCredentialsPath(dir)
	created, err := writeAdminCredentials(dir, "owner@example.com", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	// Swap the file for a symlink pointing at the very same inode.
	kept := filepath.Join(dir, "kept")
	if err := os.Rename(path, kept); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(kept, path); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	if err := removeAdminCredentials(dir, created); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("cleanup removed or replaced a symlink")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatal("cleanup removed the symlink target")
	}
}

func TestWriteAdminCredentialsRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	original := []byte("do not change")
	if err := os.WriteFile(outside, original, 0o600); err != nil {
		t.Fatal(err)
	}
	path := AdminCredentialsPath(dir)
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	if _, err := writeAdminCredentials(dir, "owner@example.com", "test-password"); err == nil {
		t.Fatal("writing credentials through a symlink succeeded")
	}

	contents, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatal("symlink target was modified")
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("credentials symlink was replaced")
	}
}

func TestWriteAdminCredentialsFailsForInvalidDataDir(t *testing.T) {
	dir := t.TempDir()
	notADirectory := filepath.Join(dir, "data")
	if err := os.WriteFile(notADirectory, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := writeAdminCredentials(notADirectory, "owner@example.com", "test-password"); err == nil {
		t.Fatal("writing credentials in an invalid data directory succeeded")
	}
}
