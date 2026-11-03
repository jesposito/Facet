package hooks

import (
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestProjectFromViewNeedsAccessToThatView(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)

	view := func(slug, visibility string, active bool) *core.Record {
		return f.create(t, "views", map[string]any{"name": "View " + slug, "slug": slug, "visibility": visibility, "is_active": active})
	}
	views := map[string]*core.Record{
		"public":   view("public-view", "public", true),
		"password": view("password-view", "password", true),
		"unlisted": view("unlisted-view", "unlisted", true),
		"private":  view("private-view", "private", true),
		"inactive": view("inactive-view", "public", false),
		"other":    view("other-view", "password", true),
	}
	inViews := map[string]bool{}
	for _, v := range views {
		inViews[v.Id] = true
	}
	f.create(t, "projects", map[string]any{"title": "Private project", "slug": "private-project", "description": "PRIVATE WRITEUP", "visibility": "private", "view_visibility": inViews})
	f.create(t, "projects", map[string]any{"title": "Draft project", "slug": "draft-project", "description": "DRAFT WRITEUP", "visibility": "private", "is_draft": true, "view_visibility": inViews})

	shareToken := func(raw string, fields map[string]any) {
		base := map[string]any{"view_id": views["unlisted"].Id, "token_hash": f.share.HMACToken(raw), "is_active": true}
		for k, v := range fields {
			base[k] = v
		}
		f.create(t, "share_tokens", base)
	}
	shareToken("valid-share", nil)
	shareToken("expired-share", map[string]any{"expires_at": time.Now().Add(-time.Hour)})
	shareToken("revoked-share", map[string]any{"is_active": false})
	shareToken("used-up-share", map[string]any{"max_uses": 1, "use_count": 1})

	passwordJWT, _, err := f.crypto.GenerateViewAccessJWT(views["password"].Id, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	otherViewJWT, _, _ := f.crypto.GenerateViewAccessJWT(views["other"].Id, time.Hour)
	expiredJWT, _, _ := f.crypto.GenerateViewAccessJWT(views["password"].Id, -time.Hour)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	for _, tc := range []struct {
		name    string
		url     string
		headers map[string]string
		allowed bool
	}{
		{"no from", "/api/project/private-project", nil, false},
		{"public view", "/api/project/private-project?from=public-view", nil, true},
		{"password view without password", "/api/project/private-project?from=password-view", nil, false},
		{"password view with its token", "/api/project/private-project?from=password-view", map[string]string{"Authorization": "Bearer " + passwordJWT}, true},
		{"password view with another view's token", "/api/project/private-project?from=password-view", map[string]string{"Authorization": "Bearer " + otherViewJWT}, false},
		{"password view with expired token", "/api/project/private-project?from=password-view", map[string]string{"Authorization": "Bearer " + expiredJWT}, false},
		{"unlisted view without share token", "/api/project/private-project?from=unlisted-view", nil, false},
		{"unlisted view with share token", "/api/project/private-project?from=unlisted-view", map[string]string{"X-Share-Token": "valid-share"}, true},
		{"unlisted view with expired share token", "/api/project/private-project?from=unlisted-view", map[string]string{"X-Share-Token": "expired-share"}, false},
		{"unlisted view with revoked share token", "/api/project/private-project?from=unlisted-view", map[string]string{"X-Share-Token": "revoked-share"}, false},
		{"unlisted view with used-up share token", "/api/project/private-project?from=unlisted-view", map[string]string{"X-Share-Token": "used-up-share"}, false},
		{"private view", "/api/project/private-project?from=private-view", nil, false},
		{"inactive view", "/api/project/private-project?from=inactive-view", nil, false},
		{"draft through public view", "/api/project/draft-project?from=public-view", nil, false},
		{"owner", "/api/project/private-project", map[string]string{"Authorization": adminToken}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, testRequest{url: tc.url, headers: tc.headers})
			if tc.allowed {
				assertStatus(t, rec, http.StatusOK)
				assertContains(t, rec, "PRIVATE WRITEUP")
			} else {
				assertStatus(t, rec, http.StatusNotFound)
			}
		})
	}

	// Opening a project from an unlisted view doesn't spend a token use.
	token, err := f.app.FindFirstRecordByFilter("share_tokens", "token_hash = {:h}", map[string]any{"h": f.share.HMACToken("valid-share")})
	if err != nil {
		t.Fatal(err)
	}
	if token.GetInt("use_count") != 0 {
		t.Fatalf("project page consumed a share token use: %d", token.GetInt("use_count"))
	}
}

func TestPrivateViewAccessInfoIsHidden(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	f.create(t, "views", map[string]any{"name": "Secret Recruiter Pitch", "slug": "secret", "visibility": "private", "is_active": true})
	f.create(t, "views", map[string]any{"name": "Locked View", "slug": "locked", "visibility": "password", "is_active": true})
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	rec := do(t, h, testRequest{url: "/api/view/secret/access"})
	assertStatus(t, rec, http.StatusNotFound)
	assertNotContains(t, rec, "Secret Recruiter Pitch")

	rec = do(t, h, testRequest{url: "/api/view/secret/access", headers: map[string]string{"Authorization": adminToken}})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Secret Recruiter Pitch")

	// The password prompt still needs to know the view exists.
	rec = do(t, h, testRequest{url: "/api/view/locked/access"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"requires_password":true`)
}

func TestViewOverrideNeverPublishesDrafts(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	override := map[string]bool{view.Id: true}
	draft := f.create(t, "projects", map[string]any{"title": "DRAFT PROJECT", "slug": "draft", "visibility": "public", "is_draft": true, "view_visibility": override})
	private := f.create(t, "projects", map[string]any{"title": "PRIVATE BUT SHOWN HERE", "slug": "shown", "visibility": "private", "view_visibility": override})
	view.Set("sections", `[{"section":"projects","enabled":true,"items":["`+draft.Id+`","`+private.Id+`"]}]`)
	if err := f.app.Save(view); err != nil {
		t.Fatal(err)
	}
	h := f.serve(t)

	rec := do(t, h, testRequest{url: "/api/view/portfolio/data"})
	assertStatus(t, rec, http.StatusOK)
	assertNotContains(t, rec, "DRAFT PROJECT")
	assertContains(t, rec, "PRIVATE BUT SHOWN HERE")
}
