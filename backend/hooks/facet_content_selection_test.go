package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// publishedIDs returns the record ids an anonymous visitor gets for a section.
func publishedIDs(t *testing.T, h http.Handler, slug, section string) []string {
	t.Helper()
	rec := do(t, h, testRequest{url: "/api/view/" + slug + "/data"})
	assertStatus(t, rec, http.StatusOK)
	var body struct {
		Sections map[string][]struct {
			ID string `json:"id"`
		} `json:"sections"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, item := range body.Sections[section] {
		ids = append(ids, item.ID)
	}
	return ids
}

func setViewSection(t *testing.T, f *securityFixture, view *core.Record, section string) {
	t.Helper()
	view.Set("sections", "["+section+"]")
	if err := f.app.Save(view); err != nil {
		t.Fatal(err)
	}
}

func assertPublished(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(got, append([]string{}, want...)) {
		t.Fatalf("published ids = %v, want %v", got, want)
	}
}

func idList(t *testing.T, ids ...string) string {
	t.Helper()
	raw, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestViewSectionItemSelection(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	override := map[string]bool{view.Id: true}
	project := func(slug, visibility string, order int, fields map[string]any) string {
		base := map[string]any{"title": slug, "slug": slug, "visibility": visibility, "sort_order": order}
		for k, v := range fields {
			base[k] = v
		}
		return f.create(t, "projects", base).Id
	}
	first := project("first", "public", 2, nil)
	second := project("second", "public", 1, nil)
	unlisted := project("unlisted", "unlisted", 3, nil)
	private := project("private", "private", 4, nil)
	opted := project("opted-in", "private", 5, map[string]any{"view_visibility": override})
	draft := project("draft", "public", 6, map[string]any{"is_draft": true, "view_visibility": override})
	h := f.serve(t)

	for _, tc := range []struct {
		name    string
		section string
		want    []string
	}{
		{"missing mode with empty items", `{"section":"projects","enabled":true,"items":[]}`, nil},
		{"missing mode and items", `{"section":"projects","enabled":true}`, nil},
		{"unknown mode", `{"section":"projects","enabled":true,"items":[],"itemSelection":"everything"}`, nil},
		{"non-string mode", `{"section":"projects","enabled":true,"items":[],"itemSelection":true}`, nil},
		{"explicit selected with empty items", `{"section":"projects","enabled":true,"items":[],"itemSelection":"selected"}`, nil},
		{"disabled all_public", `{"section":"projects","enabled":false,"items":[],"itemSelection":"all_public"}`, nil},
		{"all_public is public and published only", `{"section":"projects","enabled":true,"itemSelection":"all_public"}`, []string{first, second}},
		{"all_public ignores kept picks", `{"section":"projects","enabled":true,"items":` + idList(t, private, opted, unlisted) + `,"itemSelection":"all_public"}`, []string{first, second}},
		{"selected keeps order and private opt-in, never drafts", `{"section":"projects","enabled":true,"itemSelection":"selected","items":` + idList(t, second, opted, draft, private, unlisted, first) + `}`, []string{second, opted, unlisted, first}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setViewSection(t, f, view, tc.section)
			assertPublished(t, publishedIDs(t, h, "portfolio", "projects"), tc.want...)
		})
	}
}

func TestAllPublicContactsRespectTheirPageScope(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	other := f.create(t, "views", map[string]any{"name": "Other", "slug": "other", "visibility": "public", "is_active": true})
	contact := func(label string, order int, scope map[string]bool) *core.Record {
		fields := map[string]any{"type": "email", "value": label + "@example.com", "label": label, "protection_level": "none", "sort_order": order}
		if scope != nil {
			fields["view_visibility"] = scope
		}
		return f.create(t, "contact_methods", fields)
	}
	open := contact("open", 3, nil)
	here := contact("here", 2, map[string]bool{view.Id: true})
	elsewhere := contact("elsewhere", 1, map[string]bool{other.Id: true})
	scopeBefore := elsewhere.GetString("view_visibility")
	allPublic := `{"section":"contacts","enabled":true,"itemSelection":"all_public"}`
	setViewSection(t, f, view, allPublic)
	setViewSection(t, f, other, allPublic)
	h := f.serve(t)

	assertPublished(t, publishedIDs(t, h, "portfolio", "contacts"), open.Id, here.Id)
	assertPublished(t, publishedIDs(t, h, "other", "contacts"), open.Id, elsewhere.Id)

	setViewSection(t, f, view, `{"section":"contacts","enabled":true,"items":[]}`)
	assertPublished(t, publishedIDs(t, h, "portfolio", "contacts"))

	setViewSection(t, f, view, `{"section":"contacts","enabled":true,"items":`+idList(t, elsewhere.Id, here.Id)+`}`)
	assertPublished(t, publishedIDs(t, h, "portfolio", "contacts"), here.Id)

	stored, err := f.app.FindRecordById("contact_methods", elsewhere.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.GetString("view_visibility"); got != scopeBefore {
		t.Fatalf("contact scope changed: %s, want %s", got, scopeBefore)
	}
}

func createTestimonial(t *testing.T, f *securityFixture, author, status, visibility, list string, order int) string {
	t.Helper()
	return f.create(t, "testimonials", map[string]any{
		"content": "Great to work with", "author_name": author, "relationship": "colleague",
		"verification_method": "email", "verification_identifier": author + "@example.com",
		"status": status, "visibility": visibility, "list": list, "sort_order": order,
	}).Id
}

func TestTestimonialSelectionKeepsApprovalAndListRules(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	general := createTestimonial(t, f, "general", "approved", "public", "", 3)
	listed := createTestimonial(t, f, "listed", "approved", "public", "clients", 2)
	pending := createTestimonial(t, f, "pending", "pending", "public", "clients", 5)
	hidden := createTestimonial(t, f, "hidden", "approved", "private", "clients", 4)
	h := f.serve(t)

	for _, tc := range []struct {
		name    string
		section string
		want    []string
	}{
		{"all_public is approved and public", `{"section":"testimonials","enabled":true,"itemSelection":"all_public"}`, []string{general, listed}},
		{"a list narrows all_public", `{"section":"testimonials","enabled":true,"itemSelection":"all_public","list":"clients"}`, []string{listed}},
		{"a list shows without picks in selected mode", `{"section":"testimonials","enabled":true,"items":[],"list":"clients"}`, []string{listed}},
		{"an empty selection without a list stays empty", `{"section":"testimonials","enabled":true,"items":[]}`, nil},
		{"selected never publishes an unapproved pick", `{"section":"testimonials","enabled":true,"items":` + idList(t, pending, hidden, general) + `}`, []string{general}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setViewSection(t, f, view, tc.section)
			assertPublished(t, publishedIDs(t, h, "portfolio", "testimonials"), tc.want...)
		})
	}
}

func TestSectionsPublishPastOneHundredRecords(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	const total = 101
	ids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		slug := fmt.Sprintf("project-%d", i)
		record := f.create(t, "projects", map[string]any{"title": slug, "slug": slug, "visibility": "public", "sort_order": total - i})
		ids = append(ids, record.Id)
		createTestimonial(t, f, fmt.Sprintf("author-%d", i), "approved", "public", "clients", i)
	}
	h := f.serve(t)

	setViewSection(t, f, view, `{"section":"projects","enabled":true,"itemSelection":"all_public"}`)
	assertPublished(t, publishedIDs(t, h, "portfolio", "projects"), ids...)

	reversed := make([]string, 0, total)
	for i := total - 1; i >= 0; i-- {
		reversed = append(reversed, ids[i])
	}
	setViewSection(t, f, view, `{"section":"projects","enabled":true,"items":`+idList(t, reversed...)+`}`)
	assertPublished(t, publishedIDs(t, h, "portfolio", "projects"), reversed...)

	setViewSection(t, f, view, `{"section":"testimonials","enabled":true,"list":"clients","items":[]}`)
	if got := len(publishedIDs(t, h, "portfolio", "testimonials")); got != total {
		t.Fatalf("published %d listed testimonials, want %d", got, total)
	}
}

func TestAllPublicMembershipIsWhatTheViewPublishes(t *testing.T) {
	f := newSecurityFixture(t)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	public := f.create(t, "projects", map[string]any{"title": "Public", "slug": "public", "visibility": "public"})
	pick := f.create(t, "projects", map[string]any{"title": "Pick", "slug": "pick", "visibility": "private"})
	setViewSection(t, f, view, `{"section":"projects","enabled":true,"items":`+idList(t, pick.Id)+`,"itemSelection":"all_public"}`)

	memberships, err := buildViewMemberships(f.pb, "projects")
	if err != nil {
		t.Fatal(err)
	}
	if got := memberships[public.Id]; len(got) != 1 || got[0].Id != view.Id {
		t.Fatalf("public project memberships = %v, want %s", got, view.Id)
	}
	if got := memberships[pick.Id]; len(got) != 0 {
		t.Fatalf("kept pick counted as membership: %v", got)
	}
	stored, err := f.app.FindRecordById("projects", pick.Id)
	if err != nil {
		t.Fatal(err)
	}
	if isVisibleInView(stored, view.Id) {
		t.Fatal("kept pick holds a private grant")
	}
}

func TestResumeGetsTheSameSectionContentAsThePage(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	project := func(slug, visibility string, fields map[string]any) string {
		base := map[string]any{"title": slug, "slug": slug, "visibility": visibility}
		for k, v := range fields {
			base[k] = v
		}
		return f.create(t, "projects", base).Id
	}
	public := project("public", "public", nil)
	project("unlisted", "unlisted", nil)
	private := project("private", "private", nil)
	opted := project("opted-in", "private", map[string]any{"view_visibility": map[string]bool{view.Id: true}})
	listed := createTestimonial(t, f, "listed", "approved", "public", "clients", 1)
	createTestimonial(t, f, "general", "approved", "public", "", 2)
	h := f.serve(t)

	for _, tc := range []struct {
		name        string
		sectionName string
		section     string
		want        []string
	}{
		{"all_public ignores kept picks", "projects", `{"section":"projects","enabled":true,"items":` + idList(t, private) + `,"itemSelection":"all_public"}`, []string{public}},
		{"legacy empty stays empty", "projects", `{"section":"projects","enabled":true,"items":[]}`, nil},
		{"selected keeps private opt-in", "projects", `{"section":"projects","enabled":true,"items":` + idList(t, private, opted) + `}`, []string{opted}},
		{"testimonial list", "testimonials", `{"section":"testimonials","enabled":true,"items":[],"list":"clients"}`, []string{listed}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setViewSection(t, f, view, tc.section)
			data, err := collectViewData(f.pb, view, nil)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, item := range data.Sections[tc.sectionName] {
				got = append(got, item["id"].(string))
			}
			assertPublished(t, got, tc.want...)
			assertPublished(t, got, publishedIDs(t, h, "portfolio", tc.sectionName)...)
		})
	}
}

func TestStaleOverrideCannotOpenAPrivateProject(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	project := f.create(t, "projects", map[string]any{"title": "Private project", "slug": "private-project", "description": "PRIVATE WRITEUP", "visibility": "private", "view_visibility": map[string]bool{view.Id: true}})
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)
	const oldURL = "/api/project/private-project?from=portfolio"
	picked := idList(t, project.Id)
	selected := `{"section":"projects","enabled":true,"items":` + picked + `}`

	for _, tc := range []struct {
		name    string
		section string
		allowed bool
	}{
		{"still selected", selected, true},
		{"deselected", `{"section":"projects","enabled":true,"items":[]}`, false},
		{"section disabled", `{"section":"projects","enabled":false,"items":` + picked + `}`, false},
		{"switched to all_public", `{"section":"projects","enabled":true,"items":` + picked + `,"itemSelection":"all_public"}`, false},
		{"same id under another section", `{"section":"posts","enabled":true,"items":` + picked + `}`, false},
		{"selected again", selected, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setViewSection(t, f, view, tc.section)
			rec := do(t, h, testRequest{url: oldURL})
			if tc.allowed {
				assertStatus(t, rec, http.StatusOK)
				assertContains(t, rec, "PRIVATE WRITEUP")
			} else {
				assertStatus(t, rec, http.StatusNotFound)
			}
			owner := do(t, h, testRequest{url: "/api/project/private-project", headers: map[string]string{"Authorization": adminToken}})
			assertStatus(t, owner, http.StatusOK)
		})
	}

	view.Set("is_active", false)
	if err := f.app.Save(view); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, do(t, h, testRequest{url: oldURL}), http.StatusNotFound)
}
