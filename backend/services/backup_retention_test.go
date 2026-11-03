package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"
)

func newBackupTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	dir := filepath.Join(app.DataDir(), "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Fake archives with ascending mod times: a is oldest.
	base := time.Now().Add(-time.Hour)
	for i, name := range []string{"a.zip", "b.zip", "c.zip"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("not a real backup"), 0o644); err != nil {
			t.Fatal(err)
		}
		mt := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	return app, dir
}

func TestEnforceRetentionDeletesOldest(t *testing.T) {
	app, dir := newBackupTestApp(t)
	if err := EnforceRetention(app, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.zip")); !os.IsNotExist(err) {
		t.Fatalf("oldest backup still present (stat err %v)", err)
	}
	for _, keep := range []string{"b.zip", "c.zip"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Fatalf("%s was removed: %v", keep, err)
		}
	}
}

func TestEnforceRetentionReturnsDeleteError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory write permission")
	}
	app, dir := newBackupTestApp(t)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	if err := EnforceRetention(app, 1); err == nil {
		t.Fatal("EnforceRetention swallowed failed deletes")
	}
}

func TestEnforceRetentionReturnsListError(t *testing.T) {
	app, dir := newBackupTestApp(t)
	// A regular file where the backups directory should be cannot be listed.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnforceRetention(app, 1); err == nil {
		t.Fatal("EnforceRetention swallowed an unlistable backups path")
	}
}

func TestEnforceRetentionUnderLimitOrUnlimited(t *testing.T) {
	app, dir := newBackupTestApp(t)
	for _, limit := range []int{0, 3, 10} {
		if err := EnforceRetention(app, limit); err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("backups = %d, want all 3 kept", len(entries))
	}
}
