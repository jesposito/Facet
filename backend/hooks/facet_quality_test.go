package hooks

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestUnlistedViewIsHiddenWithoutShareAccess(t *testing.T) {
	s := newShareVisitFixture(t)
	tokens := []*core.Record{
		s.token(t, "good", s.view.Id, nil),
		s.token(t, "for-other-view", s.other.Id, nil),
		s.token(t, "expired", s.view.Id, map[string]any{"expires_at": time.Now().Add(-time.Hour)}),
		s.token(t, "revoked", s.view.Id, map[string]any{"is_active": false}),
	}
	spent := s.token(t, "spent", s.view.Id, map[string]any{"max_uses": 1, "use_count": 1})

	for _, tc := range []struct {
		name       string
		headers    map[string]string
		remoteAddr string
	}{
		{name: "no token"},
		{name: "unknown token", headers: map[string]string{"X-Share-Token": "guess"}},
		{name: "another view's token", headers: map[string]string{"X-Share-Token": "for-other-view"}},
		{name: "expired token", headers: map[string]string{"X-Share-Token": "expired"}},
		{name: "revoked token", headers: map[string]string{"X-Share-Token": "revoked"}},
		{name: "used-up token", headers: map[string]string{"X-Share-Token": "spent"}},
		{name: "forged proof", headers: map[string]string{"X-Share-Token": "spent", shareVisitHeader: "not-a-proof"}},
		{name: "claimed internal", headers: map[string]string{"X-Internal": "true"}},
		{name: "claimed internal through the proxy", headers: map[string]string{"X-Internal": "true", "X-Forwarded-For": "203.0.113.9"}, remoteAddr: "127.0.0.1:41000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, path := range []string{"/api/view/pitch/access", "/api/view/pitch/data"} {
				rec := do(t, s.h, testRequest{url: path, headers: tc.headers, remoteAddr: tc.remoteAddr})
				assertStatus(t, rec, http.StatusNotFound)
				assertNotContains(t, rec, "Pitch")
				assertNotContains(t, rec, s.view.Id)
			}
		})
	}

	for _, tok := range tokens {
		if got := s.useCount(t, tok.Id); got != 0 {
			t.Fatalf("refused requests changed use_count to %d", got)
		}
	}
	if got := s.useCount(t, spent.Id); got != 1 {
		t.Fatalf("refused requests changed used-up token use_count to %d", got)
	}
}

func TestUnlistedAccessMetadataSpendsNoTokenUse(t *testing.T) {
	s := newShareVisitFixture(t)
	tok := s.token(t, "single-use", s.view.Id, nil)
	token := map[string]string{"X-Share-Token": "single-use"}

	for range 3 {
		rec := do(t, s.h, testRequest{url: "/api/view/pitch/access", headers: token})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, s.view.Id)
		assertContains(t, rec, `"view_name":"Pitch"`)
	}
	if got := s.useCount(t, tok.Id); got != 0 {
		t.Fatalf("access metadata spent token uses: use_count = %d", got)
	}

	// Only the data request counts, and its proof keeps the visit open.
	proof := s.admit(t, "single-use")
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/pitch/access", headers: token}), http.StatusNotFound)
	visit := map[string]string{"X-Share-Token": "single-use", shareVisitHeader: proof}
	rec := do(t, s.h, testRequest{url: "/api/view/pitch/access", headers: visit})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, s.view.Id)
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: visit}), http.StatusOK)
	if got := s.useCount(t, tok.Id); got != 1 {
		t.Fatalf("one visit spent %d token uses", got)
	}
}

func TestUnlistedViewOpensForOwnerAndRequiresFrontendVisitCredentials(t *testing.T) {
	s := newShareVisitFixture(t)
	_, adminToken := s.f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	owner := map[string]string{"Authorization": adminToken}

	for _, path := range []string{"/api/view/pitch/access", "/api/view/pitch/data"} {
		rec := do(t, s.h, testRequest{url: path, headers: owner})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, s.view.Id)
	}

	// An internal transport marker alone supplies no visitor credentials.
	internal := testRequest{headers: map[string]string{"X-Internal": "true"}, remoteAddr: "127.0.0.1:41000"}
	internal.url = "/api/view/pitch/access"
	assertStatus(t, do(t, s.h, internal), http.StatusNotFound)
	internal.url = "/api/view/pitch/data"
	assertStatus(t, do(t, s.h, internal), http.StatusNotFound)
}

