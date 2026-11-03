package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const repairMigrationFile = "1779800600_repair_comments_and_log_timestamps.go"

func TestFreshMigrationsCreateCommentAndTimestampSchema(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	lessons := findCollection(t, app, "lessons")
	if lessons.Fields.GetByName("comments_enabled") == nil {
		t.Fatal("lessons.comments_enabled is missing")
	}

	course := saveRecord(t, app, "courses", map[string]any{
		"title": "Course", "slug": "course", "access_tier": "free", "visibility": "public",
	})
	module := saveRecord(t, app, "modules", map[string]any{
		"course": course.Id, "title": "Module", "sort_order": 1,
	})
	lesson := saveRecord(t, app, "lessons", map[string]any{
		"module": module.Id, "title": "Lesson", "content_type": "text", "sort_order": 1, "comments_enabled": true,
	})
	lesson = findRecord(t, app, "lessons", lesson.Id)
	if !lesson.GetBool("comments_enabled") {
		t.Fatal("lessons.comments_enabled did not round-trip")
	}

	for _, tc := range []struct {
		collection string
		fields     map[string]any
		indexes    []string
	}{
		{"access_logs", map[string]any{"view_id": "view12345678901"}, []string{"idx_access_logs_created", "idx_access_logs_view"}},
		{"system_alerts", map[string]any{"severity": "warning", "event": "test_event", "title": "Test alert"}, []string{"idx_system_alerts_created", "idx_system_alerts_ack"}},
	} {
		collection := findCollection(t, app, tc.collection)
		created := collection.Fields.GetByName("created")
		if created == nil || created.Type() != core.FieldTypeAutodate {
			t.Fatalf("%s.created is missing or has the wrong type", tc.collection)
		}
		for _, index := range tc.indexes {
			assertIndexExists(t, app, index)
		}
		record := saveRecord(t, app, tc.collection, tc.fields)
		if record.GetString("created") == "" {
			t.Fatalf("new %s record has no created timestamp", tc.collection)
		}
	}
}

func TestLegacyLibraryURLConversionPreservesValues(t *testing.T) {
	app := newIsolatedMigrationApp(t)
	fieldsToFix := map[string]string{
		"experience": "company_logo_library_url",
		"education":  "institution_logo_library_url",
		"projects":   "cover_image_library_url",
		"posts":      "cover_image_library_url",
		"talks":      "cover_image_library_url",
	}

	for collectionName, fieldName := range fieldsToFix {
		collection := core.NewBaseCollection(collectionName)
		collection.Fields.Add(&core.TextField{Name: "title"})
		collection.Fields.Add(&core.URLField{Name: fieldName})
		if err := app.Save(collection); err != nil {
			t.Fatalf("create %s: %v", collectionName, err)
		}
		value := "https://cdn.example.com/" + collectionName + ".png"
		record := saveRecord(t, app, collectionName, map[string]any{"title": "Kept title", fieldName: value})
		empty := saveRecord(t, app, collectionName, map[string]any{"title": "No image"})
		titleID := collection.Fields.GetByName("title").GetId()

		runMigration(t, app, "1738200002_fix_library_url_field_types.go")

		updated := findRecord(t, app, collectionName, record.Id)
		if got := updated.GetString(fieldName); got != value {
			t.Errorf("%s.%s = %q, want %q", collectionName, fieldName, got, value)
		}
		if got := updated.GetString("title"); got != "Kept title" {
			t.Errorf("%s.title = %q, want unchanged", collectionName, got)
		}
		if got := findRecord(t, app, collectionName, empty.Id).GetString(fieldName); got != "" {
			t.Errorf("%s.%s on empty record = %q, want empty", collectionName, fieldName, got)
		}
		updatedCollection := findCollection(t, app, collectionName)
		field := updatedCollection.Fields.GetByName(fieldName)
		if field == nil || field.Type() != core.FieldTypeText {
			t.Errorf("%s.%s was not converted to text", collectionName, fieldName)
		}
		if title := updatedCollection.Fields.GetByName("title"); title == nil || title.GetId() != titleID {
			t.Errorf("%s.title field was replaced", collectionName)
		}

		relativePath := "/api/files/library/abc123/" + collectionName + ".png"
		updated.Set(fieldName, relativePath)
		if err := app.Save(updated); err != nil {
			t.Fatalf("save relative path in %s.%s: %v", collectionName, fieldName, err)
		}
		if got := findRecord(t, app, collectionName, record.Id).GetString(fieldName); got != relativePath {
			t.Errorf("%s.%s = %q, want %q", collectionName, fieldName, got, relativePath)
		}

		runMigration(t, app, "1738200002_fix_library_url_field_types.go")
		if got := findRecord(t, app, collectionName, record.Id).GetString(fieldName); got != relativePath {
			t.Errorf("%s.%s changed on migration rerun: %q", collectionName, fieldName, got)
		}
	}
}

