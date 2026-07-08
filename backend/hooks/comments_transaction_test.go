package hooks

import (
	"errors"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newCommentsTransactionTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "facet_comments_tx_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	app, err := tests.NewTestApp(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	comments := core.NewBaseCollection("comments")
	comments.Fields.Add(&core.TextField{Name: "parent_id"})
	if err := app.Save(comments); err != nil {
		t.Fatalf("create comments collection: %v", err)
	}

	reports := core.NewBaseCollection("comment_reports")
	reports.Fields.Add(&core.TextField{Name: "comment"})
	if err := app.Save(reports); err != nil {
		t.Fatalf("create comment_reports collection: %v", err)
	}

	return app
}

func seedTestRecord(t *testing.T, app core.App, collection string, values map[string]any) string {
	t.Helper()

	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find collection %s: %v", collection, err)
	}
	record := core.NewRecord(col)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return record.Id
}

func requireRecordExists(t *testing.T, app core.App, collection, id string) {
	t.Helper()
	if _, err := app.FindRecordById(collection, id); err != nil {
		t.Fatalf("%s/%s should still exist after rollback: %v", collection, id, err)
	}
}

func TestDeleteCommentCascadeRollsBackReplyDeletesWhenParentDeleteFails(t *testing.T) {
	app := newCommentsTransactionTestApp(t)

	parentID := seedTestRecord(t, app, "comments", nil)
	replyID := seedTestRecord(t, app, "comments", map[string]any{"parent_id": parentID})
	parentReportID := seedTestRecord(t, app, "comment_reports", map[string]any{"comment": parentID})
	replyReportID := seedTestRecord(t, app, "comment_reports", map[string]any{"comment": replyID})

	app.OnRecordDelete("comments").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == parentID {
			return errors.New("forced parent delete failure")
		}
		return e.Next()
	})

	err := app.RunInTransaction(func(txApp core.App) error {
		return deleteCommentCascade(txApp, parentID)
	})
	if err == nil {
		t.Fatal("deleteCommentCascade should fail when parent delete hook fails")
	}

	requireRecordExists(t, app, "comments", parentID)
	requireRecordExists(t, app, "comments", replyID)
	requireRecordExists(t, app, "comment_reports", parentReportID)
	requireRecordExists(t, app, "comment_reports", replyReportID)
}
