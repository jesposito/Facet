package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRewriteNestedMediaURLsUpdatesOnlyURLFields(t *testing.T) {
	oldURL := "/api/files/uploads/old/image.png"
	newURL := "/api/files/uploads/new/image.png"
	input := `{
		"hero":{"intro":"Keep /api/files/uploads/old/image.png in prose","image":{"src":"/api/files/uploads/old/image.png"},"action":{"href":"/api/files/uploads/old/image.png"}},
		"sections":[{"blocks":[
			{"image":{"src":"/api/files/uploads/old/image.png"}},
			{"lessons":[{"src":"/api/files/uploads/old/image.png","poster":"/api/files/uploads/old/image.png","captions":{"src":"/api/files/uploads/old/image.png"}}]},
			{"steps":[{"image":{"src":"/api/files/uploads/old/image.png"}}]}
		]}]
	}`

	rewritten, changed := RewriteNestedMediaURLs(json.RawMessage(input), oldURL, newURL)
	if !changed {
		t.Fatal("nested media references were not rewritten")
	}
	encoded, err := json.Marshal(rewritten)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(encoded), newURL) != 7 {
		t.Fatalf("rewritten URL count = %d, want 7: %s", strings.Count(string(encoded), newURL), encoded)
	}
	if !strings.Contains(string(encoded), `"intro":"Keep `+oldURL+` in prose"`) {
		t.Fatalf("prose was changed: %s", encoded)
	}

	_, changed = RewriteNestedMediaURLs(json.RawMessage(input), "/api/files/uploads/missing/image.png", newURL)
	if changed {
		t.Fatal("unrelated media URL triggered a rewrite")
	}
}

func TestFindMediaUsageIncludesProjectExperienceURLs(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	external := core.NewBaseCollection("external_media")
	external.Fields.Add(&core.TextField{Name: "url"})
	if err := app.Save(external); err != nil {
		t.Fatal(err)
	}
	projects := core.NewBaseCollection("projects")
	projects.Fields.Add(&core.TextField{Name: "title"})
	projects.Fields.Add(&core.JSONField{Name: "experience"})
	if err := app.Save(projects); err != nil {
		t.Fatal(err)
	}

	media := core.NewRecord(external)
	media.Set("url", "/api/files/uploads/upload1/lesson.mp4")
	if err := app.Save(media); err != nil {
		t.Fatal(err)
	}
	project := core.NewRecord(projects)
	project.Set("title", "Nested media project")
	project.Set("experience", json.RawMessage(`{"version":1,"hero":{"image":{"src":"/api/files/uploads/upload1/lesson.mp4"}}}`))
	if err := app.Save(project); err != nil {
		t.Fatal(err)
	}

	usage, err := FindMediaUsage(&pocketbase.PocketBase{App: app}, media.Id)
	if err != nil {
		t.Fatal(err)
	}
	if usage.UsageCount != 1 || len(usage.UsedBy) != 1 || usage.UsedBy[0].RecordID != project.Id {
		t.Fatalf("nested media usage not found: %#v", usage)
	}

	if err := RemoveMediaRefFromRecord(&pocketbase.PocketBase{App: app}, "projects", project.Id, media.Id); err != nil {
		t.Fatalf("remove nested media reference: %v", err)
	}
	updated, err := app.FindRecordById("projects", project.Id)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(updated.Get("experience"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"src":""`) || strings.Contains(string(encoded), media.GetString("url")) {
		t.Fatalf("nested media URL was not cleared: %s", encoded)
	}
}

func TestCleanupKeepsExperienceMediaAndDeletesUnusedUpload(t *testing.T) {
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
	projects := core.NewBaseCollection("projects")
	projects.Fields.Add(&core.JSONField{Name: "experience"})
	if err := app.Save(projects); err != nil {
		t.Fatal(err)
	}
	var kept *core.Record
	for _, name := range []string{"kept.mp4", "unused.mp4"} {
		upload := core.NewRecord(uploads)
		upload.Set("file", []string{name})
		upload.Set("created", time.Now().UTC().Add(-48*time.Hour).Format("2006-01-02 15:04:05.000Z"))
		if err := app.Save(upload); err != nil {
			t.Fatal(err)
		}
		if name == "kept.mp4" {
			kept = upload
		}
	}
	path := "/api/files/" + uploads.Id + "/" + kept.Id + "/kept.mp4"
	project := core.NewRecord(projects)
	project.Set("experience", map[string]any{"hero": map[string]any{"image": map[string]any{"src": "https://portfolio.example" + path + "?thumb=640x0"}}})
	if err := app.Save(project); err != nil {
		t.Fatal(err)
	}
	if removed := cleanupOrphanedUploads(&pocketbase.PocketBase{App: app}); removed != 1 {
		t.Fatalf("removed %d uploads, want only the unused upload", removed)
	}
	if _, err := app.FindRecordById("uploads", kept.Id); err != nil {
		t.Fatalf("used media deleted: %v", err)
	}
}

func TestRewriteExperienceMediaMatchesAbsoluteURLsAndKeepsThumbnail(t *testing.T) {
	value := map[string]any{"image": map[string]any{"src": "https://portfolio.example/api/files/uploads/old/a.png?thumb=320x0"}}
	rewritten, changed := RewriteNestedMediaURLs(value, "/api/files/uploads/old/a.png", "/api/files/uploads/new/b.png")
	if !changed {
		t.Fatal("absolute URL not matched")
	}
	encoded, _ := json.Marshal(rewritten)
	if !strings.Contains(string(encoded), "/api/files/uploads/new/b.png?thumb=320x0") {
		t.Fatalf("thumbnail query lost: %s", encoded)
	}
}
