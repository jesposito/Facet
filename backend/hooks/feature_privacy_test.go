package hooks

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"facet/services"
)

func TestAnalyticsSwitchStopsViewCounter(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	h := f.serve(t)
	viewCount := func() int {
		t.Helper()
		r, err := f.app.FindRecordById("views", view.Id)
		if err != nil {
			t.Fatal(err)
		}
		return r.GetInt("view_count")
	}

	setFeatures(t, f, map[string]bool{"analytics": false})
	for _, headers := range []map[string]string{
		{"User-Agent": browserUA},
		{"User-Agent": browserUA, "X-Internal": "true"}, // SSR requests are counted unconditionally when on
	} {
		assertStatus(t, do(t, h, testRequest{url: "/api/view/portfolio/data", headers: headers}), http.StatusOK)
	}
	time.Sleep(300 * time.Millisecond)
	if n := viewCount(); n != 0 {
		t.Fatalf("view_count = %d with analytics off", n)
	}

	setFeatures(t, f, map[string]bool{"analytics": true})
	assertStatus(t, do(t, h, testRequest{url: "/api/view/portfolio/data", headers: map[string]string{"User-Agent": browserUA}}), http.StatusOK)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && viewCount() < 1 {
		time.Sleep(20 * time.Millisecond)
	}
	if n := viewCount(); n != 1 {
		t.Fatalf("view_count = %d with analytics on, want 1", n)
	}
}

func TestAnalyticsSwitchWithholdsGoogleAnalyticsID(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterSiteSettingsHooks(f.pb, f.crypto)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)
	if _, err := services.UpdateSiteSettings(f.app, map[string]any{"ga_measurement_id": "G-TEST123"}, nil); err != nil {
		t.Fatal(err)
	}
	gaID := func(headers map[string]string) string {
		t.Helper()
		rec := do(t, h, testRequest{url: "/api/site-settings", headers: headers})
		assertStatus(t, rec, http.StatusOK)
		var body struct {
			GA string `json:"ga_measurement_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.GA
	}

	if got := gaID(nil); got != "G-TEST123" {
		t.Fatalf("visitor GA id with analytics on = %q", got)
	}
	setFeatures(t, f, map[string]bool{"analytics": false})
	if got := gaID(nil); got != "" {
		t.Fatalf("visitor GA id with analytics off = %q, want none", got)
	}
	if got := gaID(map[string]string{"X-Internal": "true"}); got != "" {
		t.Fatalf("SSR GA id with analytics off = %q, want none", got)
	}
	// The owner still sees the saved ID, so saving the settings page keeps it.
	if got := gaID(map[string]string{"Authorization": adminToken}); got != "G-TEST123" {
		t.Fatalf("owner GA id with analytics off = %q", got)
	}
}

func TestCoursesSwitchClosesCourseCheckoutOnly(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterSiteSettingsHooks(f.pb, f.crypto)
	RegisterPurchaseHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)
	ip := 0
	checkout := func(body, contentType string) testRequest {
		ip++
		return testRequest{
			method:     http.MethodPost,
			url:        "/api/content/checkout",
			body:       body,
			headers:    map[string]string{"Content-Type": contentType},
			remoteAddr: "198.51.100." + string(rune('0'+ip)) + ":4000",
		}
	}
	course := `{"content_type":"courses","content_id":"abc","success_url":"","cancel_url":""}`

	if rec := do(t, h, checkout(course, "application/json")); rec.Code == http.StatusNotFound {
		assertNotContains(t, rec, "is turned off")
	}

	setFeatures(t, f, map[string]bool{"courses": false})
	rec := do(t, h, checkout(course, "application/json"))
	assertStatus(t, rec, http.StatusNotFound)
	assertContains(t, rec, "courses is turned off")

	rec = do(t, h, checkout("content_type=courses&content_id=abc", "application/x-www-form-urlencoded"))
	assertStatus(t, rec, http.StatusNotFound)
	assertContains(t, rec, "courses is turned off")

	// Other paid content still checks out, and the handler still reads the
	// whole body after the gate looked at it.
	rec = do(t, h, checkout(`{"content_type":"posts","content_id":"abc","success_url":"","cancel_url":""}`, "application/json"))
	assertNotContains(t, rec, "is turned off")
	assertNotContains(t, rec, "invalid request")
	assertNotContains(t, rec, "invalid content type")

	// Buyers' access links aren't the switch's business.
	rec = do(t, h, testRequest{url: "/api/content/access/some-token", remoteAddr: "198.51.100.200:4000"})
	assertNotContains(t, rec, "is turned off")
}
