package hooks

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pquerna/otp/totp"
)

type adminSessionFixture struct {
	*securityFixture
	handler http.Handler
}

func newAdminSessionFixture(t *testing.T) *adminSessionFixture {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterCollectionRules(f.pb)
	RegisterAdminAuth(f.pb)
	RegisterAdminSessionGuard(f.pb)
	RegisterPasswordChangeEndpoint(f.pb, f.rl)
	RegisterTOTPHooks(f.pb, f.crypto, f.rl)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	f.create(t, "views", map[string]any{"name": "Admin only view", "slug": "admin-only", "visibility": "private", "is_active": true})
	f.create(t, "views", map[string]any{"name": "Public view", "slug": "public", "visibility": "public", "is_active": true})
	return &adminSessionFixture{securityFixture: f, handler: f.serve(t)}
}

// canAdmin reports whether a request with these headers gets admin access,
// checked through both PocketBase's collection API and a Facet admin route.
func (a *adminSessionFixture) canAdmin(t *testing.T, headers map[string]string) bool {
	t.Helper()
	list := do(t, a.handler, testRequest{url: "/api/collections/views/records", headers: headers})
	custom := do(t, a.handler, testRequest{url: "/api/admin/view-memberships", headers: headers})
	listed := strings.Contains(list.Body.String(), "Admin only view")
	if listed != (custom.Code == http.StatusOK) {
		t.Fatalf("collection API and admin route disagree: list=%v custom=%d", listed, custom.Code)
	}
	return listed
}

func (a *adminSessionFixture) enableTOTP(t *testing.T, user *core.Record) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Facet", AccountName: user.Email()})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := a.crypto.Encrypt(key.Secret())
	if err != nil {
		t.Fatal(err)
	}
	user.Set("totp_enabled", true)
	user.Set("totp_secret", encrypted)
	if err := a.app.Save(user); err != nil {
		t.Fatal(err)
	}
	return key.Secret()
}

func TestTwoFactorIsEnforcedByTheServer(t *testing.T) {
	a := newAdminSessionFixture(t)
	user, token := a.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	secret := a.enableTOTP(t, user)
	auth := map[string]string{"Authorization": token}

	if a.canAdmin(t, auth) {
		t.Fatal("password-only token reached admin APIs with 2FA enabled")
	}

	// Public pages still work for a half-signed-in admin.
	assertStatus(t, do(t, a.handler, testRequest{url: "/api/view/public/data", headers: auth}), http.StatusOK)

	// The sign-in steps themselves stay open.
	rec := do(t, a.handler, testRequest{url: "/api/totp/status", headers: auth})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"verified":false`)
	assertStatus(t, do(t, a.handler, testRequest{method: http.MethodPost, url: "/api/collections/users/auth-refresh", headers: auth}), http.StatusOK)

	// A password-only attacker can't change the (non-default) password.
	change := `{"currentPassword":"a-strong-password","newPassword":"attacker-password"}`
	if rec := do(t, a.handler, testRequest{method: http.MethodPost, url: "/api/auth/change-password", headers: auth, body: change}); rec.Code == http.StatusOK {
		t.Fatal("password changed without the second factor")
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rec = do(t, a.handler, testRequest{method: http.MethodPost, url: "/api/totp/verify", headers: auth, body: `{"code":"` + code + `"}`})
	assertStatus(t, rec, http.StatusOK)
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == totpCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("verify didn't set the session cookie")
	}

	verified := map[string]string{"Authorization": token, "Cookie": totpCookieName + "=" + sessionCookie.Value}
	if !a.canAdmin(t, verified) {
		t.Fatal("verified admin was refused")
	}

	wrongCookie := map[string]string{"Authorization": token, "Cookie": totpCookieName + "=not-the-nonce"}
	if a.canAdmin(t, wrongCookie) {
		t.Fatal("a guessed session cookie was accepted")
	}

	fresh, err := a.app.FindRecordById("users", user.Id)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Set("totp_session_expires", time.Now().Add(-time.Minute).UTC().Format("2006-01-02 15:04:05.000Z"))
	if err := a.app.Save(fresh); err != nil {
		t.Fatal(err)
	}
	if a.canAdmin(t, verified) {
		t.Fatal("expired 2FA session was accepted")
	}
}

func TestDefaultPasswordMustBeChangedFirst(t *testing.T) {
	a := newAdminSessionFixture(t)
	_, token := a.createAdmin(t, "admin@example.com", defaultAdminPassword, nil)
	auth := map[string]string{"Authorization": token}

	if a.canAdmin(t, auth) {
		t.Fatal("default-password admin reached admin APIs")
	}

	rec := do(t, a.handler, testRequest{url: "/api/auth/check-default-password", headers: auth})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"has_default_password":true`)

	rec = do(t, a.handler, testRequest{method: http.MethodPost, url: "/api/auth/change-password", headers: auth, body: `{"currentPassword":"changeme123","newPassword":"my-own-password"}`})
	assertStatus(t, rec, http.StatusOK)
	var changed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &changed); err != nil || changed.Token == "" {
		t.Fatalf("no new token after password change: %s", rec.Body.String())
	}

	if !a.canAdmin(t, map[string]string{"Authorization": changed.Token}) {
		t.Fatal("admin was still refused after changing the default password")
	}
}

