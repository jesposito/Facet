package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestReplaceLibraryURLReferencesRepointsCourseCover(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	courses := core.NewBaseCollection("courses")
	courses.Fields.Add(
		&core.TextField{Name: "title"},
		&core.FileField{Name: "cover_image", MaxSelect: 1, MaxSize: 1 << 20},
		&core.TextField{Name: "cover_image_library_url"},
	)
	if err := app.Save(courses); err != nil {
		t.Fatal(err)
	}

	oldURL := "/api/files/uploads/upload1/cover.png"
	newURL := "/api/files/uploads/upload2/replacement.png"
	course := core.NewRecord(courses)
	course.Set("title", "Puppetry for beginners")
	course.Set("cover_image_library_url", oldURL)
	if err := app.Save(course); err != nil {
		t.Fatal(err)
	}

	pb := &pocketbase.PocketBase{App: app}
	if usage := findLibraryURLUsage(pb, oldURL); usage.UsageCount != 1 {
		t.Fatalf("course cover usage = %d, want 1", usage.UsageCount)
	}
	if updated := replaceLibraryURLReferences(pb, oldURL, newURL); updated != 1 {
		t.Fatalf("updated records = %d, want 1", updated)
	}
	course, err = app.FindRecordById("courses", course.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := course.GetString("cover_image_library_url"); got != newURL {
		t.Fatalf("course cover URL = %q, want %q", got, newURL)
	}
}
