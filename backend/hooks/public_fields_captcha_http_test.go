package hooks

import (
	"net/http"
	"testing"
)

// A CAPTCHA contact can't be solved yet, so every public response must carry
// its label without its value, even when a view tries to override it.
func TestCaptchaContactValueNeverReachesPublicResponses(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterCollectionRules(f.pb)

	captcha := f.create(t, "contact_methods", map[string]any{"type": "email", "value": "hidden@example.com", "label": "Captcha email", "protection_level": "captcha"})
	open := f.create(t, "contact_methods", map[string]any{"type": "email", "value": "shown@example.com", "label": "Open email", "protection_level": "none"})
	f.create(t, "contact_methods", map[string]any{"type": "email", "value": "private@example.com", "label": "Restricted email", "protection_level": "none", "view_visibility": map[string]bool{"another-view": true}})
	sections := `[{"section":"contacts","enabled":true,"items":["` + captcha.Id + `","` + open.Id + `"],"itemConfig":{"` + captcha.Id + `":{"overrides":{"value":"override-leak@example.com"}}}}]`
	f.create(t, "views", map[string]any{"name": "Public", "slug": "portfolio", "visibility": "public", "is_active": true, "sections": sections})
	h := f.serve(t)

	for _, url := range []string{"/api/homepage", "/api/view/portfolio/data"} {
		t.Run(url, func(t *testing.T) {
			rec := do(t, h, testRequest{url: url})
			assertStatus(t, rec, http.StatusOK)
			assertNotContains(t, rec, "hidden@example.com")
			assertNotContains(t, rec, "override-leak@example.com")
			assertNotContains(t, rec, "private@example.com")
			assertContains(t, rec, "Captcha email")
			assertContains(t, rec, "shown@example.com")
		})
	}

	for _, tc := range []struct {
		url    string
		status int
	}{
		{"/api/collections/contact_methods/records", http.StatusUnauthorized},
		{"/api/collections/contact_methods/records/" + captcha.Id, http.StatusNotFound},
	} {
		t.Run("anonymous "+tc.url, func(t *testing.T) {
			rec := do(t, h, testRequest{url: tc.url})
			assertStatus(t, rec, tc.status)
			assertNotContains(t, rec, "hidden@example.com")
			assertNotContains(t, rec, "override-leak@example.com")
			assertNotContains(t, rec, "private@example.com")
			assertNotContains(t, rec, `"items"`)
		})
	}
}
