package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	recoveryStructured = "# Greenhouse notes\n\nTomatoes first.\n\nThen the basil."
	recoveryFlattened  = "# Greenhouse notes Tomatoes first. Then the basil."
)

// newRecoveryFixture builds a temporary app with one post, takes a real backup
// while the post still has its paragraphs, then flattens the live value the way
// #443 did. It returns the app wrapper and the post id.
func newRecoveryFixture(t *testing.T) (*pocketbase.PocketBase, string) {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	posts := core.NewBaseCollection("posts")
	posts.Fields.Add(&core.TextField{Name: "body"})
	if err := app.Save(posts); err != nil {
		t.Fatal(err)
	}
	post := core.NewRecord(posts)
	post.Set("body", recoveryStructured)
	if err := app.Save(post); err != nil {
		t.Fatal(err)
	}

	if err := app.CreateBackup(context.Background(), "facet_auto_seed.zip"); err != nil {
		t.Fatalf("create seed backup: %v", err)
	}

	if _, err := app.DB().NewQuery("UPDATE posts SET body = {:v} WHERE id = {:id}").
		Bind(dbx.Params{"v": recoveryFlattened, "id": post.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	return &pocketbase.PocketBase{App: app}, post.Id
}

func postBody(t *testing.T, app *pocketbase.PocketBase, id string) string {
	t.Helper()
	var body string
	if err := app.DB().NewQuery("SELECT body FROM posts WHERE id = {:id}").
		Bind(dbx.Params{"id": id}).Row(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func backupNames(t *testing.T, app *pocketbase.PocketBase) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(app.DataDir(), core.LocalBackupsDirName))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestContentRecoveryRestoresFromRealBackup(t *testing.T) {
	app, id := newRecoveryFixture(t)

	report, err := RunContentRecovery(app, true)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if report.Restored != 1 || report.BackupUsed == "" {
		t.Fatalf("report = %+v, want one restore and a safety backup", report)
	}
	if got := postBody(t, app, id); got != recoveryStructured {
		t.Fatalf("body = %q, want the structured backup value", got)
	}
}

func TestContentRecoveryAbortsWhenBackupsCannotBeListed(t *testing.T) {
	app, id := newRecoveryFixture(t)

	// Replace the backups directory with a regular file so opening it fails.
	dir := filepath.Join(app.DataDir(), core.LocalBackupsDirName)
	if err := os.Rename(dir, dir+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := RunContentRecovery(app, true)
	if err == nil {
		t.Fatalf("expected an error when backups cannot be listed, got report %+v", report)
	}
	if report.Restored != 0 || report.BackupUsed != "" {
		t.Fatalf("report = %+v, want no restores and no safety backup", report)
	}
	if got := postBody(t, app, id); got != recoveryFlattened {
		t.Fatalf("body changed to %q despite the listing failure", got)
	}
}

func TestContentRecoveryAbortsOnUnreadableArchive(t *testing.T) {
	app, id := newRecoveryFixture(t)

	// A corrupt archive newer than the good one is scanned first.
	bad := filepath.Join(app.DataDir(), core.LocalBackupsDirName, "facet_auto_corrupt.zip")
	if err := os.WriteFile(bad, []byte("this is not a zip archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(bad, future, future); err != nil {
		t.Fatal(err)
	}

	report, err := RunContentRecovery(app, true)
	if err == nil || !strings.Contains(err.Error(), "facet_auto_corrupt.zip") {
		t.Fatalf("err = %v, want a scan error naming the corrupt archive", err)
	}
	if report.Restored != 0 || len(report.Items) != 0 || report.BackupUsed != "" {
		t.Fatalf("report = %+v, want no plan, no restores and no safety backup", report)
	}
	if got := postBody(t, app, id); got != recoveryFlattened {
		t.Fatalf("body changed to %q despite the unreadable archive", got)
	}
}

func TestContentRecoveryFailsClosedWithoutSafetyBackup(t *testing.T) {
	app, id := newRecoveryFixture(t)
	before := backupNames(t, app)

	// PocketBase refuses to start a backup while another is marked active.
	app.Store().Set(core.StoreKeyActiveBackup, "test")
	defer app.Store().Remove(core.StoreKeyActiveBackup)

	report, err := RunContentRecovery(app, true)
	if err == nil || !strings.Contains(err.Error(), "safety backup") {
		t.Fatalf("err = %v, want a safety backup failure", err)
	}
	if report.Restored != 0 || report.Skipped != 0 || report.BackupUsed != "" {
		t.Fatalf("report = %+v, want no mutations", report)
	}
	if got := postBody(t, app, id); got != recoveryFlattened {
		t.Fatalf("body changed to %q without a safety backup", got)
	}
	if after := backupNames(t, app); len(after) != len(before) {
		t.Fatalf("backups changed from %v to %v", before, after)
	}
}
