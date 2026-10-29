package hooks

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const validProjectExperienceJSON = `{
  "version": 1,
  "enabled": true,
  "layout": "immersive",
  "hero": {
    "intro": "A complete project walkthrough.",
    "tone": "aqua",
    "imagePosition": "left",
    "image": {"src": "/api/files/uploads/upload1/hero.png", "alt": "Project overview"},
    "action": {"label": "Read the first section", "href": "#section-1"}
  },
  "sections": [{
    "id": "section-1",
    "title": "Building the project",
    "intro": "Every supported block appears in this section.",
    "tone": "paper",
    "layout": "stack",
    "blocks": [
      {"id": "text-1", "type": "text", "body": "Plain authored text."},
      {"id": "image-1", "type": "image", "image": {"src": "https://cdn.example.com/diagram.png", "alt": "Architecture diagram"}, "caption": "The request path"},
      {"id": "video-1", "type": "video", "title": "Video lesson", "lessons": [{
        "id": "lesson-1",
        "title": "Getting started",
        "src": "/api/files/uploads/upload1/lesson.mp4",
        "description": "A short introduction",
        "poster": "/api/files/uploads/upload1/poster.jpg",
        "captions": {"src": "/api/files/uploads/upload1/captions.vtt", "language": "en", "label": "English"},
        "transcript": "Welcome to the project.",
        "credit": "Facet"
      }]},
      {"id": "code-1", "type": "code", "title": "Implementation", "steps": [{
        "id": "step-1",
        "title": "Create the handler",
        "body": "Add the HTTP handler.",
        "code": "func handler() {}",
        "language": "go",
        "image": {"src": "/api/files/uploads/upload1/code.png", "alt": "Handler code"}
      }]},
      {"id": "disclosure-1", "type": "disclosure", "title": "Tradeoffs", "body": "The reasons behind this approach."},
      {"id": "story-1", "type": "story", "title": "Try it", "template": "Hello {name}[[pause]]!", "fields": [{"key": "name", "label": "Name", "initial": "Ada"}], "pauseMs": 250, "credit": "Example story"}
    ]
  }]
}`

func minimalProjectExperience(url string) string {
	return `{"version":1,"enabled":false,"layout":"immersive","hero":{"intro":"","tone":"ink","imagePosition":"right","image":{"src":` + strconvQuote(url) + `,"alt":""}},"sections":[]}`
}

func strconvQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func TestValidateProjectExperienceAcceptsValidAllBlockDocument(t *testing.T) {
	if err := validateProjectExperience(json.RawMessage(validProjectExperienceJSON)); err != nil {
		t.Fatalf("valid all-block document rejected: %v", err)
	}
	for _, value := range []any{nil, "", "null", json.RawMessage("null")} {
		if err := validateProjectExperience(value); err != nil {
			t.Errorf("nullable experience %#v rejected: %v", value, err)
		}
	}
}

func TestValidateProjectExperienceRejectsMalformedAndUnknownShapes(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"malformed JSON", json.RawMessage(`{"version":1`)},
		{"scalar", json.RawMessage(`"experience"`)},
		{"unknown version", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"version": 1`, `"version": 2`, 1))},
		{"unknown top-level field", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"layout": "immersive"`, `"layout": "immersive", "title": "Duplicate title"`, 1))},
		{"missing hero field", json.RawMessage(`{"version":1,"enabled":true,"layout":"immersive","hero":{"tone":"paper","imagePosition":"left"},"sections":[]}`)},
		{"null optional image", json.RawMessage(`{"version":1,"enabled":false,"layout":"immersive","hero":{"intro":"","tone":"paper","imagePosition":"left","image":null},"sections":[]}`)},
		{"unknown block", json.RawMessage(`{"version":1,"enabled":false,"layout":"immersive","hero":{"intro":"","tone":"paper","imagePosition":"left"},"sections":[{"id":"section","title":"","intro":"","tone":"paper","layout":"stack","blocks":[{"id":"block","type":"iframe","url":"https://example.com"}]}]}`)},
		{"duplicate IDs", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"id": "step-1"`, `"id": "text-1"`, 1))},
		{"invalid ID", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"id": "section-1"`, `"id": "1 section"`, 1))},
		{"unknown story field", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"pauseMs": 250`, `"pauseMs": 250, "extra": "ignored"`, 1))},
		{"pause above limit", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"pauseMs": 250`, `"pauseMs": 5001`, 1))},
		{"title above limit", json.RawMessage(strings.Replace(validProjectExperienceJSON, `"title": "Building the project"`, `"title": "`+strings.Repeat("x", 201)+`"`, 1))},
		{"document above size limit", json.RawMessage(`{"version":1,"enabled":false,"layout":"immersive","hero":{"intro":"` + strings.Repeat("x", 500001) + `","tone":"paper","imagePosition":"left"},"sections":[]}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateProjectExperience(tt.value); err == nil {
				t.Fatal("invalid experience was accepted")
			}
		})
	}
}

func TestValidateProjectExperienceRejectsDuplicateStoryFieldKeys(t *testing.T) {
	doc := strings.Replace(validProjectExperienceJSON, `"pauseMs": 250`, `"pauseMs": 250`, 1)
	doc = strings.Replace(doc, `{"key": "name", "label": "Name", "initial": "Ada"}`, `{"key":"name","label":"First","initial":""},{"key":"name","label":"Second","initial":""}`, 1)
	if err := validateProjectExperience(json.RawMessage(doc)); err == nil {
		t.Fatal("duplicate story field keys were accepted")
	}
}

func TestValidateProjectExperienceURLRules(t *testing.T) {
	for _, url := range []string{
		"javascript:alert(1)",
		"http://example.com/image.png",
		"//example.com/image.png",
		"https://user:pass@example.com/image.png",
		`https://example.com\image.png`,
		"https://example.com/image.png\n",
		"#section-1",
	} {
		t.Run("reject "+url, func(t *testing.T) {
			if err := validateProjectExperience(json.RawMessage(minimalProjectExperience(url))); err == nil {
				t.Fatalf("unsafe image URL %q was accepted", url)
			}
		})
	}
	for _, url := range []string{"", "/api/files/uploads/id/file.png", "https://example.com/image.png"} {
		t.Run("allow "+url, func(t *testing.T) {
			if err := validateProjectExperience(json.RawMessage(minimalProjectExperience(url))); err != nil {
				t.Fatalf("safe image URL %q rejected: %v", url, err)
			}
		})
	}
}

func newProjectExperienceTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	for _, name := range []string{"projects", "demo_projects"} {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(&core.TextField{Name: "title"})
		collection.Fields.Add(&core.JSONField{Name: "experience", MaxSize: 500000})
		if err := app.Save(collection); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	RegisterProjectExperienceHooks(&pocketbase.PocketBase{App: app})
	return app
}

func TestProjectExperienceHooksProtectCreateAndUpdate(t *testing.T) {
	app := newProjectExperienceTestApp(t)
	for _, collectionName := range []string{"projects", "demo_projects"} {
		t.Run(collectionName, func(t *testing.T) {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				t.Fatal(err)
			}
			record := core.NewRecord(collection)
			record.Set("title", "Project")
			record.Set("experience", json.RawMessage(validProjectExperienceJSON))
			if err := app.Save(record); err != nil {
				t.Fatalf("valid create rejected: %v", err)
			}

			record.Set("experience", json.RawMessage(minimalProjectExperience("javascript:alert(1)")))
			if err := app.Save(record); err == nil {
				t.Fatal("unsafe update was accepted")
			}

			invalid := core.NewRecord(collection)
			invalid.Set("experience", json.RawMessage(`{"version":2}`))
			if err := app.Save(invalid); err == nil {
				t.Fatal("unknown-version create was accepted")
			}
		})
	}
}

func TestProjectExperiencePublicProjectionAndExport(t *testing.T) {
	for _, collectionName := range []string{"projects", "demo_projects"} {
		collection := core.NewBaseCollection(collectionName)
		collection.Fields.Add(&core.JSONField{Name: "experience"})
		collection.Fields.Add(&core.TextField{Name: "admin_tags"})
		record := core.NewRecord(collection)
		record.Id = "project1234567"
		record.Set("experience", json.RawMessage(validProjectExperienceJSON))
		record.Set("admin_tags", "private")

		for _, projected := range []map[string]any{
			serializeRecords([]*core.Record{record})[0],
			serializeRecordsWithOverrides(nil, []*core.Record{record}, nil, "projects")[0],
			sanitizeRecord(record),
		} {
			if _, ok := projected["experience"]; !ok {
				t.Fatalf("%s projection omitted experience: %#v", collectionName, projected)
			}
		}
		if _, ok := serializeRecords([]*core.Record{record})[0]["admin_tags"]; ok {
			t.Fatal("private project field leaked")
		}
	}
}

func TestProjectExperienceAPIWriteField(t *testing.T) {
	collection := core.NewBaseCollection("projects")
	collection.Fields.Add(&core.JSONField{Name: "experience"})
	record := core.NewRecord(collection)
	applyAllowedFields(record, map[string]any{"experience": map[string]any{"version": 1}}, apiV1WriteFields["projects"])
	if record.Get("experience") == nil {
		t.Fatal("public project writes ignore experience")
	}
}

func TestReplaceLibraryURLReferencesUpdatesProjectExperience(t *testing.T) {
	app := newProjectExperienceTestApp(t)
	collection, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("title", "Project")
	record.Set("experience", json.RawMessage(validProjectExperienceJSON))
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	oldURL := "/api/files/uploads/upload1/hero.png"
	newURL := "/api/files/uploads/upload2/replacement.png"
	if updated := replaceLibraryURLReferences(&pocketbase.PocketBase{App: app}, oldURL, newURL); updated != 1 {
		t.Fatalf("updated records = %d, want 1", updated)
	}
	record, err = app.FindRecordById("projects", record.Id)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(record.Get("experience"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), newURL) || strings.Contains(string(encoded), oldURL) {
		t.Fatalf("nested media URL was not replaced: %s", encoded)
	}
}