// reset-admin-password restores the default without clearing 2FA. The admin
// UI asks for the new password first, so that change must still work.
func TestDefaultPasswordChangeWorksBeforeTwoFactor(t *testing.T) {
	a := newAdminSessionFixture(t)
	user, token := a.createAdmin(t, "admin@example.com", defaultAdminPassword, nil)
	a.enableTOTP(t, user)
	auth := map[string]string{"Authorization": token}

	rec := do(t, a.handler, testRequest{method: http.MethodPost, url: "/api/auth/change-password", headers: auth, body: `{"currentPassword":"changeme123","newPassword":"my-own-password"}`})
	assertStatus(t, rec, http.StatusOK)
	var changed struct {
		Token string `json:"token"`
	}
	json.Unmarshal(rec.Body.Bytes(), &changed)

	if a.canAdmin(t, map[string]string{"Authorization": changed.Token}) {
		t.Fatal("changing the password skipped the second factor")
	}
}

func TestAdminWithoutTwoFactorKeepsAccess(t *testing.T) {
	a := newAdminSessionFixture(t)
	_, token := a.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	if !a.canAdmin(t, map[string]string{"Authorization": token}) {
		t.Fatal("ordinary admin was refused")
	}
}

func TestSignInRoutesAreRateLimited(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAuthRateLimits(f.pb)
	h := f.serve(t)

	if !f.app.Settings().RateLimits.Enabled {
		t.Fatal("rate limits were not enabled")
	}

	body := `{"identity":"owner@example.com","password":"wrong-password"}`
	var last int
	for i := 0; i < 6; i++ {
		last = do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/users/auth-with-password", body: body, remoteAddr: "203.0.113.7:5000"}).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("sixth sign-in attempt returned %d, want 429", last)
	}

	reset := `{"email":"owner@example.com"}`
	for i := 0; i < 3; i++ {
		last = do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/users/request-password-reset", body: reset, remoteAddr: "203.0.113.8:5000"}).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("third password reset request returned %d, want 429", last)
	}

	// Ordinary API traffic isn't throttled by these rules.
	for i := 0; i < 20; i++ {
		if code := do(t, h, testRequest{url: "/api/health", remoteAddr: "203.0.113.7:5000"}).Code; code == http.StatusTooManyRequests {
			t.Fatal("health check was rate limited")
		}
	}
}

func TestOperatorRateLimitSettingsAreKept(t *testing.T) {
	f := newSecurityFixture(t)
	settings := f.app.Settings()
	settings.RateLimits.Enabled = true
	settings.RateLimits.Rules = []core.RateLimitRule{{Label: "/api/", MaxRequests: 1000, Duration: 10}}
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	if err := enableAuthRateLimits(f.app); err != nil {
		t.Fatal(err)
	}
	if rules := f.app.Settings().RateLimits.Rules; len(rules) != 1 || rules[0].Label != "/api/" {
		t.Fatalf("operator rules were replaced: %+v", rules)
	}
}
