package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newLibraryRefsTestApp(t *testing.T) (*tests.TestApp, *core.Collection) {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	uploads := core.NewBaseCollection("uploads")
	uploads.Fields.Add(&core.JSONField{Name: "file"}, &core.DateField{Name: "created"})
	if err := app.Save(uploads); err != nil {
		t.Fatal(err)
	}
	courses := core.NewBaseCollection("courses")
	courses.Fields.Add(&core.TextField{Name: "title"}, &core.TextField{Name: "cover_image_library_url"})
	if err := app.Save(courses); err != nil {
		t.Fatal(err)
	}
	return app, uploads
}

func saveTestUpload(t *testing.T, app core.App, uploads *core.Collection, name string, age time.Duration) (*core.Record, string) {
	t.Helper()
	upload := core.NewRecord(uploads)
	upload.Set("file", []string{name})
	upload.Set("created", time.Now().UTC().Add(-age).Format("2006-01-02 15:04:05.000Z"))
	if err := app.Save(upload); err != nil {
		t.Fatal(err)
	}
	return upload, "/api/files/" + uploads.Id + "/" + upload.Id + "/" + name
}

func saveTestCourse(t *testing.T, app core.App, coverURL string) *core.Record {
	t.Helper()
	courses, err := app.FindCollectionByNameOrId("courses")
	if err != nil {
		t.Fatal(err)
	}
	course := core.NewRecord(courses)
	course.Set("title", "Greenhouse basics")
	course.Set("cover_image_library_url", coverURL)
	if err := app.Save(course); err != nil {
		t.Fatal(err)
	}
	return course
}

func TestLibraryURLFieldsIncludesCourseCover(t *testing.T) {
	fields := LibraryURLFields["courses"]
	if len(fields) != 1 || fields[0] != "cover_image_library_url" {
		t.Fatalf("courses library fields = %v, want [cover_image_library_url]", fields)
	}
}

func TestCleanupKeepsCourseLibraryCover(t *testing.T) {
	app, uploads := newLibraryRefsTestApp(t)

	cover, coverURL := saveTestUpload(t, app, uploads, "cover.png", 48*time.Hour)
	orphan, _ := saveTestUpload(t, app, uploads, "orphan.png", 48*time.Hour)
	saveTestCourse(t, app, coverURL)

	if removed, err := cleanupOrphanedUploads(&pocketbase.PocketBase{App: app}); err != nil || removed != 1 {
		t.Fatalf("removed %d uploads (err %v), want only the orphan", removed, err)
	}
	if _, err := app.FindRecordById("uploads", cover.Id); err != nil {
		t.Fatalf("course cover upload deleted: %v", err)
	}
	if _, err := app.FindRecordById("uploads", orphan.Id); err == nil {
		t.Fatal("orphaned upload survived cleanup")
	}
}

func TestFindMediaUsageReportsCourseLibraryCover(t *testing.T) {
	app, uploads := newLibraryRefsTestApp(t)
	external := core.NewBaseCollection("external_media")
	external.Fields.Add(&core.TextField{Name: "url"})
	if err := app.Save(external); err != nil {
		t.Fatal(err)
	}

	_, coverURL := saveTestUpload(t, app, uploads, "cover.png", time.Hour)
	course := saveTestCourse(t, app, coverURL)
	mirror := core.NewRecord(external)
	mirror.Set("url", coverURL)
	if err := app.Save(mirror); err != nil {
		t.Fatal(err)
	}

	usage, err := FindMediaUsage(&pocketbase.PocketBase{App: app}, mirror.Id)
	if err != nil {
		t.Fatal(err)
	}
	if usage.UsageCount != 1 || len(usage.UsedBy) != 1 || usage.UsedBy[0].RecordID != course.Id {
		t.Fatalf("course cover usage not reported: %#v", usage)
	}
}

// Offset paging skipped records once earlier pages lost rows to deletion. The
// cursor must reach every old orphan in a single run while keeping referenced
// and fresh uploads.
func TestCleanupDeletesEveryOldOrphanInOneRun(t *testing.T) {
	app, uploads := newLibraryRefsTestApp(t)

	const orphanCount = 120
	for i := 0; i < orphanCount; i++ {
		saveTestUpload(t, app, uploads, fmt.Sprintf("orphan-%03d.png", i), 48*time.Hour)
	}
	used, usedURL := saveTestUpload(t, app, uploads, "used.png", 48*time.Hour)
	saveTestCourse(t, app, usedURL)
	fresh, _ := saveTestUpload(t, app, uploads, "fresh.png", time.Hour)

	if removed, err := cleanupOrphanedUploads(&pocketbase.PocketBase{App: app}); err != nil || removed != orphanCount {
		t.Fatalf("removed %d uploads (err %v), want %d", removed, err, orphanCount)
	}
	remaining, err := app.FindAllRecords("uploads")
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Fatalf("%d uploads remain, want the referenced and fresh uploads", len(remaining))
	}
	for _, id := range []string{used.Id, fresh.Id} {
		if _, err := app.FindRecordById("uploads", id); err != nil {
			t.Fatalf("upload %s should survive cleanup: %v", id, err)
		}
	}
}

// A reference lookup that fails must keep the upload and report no deletions,
// instead of treating the unreadable collection as "no references".
func TestCleanupKeepsUploadsWhenReferenceLookupFails(t *testing.T) {
	app, uploads := newLibraryRefsTestApp(t)
	orphan, _ := saveTestUpload(t, app, uploads, "orphan.png", 48*time.Hour)

	// The collection still exists in the schema, but its table is gone, so every
	// courses query now fails.
	if _, err := app.DB().NewQuery("DROP TABLE courses").Execute(); err != nil {
		t.Fatal(err)
	}

	deleted, err := cleanupOrphanedUploads(&pocketbase.PocketBase{App: app})
	if err == nil {
		t.Fatal("expected the failed courses lookup to be reported")
	}
	if deleted != 0 {
		t.Fatalf("deleted %d uploads despite a failed lookup", deleted)
	}
	if _, err := app.FindRecordById("uploads", orphan.Id); err != nil {
		t.Fatalf("upload deleted despite a failed lookup: %v", err)
	}
}
