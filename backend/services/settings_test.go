package services

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestHomepageSectionChoicesSurviveTypedSaveAndReload(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := core.NewBaseCollection("site_settings")
	collection.Fields.Add(
		&core.BoolField{Name: "homepage_enabled"},
		&core.TextField{Name: "landing_page_message"},
		&core.JSONField{Name: "homepage_sections"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}

	// Match the typed API request before passing it through the actual persistence path.
	const input = `{"testimonials":{"enabled":true,"items":["shown"],"layout":"featured","featuredId":"shown"},"skills":{"enabled":true,"items":["skill"],"categoryOrder":["Languages","Tools"],"disabledCategories":["Tools"],"categoryDisplayModes":{"Languages":"tags"}}}`
	var sections map[string]HomepageSectionConfig
	if err := json.Unmarshal([]byte(input), &sections); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateSiteSettings(app, map[string]any{"homepage_sections": sections}, app.Logger()); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadSiteSettings(app)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(settings.HomepageSections)
	if err != nil {
		t.Fatal(err)
	}
	var want, got map[string]any
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("homepage choices changed after save and reload: got %s", encoded)
	}

	// Omitting a section update must also keep the saved choices intact.
	if _, err := UpdateSiteSettings(app, map[string]any{"homepage_enabled": false}, app.Logger()); err != nil {
		t.Fatal(err)
	}
	settings, err = LoadSiteSettings(app)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(settings.HomepageSections, sections) {
		t.Fatal("an unrelated site setting changed homepage choices")
	}
}