func TestLibraryURLConversionLeavesTextFieldsAlone(t *testing.T) {
	app := newIsolatedMigrationApp(t)

	projects := core.NewBaseCollection("projects")
	projects.Fields.Add(&core.TextField{Name: "cover_image_library_url"})
	if err := app.Save(projects); err != nil {
		t.Fatal(err)
	}
	value := "/api/files/library/abc123/cover.png"
	record := saveRecord(t, app, "projects", map[string]any{"cover_image_library_url": value})
	fieldID := projects.Fields.GetByName("cover_image_library_url").GetId()

	runMigration(t, app, "1738200002_fix_library_url_field_types.go")

	field := findCollection(t, app, "projects").Fields.GetByName("cover_image_library_url")
	if field == nil || field.Type() != core.FieldTypeText || field.GetId() != fieldID {
		t.Fatal("existing text field was changed")
	}
	if got := findRecord(t, app, "projects", record.Id).GetString("cover_image_library_url"); got != value {
		t.Fatalf("cover_image_library_url = %q, want %q", got, value)
	}
}

func TestRepairMigrationIsIdempotentAndPreservesUnknownCreated(t *testing.T) {
	app := newIsolatedMigrationApp(t)

	lessons := core.NewBaseCollection("lessons")
	lessons.Fields.Add(&core.TextField{Name: "title"})
	if err := app.Save(lessons); err != nil {
		t.Fatal(err)
	}
	lesson := saveRecord(t, app, "lessons", map[string]any{"title": "Existing lesson"})

	accessLogs := core.NewBaseCollection("access_logs")
	accessLogs.Fields.Add(&core.TextField{Name: "view_id"})
	if err := app.Save(accessLogs); err != nil {
		t.Fatal(err)
	}
	accessLog := saveRecord(t, app, "access_logs", map[string]any{"view_id": "existing-view"})

	alerts := core.NewBaseCollection("system_alerts")
	alerts.Fields.Add(&core.TextField{Name: "severity"})
	// acknowledged_at shipped with the original system_alerts collection.
	alerts.Fields.Add(&core.DateField{Name: "acknowledged_at"})
	if err := app.Save(alerts); err != nil {
		t.Fatal(err)
	}
	alert := saveRecord(t, app, "system_alerts", map[string]any{"severity": "warning"})

	runMigration(t, app, repairMigrationFile)

	lesson = findRecord(t, app, "lessons", lesson.Id)
	lesson.Set("comments_enabled", true)
	if err := app.Save(lesson); err != nil {
		t.Fatalf("save comments_enabled: %v", err)
	}
	if !findRecord(t, app, "lessons", lesson.Id).GetBool("comments_enabled") {
		t.Fatal("repaired comments_enabled did not round-trip")
	}

	for _, tc := range []struct {
		collection string
		recordID   string
		fields     map[string]any
		indexes    []string
	}{
		{"access_logs", accessLog.Id, map[string]any{"view_id": "new-view"}, []string{"idx_access_logs_created", "idx_access_logs_view"}},
		{"system_alerts", alert.Id, map[string]any{"severity": "info"}, []string{"idx_system_alerts_created", "idx_system_alerts_ack"}},
	} {
		assertRawCreated(t, app, tc.collection, tc.recordID, "")
		for _, index := range tc.indexes {
			assertIndexExists(t, app, index)
		}
		createdRecord := saveRecord(t, app, tc.collection, tc.fields)
		if createdRecord.GetString("created") == "" {
			t.Fatalf("new %s record has no created timestamp after repair", tc.collection)
		}
	}

	runMigration(t, app, repairMigrationFile)
	if !findRecord(t, app, "lessons", lesson.Id).GetBool("comments_enabled") {
		t.Fatal("comments_enabled changed on repair rerun")
	}
	assertRawCreated(t, app, "access_logs", accessLog.Id, "")
	assertRawCreated(t, app, "system_alerts", alert.Id, "")
}

func newIsolatedMigrationApp(t *testing.T) *tests.TestApp {
	t.Helper()
	registered := core.AppMigrations
	core.AppMigrations = core.MigrationsList{}
	app, err := tests.NewTestApp(t.TempDir())
	core.AppMigrations = registered
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

func runMigration(t *testing.T, app core.App, filename string) {
	t.Helper()
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == filename {
			if err := migration.Up(app); err != nil {
				t.Fatalf("run %s: %v", filename, err)
			}
			return
		}
	}
	t.Fatalf("migration %s is not registered", filename)
}

func findCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}
	return collection
}

func saveRecord(t *testing.T, app core.App, collectionName string, fields map[string]any) *core.Record {
	t.Helper()
	record := core.NewRecord(findCollection(t, app, collectionName))
	for name, value := range fields {
		record.Set(name, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s record: %v", collectionName, err)
	}
	return record
}

func findRecord(t *testing.T, app core.App, collectionName, id string) *core.Record {
	t.Helper()
	record, err := app.FindRecordById(collectionName, id)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func assertIndexExists(t *testing.T, app core.App, name string) {
	t.Helper()
	var count int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = {:name}").
		Bind(map[string]any{"name": name}).Row(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("index %s does not exist", name)
	}
}

func assertRawCreated(t *testing.T, app core.App, collection, id, want string) {
	t.Helper()
	var got string
	if err := app.DB().NewQuery("SELECT created FROM {{" + collection + "}} WHERE id = {:id}").
		Bind(map[string]any{"id": id}).Row(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s %s created = %q, want %q", collection, id, got, want)
	}
}
