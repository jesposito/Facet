package hooks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type shareVisitFixture struct {
	f       *securityFixture
	h       http.Handler
	view    *core.Record
	other   *core.Record
	project string
}

func newShareVisitFixture(t *testing.T) *shareVisitFixture {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterShareHooks(f.pb, f.share, f.crypto, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Pitch", "slug": "pitch", "visibility": "unlisted", "is_active": true})
	other := f.create(t, "views", map[string]any{"name": "Other", "slug": "other-pitch", "visibility": "unlisted", "is_active": true})
	project := f.create(t, "projects", map[string]any{"title": "Private project", "slug": "private-project", "description": "PRIVATE WRITEUP", "visibility": "private", "view_visibility": map[string]bool{view.Id: true, other.Id: true}})
	for _, v := range []*core.Record{view, other} {
		setViewSection(t, f, v, `{"section":"projects","enabled":true,"items":`+idList(t, project.Id)+`}`)
	}
	return &shareVisitFixture{f: f, h: f.serve(t), view: view, other: other, project: "/api/project/private-project?from=pitch"}
}

func (s *shareVisitFixture) token(t *testing.T, raw string, viewID string, fields map[string]any) *core.Record {
	t.Helper()
	base := map[string]any{"view_id": viewID, "token_hash": s.f.share.HMACToken(raw), "is_active": true, "max_uses": 1}
	for k, v := range fields {
		base[k] = v
	}
	return s.f.create(t, "share_tokens", base)
}

func (s *shareVisitFixture) useCount(t *testing.T, id string) int {
	t.Helper()
	r, err := s.f.app.FindRecordById("share_tokens", id)
	if err != nil {
		t.Fatal(err)
	}
	return r.GetInt("use_count")
}

func (s *shareVisitFixture) admit(t *testing.T, raw string) string {
	t.Helper()
	rec := do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: map[string]string{"X-Share-Token": raw}})
	assertStatus(t, rec, http.StatusOK)
	proof := rec.Header().Get(shareVisitHeader)
	if proof == "" {
		t.Fatal("admitted visit got no visit proof")
	}
	return proof
}

func (s *shareVisitFixture) project200(t *testing.T, headers map[string]string) bool {
	t.Helper()
	rec := do(t, s.h, testRequest{url: s.project, headers: headers})
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "PRIVATE WRITEUP") {
		return true
	}
	assertStatus(t, rec, http.StatusNotFound)
	return false
}

func TestShareVisitProofKeepsFinalUseVisitOpen(t *testing.T) {
	s := newShareVisitFixture(t)
	tok := s.token(t, "final-use", s.view.Id, nil)

	proof := s.admit(t, "final-use")
	if got := s.useCount(t, tok.Id); got != 1 {
		t.Fatalf("use_count after admission = %d, want 1", got)
	}

	visit := map[string]string{"X-Share-Token": "final-use", shareVisitHeader: proof}
	if !s.project200(t, visit) {
		t.Fatal("admitted visit could not open a project from the view")
	}
	// Returning to the view during the visit works and spends nothing.
	rec := do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: visit})
	assertStatus(t, rec, http.StatusOK)
	if rec.Header().Get(shareVisitHeader) != "" {
		t.Fatal("a continuing visit was issued a new proof")
	}
	if got := s.useCount(t, tok.Id); got != 1 {
		t.Fatalf("visit navigation spent token uses: use_count = %d", got)
	}

	// The used link alone admits no one new.
	raw := map[string]string{"X-Share-Token": "final-use"}
	if s.project200(t, raw) {
		t.Fatal("exhausted raw token opened a project")
	}
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: raw}), http.StatusNotFound)
	// Nor does the proof without the token it was issued for.
	if s.project200(t, map[string]string{shareVisitHeader: proof}) {
		t.Fatal("proof without share token opened a project")
	}
	if got := s.useCount(t, tok.Id); got != 1 {
		t.Fatalf("rejected requests changed use_count to %d", got)
	}
}

