package hooks

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type privateDetail struct {
	section string
	url     string
	body    string
	record  *core.Record
}

// createPrivateDetails makes a private post and talk holding an override for
// each view, so whether one opens depends on the other rules.
func createPrivateDetails(t *testing.T, f *securityFixture, views ...*core.Record) []privateDetail {
	t.Helper()
	overrides := map[string]bool{}
	for _, view := range views {
		overrides[view.Id] = true
	}
	post := f.create(t, "posts", map[string]any{"title": "Private post", "slug": "private-post", "content": "PRIVATE POST BODY", "visibility": "private", "view_visibility": overrides})
	talk := f.create(t, "talks", map[string]any{"title": "Private talk", "slug": "private-talk", "event": "Conf", "date": "2026-02-01 00:00:00.000Z", "description": "PRIVATE TALK BODY", "visibility": "private", "view_visibility": overrides})
	return []privateDetail{
		{"posts", "/api/post/private-post", "PRIVATE POST BODY", post},
		{"talks", "/api/talk/private-talk", "PRIVATE TALK BODY", talk},
	}
}

func (d privateDetail) sectionConfig(t *testing.T, enabled string, extra string) string {
	t.Helper()
	return `{"section":"` + d.section + `","enabled":` + enabled + `,"items":` + idList(t, d.record.Id) + extra + `}`
}

// assertOpens checks a request reads the body, or is refused in the same words
// a slug that doesn't exist gets, so a refusal tells a visitor nothing.
func (d privateDetail) assertOpens(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder, allowed bool) {
	t.Helper()
	if allowed {
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, d.body)
		return
	}
	assertStatus(t, rec, http.StatusNotFound)
	missing := do(t, h, testRequest{url: d.url + "-missing"})
	if rec.Body.String() != missing.Body.String() {
		t.Fatalf("refusal %q differs from a missing slug %q", rec.Body.String(), missing.Body.String())
	}
}

