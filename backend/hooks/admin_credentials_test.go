package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAdminCredentialsUsesPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	path := AdminCredentialsPath(dir)

	if err := writeAdminCredentials(dir, "owner@example.com", "test-password"); err != nil {
		t.Fatal(err)
	}

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

func TestWriteAdminCredentialsReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := AdminCredentialsPath(dir)
	if err := os.WriteFile(path, []byte("old credentials"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeAdminCredentials(dir, "new-owner@example.com", "replacement-password"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("replacement mode = %04o, want 0600", got)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "Email: new-owner@example.com\nPassword: replacement-password\n"
	if string(contents) != want {
		t.Fatal("existing credentials file was not replaced")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatal("temporary credentials files were left behind")
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

	if err := writeAdminCredentials(dir, "owner@example.com", "test-password"); err == nil {
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

	if err := writeAdminCredentials(notADirectory, "owner@example.com", "test-password"); err == nil {
		t.Fatal("writing credentials in an invalid data directory succeeded")
	}
}