func TestShareValidateIssuesVisitProofWithoutDoubleCounting(t *testing.T) {
	s := newShareVisitFixture(t)
	tok := s.token(t, "link-use", s.view.Id, nil)

	rec := do(t, s.h, testRequest{method: http.MethodPost, url: "/api/share/validate", body: `{"token":"link-use"}`})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"valid":true`)
	proof := rec.Header().Get(shareVisitHeader)
	if proof == "" {
		t.Fatal("validated share link got no visit proof")
	}
	if rec.Header().Get(shareVisitExpiresHeader) == "" {
		t.Fatal("visit proof has no expiry")
	}

	// The browser lands on the view and opens a project within the same visit.
	visit := map[string]string{"X-Share-Token": "link-use", shareVisitHeader: proof}
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: visit}), http.StatusOK)
	if !s.project200(t, visit) {
		t.Fatal("validated visit could not open a project")
	}
	if got := s.useCount(t, tok.Id); got != 1 {
		t.Fatalf("one visit spent %d uses of the token", got)
	}

	// A second validation of the exhausted link gets nothing.
	rec = do(t, s.h, testRequest{method: http.MethodPost, url: "/api/share/validate", body: `{"token":"link-use"}`})
	assertContains(t, rec, `"valid":false`)
	if rec.Header().Get(shareVisitHeader) != "" {
		t.Fatal("rejected validation issued a visit proof")
	}
}

func TestShareVisitProofEndsWithTheToken(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(t *testing.T, s *shareVisitFixture, tok *core.Record)
	}{
		{"revoked", func(t *testing.T, s *shareVisitFixture, tok *core.Record) {
			tok.Set("is_active", false)
			if err := s.f.app.Save(tok); err != nil {
				t.Fatal(err)
			}
		}},
		{"expired", func(t *testing.T, s *shareVisitFixture, tok *core.Record) {
			tok.Set("expires_at", time.Now().Add(-time.Minute))
			if err := s.f.app.Save(tok); err != nil {
				t.Fatal(err)
			}
		}},
		{"deleted", func(t *testing.T, s *shareVisitFixture, tok *core.Record) {
			if err := s.f.app.Delete(tok); err != nil {
				t.Fatal(err)
			}
		}},
		{"moved to another view", func(t *testing.T, s *shareVisitFixture, tok *core.Record) {
			tok.Set("view_id", s.other.Id)
			if err := s.f.app.Save(tok); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newShareVisitFixture(t)
			tok := s.token(t, "ending", s.view.Id, nil)
			proof := s.admit(t, "ending")
			visit := map[string]string{"X-Share-Token": "ending", shareVisitHeader: proof}
			if !s.project200(t, visit) {
				t.Fatal("visit was not open before the change")
			}
			tc.change(t, s, tok)
			if s.project200(t, visit) {
				t.Fatal("visit proof outlived its token")
			}
			assertStatus(t, do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: visit}), http.StatusNotFound)
		})
	}
}

func TestShareVisitProofRejectsForgedAndMisboundProofs(t *testing.T) {
	s := newShareVisitFixture(t)
	s.token(t, "mine", s.view.Id, nil)
	s.token(t, "theirs", s.view.Id, map[string]any{"max_uses": 1, "use_count": 1})
	otherTok := s.token(t, "other-view", s.other.Id, map[string]any{"max_uses": 0})
	proof := s.admit(t, "mine")

	otherProofRec := do(t, s.h, testRequest{url: "/api/view/other-pitch/data", headers: map[string]string{"X-Share-Token": "other-view"}})
	assertStatus(t, otherProofRec, http.StatusOK)
	otherProof := otherProofRec.Header().Get(shareVisitHeader)

	passwordJWT, _, err := s.f.crypto.GenerateViewAccessJWT(s.view.Id, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := s.f.crypto.GenerateShareVisitProof(s.view.Id, otherTok.Id, otherTok.GetString("token_hash"), time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	// Signed for this view but naming a token record of another view.
	misbound, err := s.f.crypto.GenerateShareVisitProof(s.view.Id, otherTok.Id, otherTok.GetString("token_hash"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(proof, ".")
	tampered := parts[0] + "." + parts[1] + "." + strings.Repeat("A", len(parts[2]))

	for _, tc := range []struct {
		name    string
		headers map[string]string
	}{
		{"other token with my proof", map[string]string{"X-Share-Token": "theirs", shareVisitHeader: proof}},
		{"other view's proof", map[string]string{"X-Share-Token": "other-view", shareVisitHeader: otherProof}},
		{"other view's proof with my token", map[string]string{"X-Share-Token": "mine", shareVisitHeader: otherProof}},
		{"tampered signature", map[string]string{"X-Share-Token": "theirs", shareVisitHeader: tampered}},
		{"password token as proof", map[string]string{"X-Share-Token": "theirs", shareVisitHeader: passwordJWT}},
		{"expired proof", map[string]string{"X-Share-Token": "other-view", shareVisitHeader: expired}},
		{"misbound proof", map[string]string{"X-Share-Token": "other-view", shareVisitHeader: misbound}},
		{"garbage", map[string]string{"X-Share-Token": "theirs", shareVisitHeader: "not-a-proof"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if s.project200(t, tc.headers) {
				t.Fatal("bad visit proof opened a project from the view")
			}
		})
	}

	// A visit proof is not a password view token.
	pwView := s.f.create(t, "views", map[string]any{"name": "Locked", "slug": "locked", "visibility": "password", "is_active": true})
	pwProof, err := s.f.crypto.GenerateShareVisitProof(pwView.Id, otherTok.Id, otherTok.GetString("token_hash"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, do(t, s.h, testRequest{url: "/api/view/locked/data", headers: map[string]string{"Authorization": "Bearer " + pwProof}}), http.StatusUnauthorized)
}

func TestShareVisitProofLifetimeIsBounded(t *testing.T) {
	s := newShareVisitFixture(t)
	soon := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)
	s.token(t, "short", s.view.Id, map[string]any{"expires_at": soon})
	s.token(t, "long", s.view.Id, nil)

	check := func(raw string, latest time.Time) {
		rec := do(t, s.h, testRequest{url: "/api/view/pitch/data", headers: map[string]string{"X-Share-Token": raw}})
		assertStatus(t, rec, http.StatusOK)
		exp, err := time.Parse(time.RFC3339, rec.Header().Get(shareVisitExpiresHeader))
		if err != nil {
			t.Fatal(err)
		}
		if exp.After(latest) {
			t.Fatalf("%s: proof expires %v, after %v", raw, exp, latest)
		}
		claims, err := s.f.crypto.ValidateShareVisitProof(rec.Header().Get(shareVisitHeader))
		if err != nil {
			t.Fatal(err)
		}
		if claims.ExpiresAt.Time.After(latest) {
			t.Fatalf("%s: signed expiry %v, after %v", raw, claims.ExpiresAt.Time, latest)
		}
	}
	check("short", soon)
	check("long", time.Now().Add(shareVisitProofTTL+time.Second))
}

func TestShareVisitFinalUseAdmitsExactlyOneConcurrentVisit(t *testing.T) {
	s := newShareVisitFixture(t)
	tok := s.token(t, "race", s.view.Id, map[string]any{"max_uses": 3, "use_count": 2})

	results := concurrently(t, s.h, testRequest{url: "/api/view/pitch/data", headers: map[string]string{"X-Share-Token": "race"}})
	var proofs []*httptest.ResponseRecorder
	for _, r := range results {
		if r.Header().Get(shareVisitHeader) != "" {
			proofs = append(proofs, r)
		}
	}
	if got := countStatus(results, http.StatusOK); got != 1 || len(proofs) != 1 {
		t.Fatalf("final use admitted %d visits with %d proofs, want 1", got, len(proofs))
	}
	if got := s.useCount(t, tok.Id); got != 3 {
		t.Fatalf("use_count = %d, want 3", got)
	}
}
