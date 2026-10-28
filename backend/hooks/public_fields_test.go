package hooks

import (
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

func publicTestRecord(collection string, fields ...string) *core.Record {
	c := core.NewBaseCollection(collection)
	for _, field := range fields {
		c.Fields.Add(&core.TextField{Name: field})
	}
	r := core.NewRecord(c)
	r.Id = "publicfixture01"
	for _, field := range fields {
		r.Set(field, "value-"+field)
	}
	return r
}

func TestPublicProjectionKeepsPresentationAndDropsPrivateFields(t *testing.T) {
	for _, collection := range []string{"projects", "demo_projects"} {
		r := publicTestRecord(collection, "title", "description", "admin_tags", "source_id", "field_locks", "password_hash", "view_visibility", "future_private_field")
		for _, item := range []map[string]interface{}{serializeRecords([]*core.Record{r})[0], serializeRecordsWithOverrides(nil, []*core.Record{r}, nil, "projects")[0]} {
			if len(item) != 3 || item["id"] != r.Id || item["title"] != "value-title" || item["description"] != "value-description" {
				t.Fatalf("unexpected public projection: %#v", item)
			}
		}
	}
	r := publicTestRecord("testimonials", "content", "author_name", "verification_identifier", "verification_data", "rejection_reason", "request_id")
	item := serializeRecords([]*core.Record{r})[0]
	if len(item) != 3 || item["content"] != "value-content" {
		t.Fatalf("testimonial private fields leaked: %#v", item)
	}
	r = publicTestRecord("unknown_collection", "title", "content")
	if item := serializeRecords([]*core.Record{r})[0]; len(item) != 1 {
		t.Fatalf("unknown collection must fail closed: %#v", item)
	}
}

func TestHiddenFieldsCannotReturnThroughOverridesOrDerivedURLs(t *testing.T) {
	r := publicTestRecord("projects", "title", "description", "cover_image_library_url", "cover_image")
	for _, name := range []string{"description", "cover_image_library_url", "cover_image"} {
		r.Collection().Fields.GetByName(name).SetHidden(true)
	}
	overrides := map[string]map[string]interface{}{r.Id: {"overrides": map[string]interface{}{"title": "Facet title", "description": "hidden override", "admin_tags": "private"}}}
	item := serializeRecordsWithOverrides(nil, []*core.Record{r}, overrides, "projects")[0]
	if len(item) != 2 || item["title"] != "Facet title" {
		t.Fatalf("hidden or override field leaked: %#v", item)
	}
	for _, section := range []string{"experience", "education", "testimonials", "custom:example"} {
		collection := getCollectionName(section)
		r = publicTestRecord(collection, "company_logo", "company_logo_library_url", "institution_logo", "institution_logo_library_url", "author_photo", "media", "media_refs", "cover_image", "cover_image_library_url")
		for _, field := range r.Collection().Fields {
			if field.GetName() != "id" {
				field.SetHidden(true)
			}
		}
		item = serializeRecordsWithOverrides(nil, []*core.Record{r}, nil, section)[0]
		if len(item) != 1 {
			t.Fatalf("hidden media leaked for %s: %#v", section, item)
		}
	}
}

func TestPublicFieldsRetainMediaAndFacetOverrides(t *testing.T) {
	r := publicTestRecord("projects", "title", "summary", "cover_image_library_url", "related_experience", "related_education")
	item := serializeRecordsWithOverrides(nil, []*core.Record{r}, map[string]map[string]interface{}{r.Id: {"overrides": map[string]interface{}{"summary": "Facet summary"}}}, "projects")[0]
	if item["summary"] != "Facet summary" || item["cover_image_url"] != "value-cover_image_library_url" || item["related_experience"] != "value-related_experience" {
		t.Fatalf("public content lost: %#v", item)
	}
}

func TestContactRestrictionsSurvivePublicProjection(t *testing.T) {
	for _, tc := range []struct {
		raw  any
		view string
		want bool
	}{
		{nil, "", true}, {map[string]bool{}, "other", true},
		{map[string]bool{"selected": true}, "selected", true},
		{map[string]bool{"selected": true}, "", false},
		{map[string]bool{"selected": true}, "other", false},
		{map[string]bool{"selected": false}, "selected", false},
		{map[string]any{"selected": "true"}, "selected", false},
	} {
		record := createMockRecord(map[string]any{"visibility": "public", "view_visibility": tc.raw})
		if got := contactVisibleInView(record, tc.view); got != tc.want {
			t.Errorf("%v on %q = %v", tc.raw, tc.view, got)
		}
		if got := isRecordVisibleForSection(record, "contacts", tc.view); got != tc.want {
			t.Errorf("contact section: %v on %q = %v", tc.raw, tc.view, got)
		}
	}
}
