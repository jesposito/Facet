package hooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"facet/services"
)

const browserUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

func setFeatures(t *testing.T, f *securityFixture, features map[string]bool) {
	t.Helper()
	if _, err := services.UpdateSiteSettings(f.app, map[string]any{"enabled_features": features}, nil); err != nil {
		t.Fatal(err)
	}
}

func countAccessLogs(t *testing.T, f *securityFixture) int64 {
	t.Helper()
	n, err := f.app.CountRecords("access_logs")
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func homepageViewCount(t *testing.T, f *securityFixture) int {
	t.Helper()
	settings, err := services.LoadSiteSettings(f.app)
	if err != nil {
		t.Fatal(err)
	}
	return settings.Record.GetInt("homepage_view_count")
}

func TestFeaturesDefaultOnAndPartialUpdatesMerge(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterSiteSettingsHooks(f.pb, f.crypto)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	features := func(rec *httptest.ResponseRecorder) map[string]bool {
		t.Helper()
		assertStatus(t, rec, http.StatusOK)
		var body struct {
			EnabledFeatures map[string]bool `json:"enabled_features"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.EnabledFeatures
	}
	allOn := map[string]bool{"analytics": true, "courses": true, "discussions": true, "newsletter": true}

	if got := features(do(t, h, testRequest{url: "/api/site-settings"})); !reflect.DeepEqual(got, allOn) {
		t.Fatalf("fresh install features = %v, want all on", got)
	}

	// Turning one switch off leaves the others alone and drops unknown keys.
	got := features(do(t, h, testRequest{
		method:  http.MethodPut,
		url:     "/api/site-settings",
		body:    `{"enabled_features":{"newsletter":false,"made_up":false}}`,
		headers: map[string]string{"Authorization": adminToken},
	}))
	want := map[string]bool{"analytics": true, "courses": true, "discussions": true, "newsletter": false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("features after partial update = %v, want %v", got, want)
	}

	rec := do(t, h, testRequest{
		method: http.MethodPut,
		url:    "/api/site-settings",
		body:   `{"enabled_features":{"analytics":false}}`,
	})
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("anonymous settings update returned %d", rec.Code)
	}
}

func TestAnalyticsSwitchStopsAccessLogsAndHomepageCounter(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	h := f.serve(t)
	visit := func(url string) {
		t.Helper()
		assertStatus(t, do(t, h, testRequest{url: url, headers: map[string]string{"User-Agent": browserUA}}), http.StatusOK)
	}

	setFeatures(t, f, map[string]bool{"analytics": false})
	visit("/api/view/portfolio/data")
	visit("/api/homepage")
	visit("/api/default-view")
	time.Sleep(300 * time.Millisecond) // the writes, if any, are asynchronous
	if n := countAccessLogs(t, f); n != 0 {
		t.Fatalf("%d access logs written with analytics off", n)
	}
	if n := homepageViewCount(t, f); n != 0 {
		t.Fatalf("homepage counter = %d with analytics off", n)
	}

	setFeatures(t, f, map[string]bool{"analytics": true})
	visit("/api/view/portfolio/data")
	visit("/api/homepage")
	visit("/api/default-view")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && (countAccessLogs(t, f) < 2 || homepageViewCount(t, f) < 1) {
		time.Sleep(20 * time.Millisecond)
	}
	if n := countAccessLogs(t, f); n != 2 {
		t.Fatalf("%d access logs written with analytics on, want 2", n)
	}
	if n := homepageViewCount(t, f); n != 1 {
		t.Fatalf("homepage counter = %d with analytics on, want 1", n)
	}
}

func TestNewsletterSwitchClosesSignupButNotUnsubscribe(t *testing.T) {
	n := newNewsletterFixture(t, true)
	RegisterSiteSettingsHooks(n.pb, n.crypto)
	h := n.serve(t)
	n.handler = h

	token := "reader-unsubscribe-token"
	n.create(t, "subscribers", map[string]any{
		"email":                  "reader@example.com",
		"status":                 "active",
		"unsubscribe_token_hash": n.crypto.HMACToken(token),
	})
	n.create(t, "subscribers", map[string]any{"email": "other@example.com", "status": "active"})
	_, adminToken := n.createAdmin(t, "owner@example.com", "a-strong-password", nil)

	setFeatures(t, n.securityFixture, map[string]bool{"newsletter": false})

	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"new@example.com"}`})
	assertStatus(t, rec, http.StatusNotFound)
	if _, err := n.app.FindFirstRecordByFilter("subscribers", "email = 'new@example.com'"); err == nil {
		t.Fatal("signup stored a subscriber while the newsletter was off")
	}

	rec = do(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/admin/newsletter/send",
		body:    `{"subject":"Hi","body_markdown":"Hello"}`,
		headers: map[string]string{"Authorization": adminToken},
	})
	assertStatus(t, rec, http.StatusForbidden)

	// Readers can always leave by signed link or RFC 8058 one-click. An
	// address alone proves nothing, so it never unsubscribes anyone.
	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/public/unsubscribe", body: `{"token":"` + token + `"}`})
	assertStatus(t, rec, http.StatusOK)
	if got := n.status(t, "reader@example.com"); got != "unsubscribed" {
		t.Fatalf("status after unsubscribe = %q", got)
	}
	oneClick := "other-one-click-token"
	other, err := n.app.FindFirstRecordByFilter("subscribers", "email = 'other@example.com'")
	if err != nil {
		t.Fatal(err)
	}
	other.Set("unsubscribe_token_hash", n.crypto.HMACToken(oneClick))
	if err := n.app.Save(other); err != nil {
		t.Fatal(err)
	}
	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/public/unsubscribe/by-email", body: `{"email":"other@example.com"}`})
	assertStatus(t, rec, http.StatusOK)
	if got := n.status(t, "other@example.com"); got != "active" {
		t.Fatalf("email-only request changed status to %q", got)
	}
	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/public/unsubscribe/one-click/" + oneClick, body: "List-Unsubscribe=One-Click", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}})
	assertStatus(t, rec, http.StatusOK)
	if got := n.status(t, "other@example.com"); got != "unsubscribed" {
		t.Fatalf("status after one-click unsubscribe = %q", got)
	}

	// Owner data stays reachable.
	rec = do(t, h, testRequest{url: "/api/admin/subscribers", headers: map[string]string{"Authorization": adminToken}})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "reader@example.com")

	if got := n.app.TestMailer.TotalSend(); got != 0 {
		t.Fatalf("%d emails sent while the newsletter was off", got)
	}
}

