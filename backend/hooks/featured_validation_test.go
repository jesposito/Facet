package hooks

import (
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newFeaturedTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	dir, err := os.MkdirTemp("", "facet_featured_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	app, err := tests.NewTestApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	posts := core.NewBaseCollection("posts")
	posts.Fields.Add(&core.BoolField{Name: "featured"})
	posts.Fields.Add(&core.BoolField{Name: "is_draft"})
	if err := app.Save(posts); err != nil {
		t.Fatalf("create posts collection: %v", err)
	}

	projects := core.NewBaseCollection("projects")
	projects.Fields.Add(&core.BoolField{Name: "is_featured"})
	projects.Fields.Add(&core.BoolField{Name: "is_draft"})
	if err := app.Save(projects); err != nil {
		t.Fatalf("create projects collection: %v", err)
	}

	testimonials := core.NewBaseCollection("testimonials")
	testimonials.Fields.Add(&core.BoolField{Name: "featured"})
	if err := app.Save(testimonials); err != nil {
		t.Fatalf("create testimonials collection: %v", err)
	}

	return app
}

func saveFeaturedRecord(t *testing.T, app core.App, collection string, values map[string]any) string {
	t.Helper()

	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s collection: %v", collection, err)
	}
	record := core.NewRecord(col)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s record: %v", collection, err)
	}
	return record.Id
}

func TestEnforceFeaturedCapAllowsThreeAndRejectsFourth(t *testing.T) {
	app := newFeaturedTestApp(t)
	cfg := featuredConfig{collection: "projects", field: "is_featured", draftField: "is_draft"}

	saveFeaturedRecord(t, app, "projects", map[string]any{"is_featured": true, "is_draft": false})
	saveFeaturedRecord(t, app, "projects", map[string]any{"is_featured": true, "is_draft": false})

	if err := enforceFeaturedCap(app, cfg, ""); err != nil {
		t.Fatalf("third featured project should pass: %v", err)
	}

	saveFeaturedRecord(t, app, "projects", map[string]any{"is_featured": true, "is_draft": false})
	if err := enforceFeaturedCap(app, cfg, ""); err == nil {
		t.Fatal("fourth featured project should fail")
	}
}

func TestEnforceFeaturedCapIgnoresDraftsAndCurrentRecord(t *testing.T) {
	app := newFeaturedTestApp(t)
	cfg := featuredConfig{collection: "posts", field: "featured", draftField: "is_draft"}

	existingID := saveFeaturedRecord(t, app, "posts", map[string]any{"featured": true, "is_draft": false})
	saveFeaturedRecord(t, app, "posts", map[string]any{"featured": true, "is_draft": false})
	saveFeaturedRecord(t, app, "posts", map[string]any{"featured": true, "is_draft": true})

	if err := enforceFeaturedCap(app, cfg, ""); err != nil {
		t.Fatalf("draft featured post should not count against third slot: %v", err)
	}

	saveFeaturedRecord(t, app, "posts", map[string]any{"featured": true, "is_draft": false})
	if err := enforceFeaturedCap(app, cfg, existingID); err != nil {
		t.Fatalf("resaving an existing featured post should not trip the cap: %v", err)
	}
}

func TestEnforceFeaturedCapCoversTestimonials(t *testing.T) {
	app := newFeaturedTestApp(t)
	cfg := featuredConfig{collection: "testimonials", field: "featured"}

	for i := 0; i < maxFeaturedPerKind; i++ {
		saveFeaturedRecord(t, app, "testimonials", map[string]any{"featured": true})
	}

	if err := enforceFeaturedCap(app, cfg, ""); err == nil {
		t.Fatal("fourth featured testimonial should fail")
	}
}