func TestOtherVisibilitiesKeepTheirAccessContracts(t *testing.T) {
	s := newShareVisitFixture(t)
	public := s.f.create(t, "views", map[string]any{"name": "Open Door", "slug": "open-door", "visibility": "public", "is_active": true})
	locked := s.f.create(t, "views", map[string]any{"name": "Locked Door", "slug": "locked-door", "visibility": "password", "is_active": true})
	s.f.create(t, "views", map[string]any{"name": "Hidden Door", "slug": "hidden-door", "visibility": "private", "is_active": true})

	rec := do(t, s.h, testRequest{url: "/api/view/open-door/access"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, public.Id)
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/open-door/data"}), http.StatusOK)

	// The password prompt needs the view's id and name before any credential.
	rec = do(t, s.h, testRequest{url: "/api/view/locked-door/access"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, locked.Id)
	assertContains(t, rec, `"view_name":"Locked Door"`)
	assertContains(t, rec, `"requires_password":true`)
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/locked-door/data"}), http.StatusUnauthorized)

	for _, path := range []string{"/api/view/hidden-door/access", "/api/view/hidden-door/data"} {
		rec = do(t, s.h, testRequest{url: path})
		assertStatus(t, rec, http.StatusNotFound)
		assertNotContains(t, rec, "Hidden Door")
	}
}

type viewPasswordFixture struct {
	f     *securityFixture
	h     http.Handler
	owner map[string]string
}

func newViewPasswordFixture(t *testing.T) *viewPasswordFixture {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterPasswordHooks(f.pb, f.crypto, f.rl)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	return &viewPasswordFixture{f: f, h: f.serve(t), owner: map[string]string{"Authorization": adminToken}}
}

func (p *viewPasswordFixture) send(t *testing.T, method, url string, fields map[string]any) (int, string) {
	t.Helper()
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	rec := do(t, p.h, testRequest{method: method, url: url, body: string(body), headers: p.owner})
	return rec.Code, rec.Body.String()
}

func (p *viewPasswordFixture) createView(t *testing.T, fields map[string]any) (int, string) {
	t.Helper()
	return p.send(t, http.MethodPost, "/api/collections/views/records", fields)
}

func (p *viewPasswordFixture) updateView(t *testing.T, id string, fields map[string]any) (int, string) {
	t.Helper()
	return p.send(t, http.MethodPatch, "/api/collections/views/records/"+id, fields)
}

func (p *viewPasswordFixture) view(t *testing.T, id string) *core.Record {
	t.Helper()
	record, err := p.f.app.FindRecordById("views", id)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func assertPasswordRejected(t *testing.T, code int, body string) {
	t.Helper()
	var response struct {
		Data map[string]struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	if code != http.StatusBadRequest || response.Data["password"].Code != "validation_required" {
		t.Fatalf("status = %d, body = %s; want a 400 password field error", code, body)
	}
}

func TestPasswordFacetCreateNeedsAPassword(t *testing.T) {
	p := newViewPasswordFixture(t)
	current := p.f.create(t, "views", map[string]any{"name": "Current", "slug": "current", "visibility": "public", "is_active": true, "is_default": true})

	for _, tc := range []struct {
		name   string
		fields map[string]any
	}{
		{"no password", nil},
		{"blank password", map[string]any{"password": "   "}},
		{"client-supplied hash", map[string]any{"password_hash": "not-a-configured-password"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]any{"name": "Locked", "slug": "locked", "visibility": "password", "is_active": true, "is_default": true}
			for k, v := range tc.fields {
				fields[k] = v
			}
			code, body := p.createView(t, fields)
			assertPasswordRejected(t, code, body)
			if _, err := p.f.app.FindFirstRecordByFilter("views", "slug = 'locked'"); err == nil {
				t.Fatal("rejected Facet was created")
			}
			if !p.view(t, current.Id).GetBool("is_default") {
				t.Fatal("rejected Facet cleared the existing default")
			}
		})
	}

	code, body := p.createView(t, map[string]any{"name": "Locked", "slug": "locked", "visibility": "password", "is_active": true, "password": "open sesame"})
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", code, body)
	}
	created, err := p.f.app.FindFirstRecordByFilter("views", "slug = 'locked'")
	if err != nil {
		t.Fatal(err)
	}
	if created.GetString("password") != "" || !p.f.crypto.CheckPassword("open sesame", created.GetString("password_hash")) {
		t.Fatal("supplied password was not hashed and cleared")
	}

	check, err := json.Marshal(map[string]string{"view_id": created.Id, "password": "open sesame"})
	if err != nil {
		t.Fatal(err)
	}
	rec := do(t, p.h, testRequest{method: http.MethodPost, url: "/api/password/check", body: string(check)})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "access_token")
}

func TestPasswordFacetUpdateNeedsAConfiguredPassword(t *testing.T) {
	p := newViewPasswordFixture(t)
	current := p.f.create(t, "views", map[string]any{"name": "Current", "slug": "current", "visibility": "public", "is_active": true, "is_default": true})
	view := p.f.create(t, "views", map[string]any{"name": "Pitch", "slug": "pitch", "visibility": "public", "is_active": true})

	code, body := p.updateView(t, view.Id, map[string]any{"visibility": "password", "is_default": true})
	assertPasswordRejected(t, code, body)
	if got := p.view(t, view.Id); got.GetString("visibility") != "public" || got.GetBool("is_default") {
		t.Fatalf("rejected update changed the Facet: visibility %q", got.GetString("visibility"))
	}
	if !p.view(t, current.Id).GetBool("is_default") {
		t.Fatal("rejected update cleared the existing default")
	}

	code, body = p.updateView(t, view.Id, map[string]any{"visibility": "password", "password": "first-secret"})
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", code, body)
	}
	firstHash := p.view(t, view.Id).GetString("password_hash")
	if !p.f.crypto.CheckPassword("first-secret", firstHash) {
		t.Fatal("transition password was not hashed")
	}

	// An unchanged password stays through other edits.
	code, body = p.updateView(t, view.Id, map[string]any{"name": "Renamed"})
	if code != http.StatusOK {
		t.Fatalf("rename status = %d, body = %s", code, body)
	}
	renamed := p.view(t, view.Id)
	if renamed.GetString("name") != "Renamed" || renamed.GetString("password_hash") != firstHash {
		t.Fatal("rename did not keep the configured password")
	}

	code, body = p.updateView(t, view.Id, map[string]any{"password": "   "})
	assertPasswordRejected(t, code, body)
	if p.view(t, view.Id).GetString("password_hash") != firstHash {
		t.Fatal("blank password replaced the configured one")
	}

	code, body = p.updateView(t, view.Id, map[string]any{"password": "second-secret"})
	if code != http.StatusOK {
		t.Fatalf("replacement status = %d, body = %s", code, body)
	}
	replaced := p.view(t, view.Id)
	if replaced.GetString("password") != "" ||
		!p.f.crypto.CheckPassword("second-secret", replaced.GetString("password_hash")) ||
		p.f.crypto.CheckPassword("first-secret", replaced.GetString("password_hash")) {
		t.Fatal("replacement password was not hashed and cleared")
	}

	// Leaving password visibility and returning can reuse the stored hash.
	for _, visibility := range []string{"public", "password"} {
		if code, body = p.updateView(t, view.Id, map[string]any{"visibility": visibility}); code != http.StatusOK {
			t.Fatalf("%s status = %d, body = %s", visibility, code, body)
		}
	}
}