func TestPrivateDetailOpensOnlyFromTheFacetThatSelectsIt(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := func(slug, visibility string, active bool) *core.Record {
		return f.create(t, "views", map[string]any{"name": "View " + slug, "slug": slug, "visibility": visibility, "is_active": active})
	}
	public := view("public-view", "public", true)
	locked := view("locked-view", "password", true)
	hidden := view("hidden-view", "private", true)
	inactive := view("inactive-view", "public", false)
	unrelated := view("unrelated-view", "public", true)
	other := view("other-view", "password", true)
	details := createPrivateDetails(t, f, public, locked, hidden, inactive, unrelated)
	lockedJWT, _, err := f.crypto.GenerateViewAccessJWT(locked.Id, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	otherJWT, _, _ := f.crypto.GenerateViewAccessJWT(other.Id, time.Hour)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	for _, d := range details {
		selected := d.sectionConfig(t, "true", "")
		for _, v := range []*core.Record{public, locked, hidden, inactive} {
			setViewSection(t, f, v, selected)
		}
		setViewSection(t, f, unrelated, `{"section":"`+d.section+`","enabled":true,"items":[]}`)

		for _, tc := range []struct {
			name    string
			query   string
			headers map[string]string
			allowed bool
		}{
			{"no from", "", nil, false},
			{"unknown Facet", "?from=nowhere", nil, false},
			{"selecting Facet", "?from=public-view", nil, true},
			{"Facet with an override but no selection", "?from=unrelated-view", nil, false},
			{"password Facet without its password", "?from=locked-view", nil, false},
			{"password Facet with its token", "?from=locked-view", map[string]string{"Authorization": "Bearer " + lockedJWT}, true},
			{"password Facet with another Facet's token", "?from=locked-view", map[string]string{"Authorization": "Bearer " + otherJWT}, false},
			{"private Facet", "?from=hidden-view", nil, false},
			{"inactive Facet", "?from=inactive-view", nil, false},
			{"owner without a Facet", "", map[string]string{"Authorization": adminToken}, true},
		} {
			t.Run(d.section+"/"+tc.name, func(t *testing.T) {
				d.assertOpens(t, h, do(t, h, testRequest{url: d.url + tc.query, headers: tc.headers}), tc.allowed)
			})
		}

		fromPublic := testRequest{url: d.url + "?from=public-view"}
		for _, tc := range []struct {
			name    string
			section string
			allowed bool
		}{
			{"deselected", `{"section":"` + d.section + `","enabled":true,"items":[]}`, false},
			{"section disabled", d.sectionConfig(t, "false", ""), false},
			{"switched to all_public", d.sectionConfig(t, "true", `,"itemSelection":"all_public"`), false},
			{"same id under another section", `{"section":"projects","enabled":true,"items":` + idList(t, d.record.Id) + `}`, false},
			{"selected again", selected, true},
		} {
			t.Run(d.section+"/stale grant/"+tc.name, func(t *testing.T) {
				setViewSection(t, f, public, tc.section)
				d.assertOpens(t, h, do(t, h, fromPublic), tc.allowed)
			})
		}

		public.Set("is_active", false)
		if err := f.app.Save(public); err != nil {
			t.Fatal(err)
		}
		d.assertOpens(t, h, do(t, h, fromPublic), false)
		public.Set("is_active", true)
		if err := f.app.Save(public); err != nil {
			t.Fatal(err)
		}

		d.record.Set("is_draft", true)
		if err := f.app.Save(d.record); err != nil {
			t.Fatal(err)
		}
		d.assertOpens(t, h, do(t, h, fromPublic), false)
	}
}

func TestSingleUseShareOpensPrivateDetailsWithoutASecondUse(t *testing.T) {
	s := newShareVisitFixture(t)
	details := createPrivateDetails(t, s.f, s.view)
	sections := make([]string, 0, len(details))
	for _, d := range details {
		sections = append(sections, d.sectionConfig(t, "true", ""))
	}
	setViewSection(t, s.f, s.view, strings.Join(sections, ","))
	tok := s.token(t, "single-use", s.view.Id, nil)

	proof := s.admit(t, "single-use")
	for _, d := range details {
		url := d.url + "?from=pitch"
		visit := map[string]string{"X-Share-Token": "single-use", shareVisitHeader: proof}
		d.assertOpens(t, s.h, do(t, s.h, testRequest{url: url, headers: visit}), true)
		// The spent link and the proof each admit nobody on their own.
		d.assertOpens(t, s.h, do(t, s.h, testRequest{url: url, headers: map[string]string{"X-Share-Token": "single-use"}}), false)
		d.assertOpens(t, s.h, do(t, s.h, testRequest{url: url, headers: map[string]string{shareVisitHeader: proof}}), false)
	}
	if got := s.useCount(t, tok.Id); got != 1 {
		t.Fatalf("opening details changed use_count to %d, want 1", got)
	}
}

func grantRequest(viewID, token, body string) testRequest {
	return testRequest{method: http.MethodPost, url: "/api/view/" + viewID + "/private-items", body: body, headers: map[string]string{"Authorization": token}}
}

func assertGranted(t *testing.T, h http.Handler, viewID, token string, want int) {
	t.Helper()
	rec := do(t, h, grantRequest(viewID, token, ""))
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, fmt.Sprintf(`"granted":%d`, want))
}

func hasOverride(t *testing.T, f *securityFixture, collection, id, viewID string) bool {
	t.Helper()
	record, err := f.app.FindRecordById(collection, id)
	if err != nil {
		t.Fatal(err)
	}
	return isVisibleInView(record, viewID)
}