func TestCoursesAndDiscussionsSwitchesClosePublicRoutes(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterSiteSettingsHooks(f.pb, f.crypto)
	RegisterCourseHooks(f.pb, &services.PlanConfig{}, f.crypto, f.rl)
	RegisterCommentHooks(f.pb, &services.PlanConfig{}, f.crypto, f.share, f.rl)
	h := f.serve(t)

	routes := []testRequest{
		{url: "/api/courses"},
		{method: http.MethodPost, url: "/api/courses/intro/enroll", body: `{"email":"learner@example.com"}`},
		{url: "/api/comments/posts/abc"},
		{method: http.MethodPost, url: "/api/comments/posts/abc", body: `{"body":"hi"}`},
	}

	for _, r := range routes {
		if rec := do(t, h, r); rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), "is turned off") {
			t.Fatalf("%s %s gated while every feature is on", r.method, r.url)
		}
	}

	setFeatures(t, f, map[string]bool{"courses": false, "discussions": false})
	for _, r := range routes {
		rec := do(t, h, r)
		assertStatus(t, rec, http.StatusNotFound)
		assertContains(t, rec, "is turned off")
	}

	// Existing learners' routes aren't the switch's business.
	rec := do(t, h, testRequest{url: "/api/courses/intro"})
	assertNotContains(t, rec, "is turned off")
}
