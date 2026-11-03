package hooks

import (
	"bytes"
	"log"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func signIn(t *testing.T, h http.Handler, email, password string) int {
	t.Helper()
	body := `{"identity":"` + email + `","password":"` + password + `"}`
	return do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/users/auth-with-password", body: body}).Code
}

var loggedPassword = regexp.MustCompile(`Password: (\S+)`)

// captureLog sends the standard logger to a buffer for one test and puts the
// previous writer back afterwards, so later log calls in the package still work.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&out)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &out
}

// redactPasswords keeps generated passwords out of test failure output.
func redactPasswords(s string) string {
	return loggedPassword.ReplaceAllString(s, "Password: [redacted]")
}

func TestFirstRunAdminGetsAPrivatePassword(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", "owner@example.com")
	f := newSecurityFixture(t)
	if n, _ := f.app.CountRecords("users"); n != 0 {
		t.Fatalf("template database already has %d users", n)
	}

	out := captureLog(t)
	err := seedMinimalData(f.pb)
	if err != nil {
		t.Fatal(err)
	}

	user, err := f.app.FindAuthRecordByEmail("users", "owner@example.com")
	if err != nil {
		t.Fatalf("admin not created: %v", err)
	}
	if user.ValidatePassword(defaultAdminPassword) {
		t.Fatal("a fresh install still uses the published default password")
	}

	password := readTestAdminPassword(t, f.app.DataDir())
	if strings.Contains(out.String(), password) || loggedPassword.MatchString(out.String()) {
		t.Fatal("startup log exposed the private password")
	}
	if len(password) < 20 || !user.ValidatePassword(password) {
		t.Fatal("credential file does not contain a valid random password")
	}
	h := f.serve(t)
	if code := signIn(t, h, "owner@example.com", password); code != http.StatusOK {
		t.Fatalf("signing in with the private credential returned %d", code)
	}
	if code := signIn(t, h, "owner@example.com", defaultAdminPassword); code == http.StatusOK {
		t.Fatal("the published default password signs in")
	}
}

func TestUpgradeKeepsTheOwnersPassword(t *testing.T) {
	f := newSecurityFixture(t)
	f.createAdmin(t, "owner@example.com", "owner-chosen-password", nil)
	f.createAdmin(t, "legacy@example.com", defaultAdminPassword, nil)

	out := captureLog(t)
	err := seedMinimalData(f.pb)
	if err != nil {
		t.Fatal(err)
	}

	if n, _ := f.app.CountRecords("users"); n != 2 {
		t.Fatalf("seeding an existing install changed the user count to %d", n)
	}
	for email, password := range map[string]string{"owner@example.com": "owner-chosen-password", "legacy@example.com": defaultAdminPassword} {
		user, err := f.app.FindAuthRecordByEmail("users", email)
		if err != nil || !user.ValidatePassword(password) {
			t.Fatalf("%s lost their password on upgrade", email)
		}
	}
	if strings.Contains(out.String(), "Password:") {
		t.Fatalf("an existing install printed a password:\n%s", redactPasswords(out.String()))
	}
}

func TestResetAdminPasswordSignsOutOldSessions(t *testing.T) {
	f := newSecurityFixture(t)
	_, oldToken := f.createAdmin(t, "owner@example.com", "old-password-123", map[string]any{"totp_session_nonce": "nonce-before-reset"})
	h := f.serve(t)

	refresh := func(token string) int {
		return do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/users/auth-refresh", headers: map[string]string{"Authorization": token}}).Code
	}
	if code := refresh(oldToken); code != http.StatusOK {
		t.Fatalf("token didn't work before the reset: %d", code)
	}

	password, err := ResetAdminPassword(f.app, "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if password == defaultAdminPassword || len(password) < 20 {
		t.Fatalf("reset produced a weak or published password (%d chars)", len(password))
	}

	if code := refresh(oldToken); code == http.StatusOK {
		t.Fatal("a token issued before the reset still works")
	}
	if code := signIn(t, h, "owner@example.com", "old-password-123"); code == http.StatusOK {
		t.Fatal("the old password still signs in")
	}
	if code := signIn(t, h, "owner@example.com", password); code != http.StatusOK {
		t.Fatalf("the new password returned %d", code)
	}

	user, err := f.app.FindAuthRecordByEmail("users", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.GetString("totp_session_nonce") != "" {
		t.Fatal("the 2FA session survived the reset")
	}

	if _, err := ResetAdminPassword(f.app, "nobody@example.com"); err == nil {
		t.Fatal("resetting an unknown account succeeded")
	}
}
