package hooks

import (
	"encoding/json"
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

	for _, url := range []string{
		"/api/collections/contact_methods/records",
		"/api/collections/contact_methods/records/" + captcha.Id,
	} {
		t.Run("anonymous "+url, func(t *testing.T) {
			rec := do(t, h, testRequest{url: url})
			assertNotContains(t, rec, "hidden@example.com")
			if rec.Code == http.StatusOK {
				var list struct {
					Items []any `json:"items"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Items) != 0 {
					t.Fatalf("anonymous collection read returned records: %s", rec.Body.String())
				}
			} else if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
				t.Fatalf("anonymous collection read status = %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