func TestPrivateItemGrantFollowsTheSavedFacet(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	other := f.create(t, "views", map[string]any{"name": "Other", "slug": "other", "visibility": "public", "is_active": true})
	project := func(slug, visibility string, fields map[string]any) string {
		base := map[string]any{"title": slug, "slug": slug, "visibility": visibility}
		for k, v := range fields {
			base[k] = v
		}
		return f.create(t, "projects", base).Id
	}
	picked := project("picked", "private", map[string]any{"view_visibility": map[string]bool{other.Id: true}})
	granted := project("granted", "private", map[string]any{"view_visibility": map[string]bool{view.Id: true}})
	draft := project("draft", "private", map[string]any{"is_draft": true})
	public := project("public", "public", nil)
	unpicked := project("unpicked", "private", nil)
	kept := f.create(t, "posts", map[string]any{"title": "Kept", "slug": "kept", "content": "x", "visibility": "private"}).Id
	off := f.create(t, "talks", map[string]any{"title": "Off", "slug": "off", "event": "Conf", "date": "2026-02-01 00:00:00.000Z", "description": "x", "visibility": "private"}).Id
	listed := createTestimonial(t, f, "listed", "approved", "private", "clients", 1)
	scope := map[string]bool{other.Id: true}
	contact := f.create(t, "contact_methods", map[string]any{"type": "email", "value": "a@example.com", "label": "a", "protection_level": "none", "view_visibility": scope})
	custom := f.create(t, "custom_content", map[string]any{"title": "Custom", "visibility": "private"})
	scopeBefore := contact.GetString("view_visibility")
	setViewSection(t, f, view, strings.Join([]string{
		`{"section":"projects","enabled":true,"items":` + idList(t, picked, granted, draft, public, "gone00000000000") + `}`,
		`{"section":"posts","enabled":true,"items":` + idList(t, kept) + `,"itemSelection":"all_public"}`,
		`{"section":"talks","enabled":false,"items":` + idList(t, off) + `}`,
		`{"section":"testimonials","enabled":true,"items":` + idList(t, listed) + `,"list":"clients"}`,
		`{"section":"contacts","enabled":true,"items":` + idList(t, contact.Id) + `}`,
		`{"section":"custom:` + custom.Id + `","enabled":true,"items":[]}`,
		`{"section":"mystery","enabled":true,"items":` + idList(t, unpicked) + `}`,
	}, ","))
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	// A client can't name anything: only the saved selection counts.
	rec := do(t, h, grantRequest(view.Id, adminToken, `{"items":["`+unpicked+`"]}`))
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"granted":1`)

	if !hasOverride(t, f, "projects", picked, view.Id) || !hasOverride(t, f, "projects", picked, other.Id) {
		t.Fatal("selected private item lost a grant in the merge")
	}
	for _, id := range []string{draft, public, unpicked} {
		if hasOverride(t, f, "projects", id, view.Id) {
			t.Fatalf("project %s was granted", id)
		}
	}
	if hasOverride(t, f, "posts", kept, view.Id) {
		t.Fatal("pick kept behind all_public was granted")
	}
	if hasOverride(t, f, "talks", off, view.Id) {
		t.Fatal("pick in a disabled section was granted")
	}
	if hasOverride(t, f, "testimonials", listed, view.Id) {
		t.Fatal("pick a testimonial list ignores was granted")
	}
	if hasOverride(t, f, "custom_content", custom.Id, view.Id) {
		t.Fatal("custom section was granted")
	}
	stored, err := f.app.FindRecordById("contact_methods", contact.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.GetString("view_visibility"); got != scopeBefore {
		t.Fatalf("contact scope changed: %s, want %s", got, scopeBefore)
	}
	assertGranted(t, h, view.Id, adminToken, 0)
}

func TestPrivateItemGrantIsOwnerOnly(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "password", "is_active": true})
	pick := f.create(t, "projects", map[string]any{"title": "Pick", "slug": "pick", "visibility": "private"})
	setViewSection(t, f, view, `{"section":"projects","enabled":true,"items":`+idList(t, pick.Id)+`}`)
	viewJWT, _, err := f.crypto.GenerateViewAccessJWT(view.Id, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	members := core.NewAuthCollection("members")
	if err := f.app.Save(members); err != nil {
		t.Fatal(err)
	}
	member := newAuthRecord(members, "member@example.com", "a-strong-password")
	if err := f.app.Save(member); err != nil {
		t.Fatal(err)
	}
	memberToken, err := member.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	for name, token := range map[string]string{"anonymous": "", "Facet password token": "Bearer " + viewJWT, "other auth collection": memberToken} {
		t.Run(name, func(t *testing.T) {
			assertStatus(t, do(t, h, grantRequest(view.Id, token, "")), http.StatusUnauthorized)
		})
	}
	if hasOverride(t, f, "projects", pick.Id, view.Id) {
		t.Fatal("a refused request wrote a grant")
	}
	assertStatus(t, do(t, h, grantRequest("missing000000000", adminToken, "")), http.StatusNotFound)
}

func TestPrivateItemGrantRollsBackAsOne(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	first := f.create(t, "projects", map[string]any{"title": "First", "slug": "first", "visibility": "private"})
	second := f.create(t, "projects", map[string]any{"title": "Second", "slug": "second", "visibility": "private"})
	setViewSection(t, f, view, `{"section":"projects","enabled":true,"items":`+idList(t, first.Id, second.Id)+`}`)
	failing := true
	f.app.OnRecordUpdate("projects").BindFunc(func(e *core.RecordEvent) error {
		if failing && e.Record.Id == second.Id {
			return errors.New("forced grant failure")
		}
		return e.Next()
	})
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	assertStatus(t, do(t, h, grantRequest(view.Id, adminToken, "")), http.StatusInternalServerError)
	if hasOverride(t, f, "projects", first.Id, view.Id) {
		t.Fatal("first grant survived the failed transaction")
	}

	failing = false
	assertGranted(t, h, view.Id, adminToken, 2)
}

func TestConcurrentPrivateItemGrantsKeepEachOther(t *testing.T) {
	f := newSecurityFixture(t)
	project := f.create(t, "projects", map[string]any{"title": "Shared", "slug": "shared", "visibility": "private"})
	section := `[{"section":"projects","enabled":true,"items":` + idList(t, project.Id) + `}]`
	views := make([]*core.Record, 8)
	for i := range views {
		slug := fmt.Sprintf("facet-%d", i)
		views[i] = f.create(t, "views", map[string]any{"name": slug, "slug": slug, "visibility": "public", "is_active": true, "sections": section})
	}

	counts := make([]int, len(views))
	errs := make([]error, len(views))
	var wg sync.WaitGroup
	for i, view := range views {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counts[i], errs[i] = grantSelectedPrivateRecords(f.pb, view.Id)
		}()
	}
	wg.Wait()

	for i, view := range views {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if counts[i] != 1 {
			t.Fatalf("facet %d granted %d items, want 1", i, counts[i])
		}
		if !hasOverride(t, f, "projects", project.Id, view.Id) {
			t.Fatalf("facet %d lost its grant to another save", i)
		}
	}
}

func TestViewMembershipsReachPastTwoHundredViews(t *testing.T) {
	f := newSecurityFixture(t)
	project := f.create(t, "projects", map[string]any{"title": "Shared", "slug": "shared", "visibility": "public"})
	section := `[{"section":"projects","enabled":true,"items":` + idList(t, project.Id) + `}]`
	const total = 201
	for i := 0; i < total; i++ {
		slug := fmt.Sprintf("facet-%d", i)
		f.create(t, "views", map[string]any{"name": slug, "slug": slug, "visibility": "public", "is_active": true, "sections": section})
	}

	memberships, err := buildViewMemberships(f.pb, "projects")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(memberships[project.Id]); got != total {
		t.Fatalf("project has %d memberships, want %d", got, total)
	}
}