func TestPasswordFacetWithoutHashMustBeRepairedOnSave(t *testing.T) {
	p := newViewPasswordFixture(t)
	broken := p.f.create(t, "views", map[string]any{"name": "Broken", "slug": "broken", "visibility": "password", "is_active": true})

	code, body := p.updateView(t, broken.Id, map[string]any{"name": "Still broken"})
	assertPasswordRejected(t, code, body)
	if p.view(t, broken.Id).GetString("name") != "Broken" {
		t.Fatal("rejected update renamed the Facet")
	}

	if code, body = p.updateView(t, broken.Id, map[string]any{"visibility": "public"}); code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", code, body)
	}
}

func TestPasswordFacetRejectsUnhashablePasswordsWithoutSaving(t *testing.T) {
	p := newViewPasswordFixture(t)
	for _, password := range []string{strings.Repeat("a", 73), strings.Repeat("é", 37)} {
		code, body := p.createView(t, map[string]any{
			"name": "Too long", "slug": "too-long", "visibility": "password", "password": password,
		})
		if code != http.StatusBadRequest || !strings.Contains(body, `"validation_length"`) {
			t.Fatalf("unhashable password: status = %d, body = %s", code, body)
		}
		if _, err := p.f.app.FindFirstRecordByFilter("views", "slug = 'too-long'"); err == nil {
			t.Fatal("unhashable password created a Facet")
		}
	}
}
