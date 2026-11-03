package hooks

import (
	"os"
	"strings"
	"testing"
)

func readTestAdminPassword(t *testing.T, dataDir string) string {
	t.Helper()
	contents, err := os.ReadFile(AdminCredentialsPath(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	const marker = "\nPassword: "
	_, password, ok := strings.Cut(string(contents), marker)
	if !ok {
		t.Fatal("credentials file has no password")
	}
	return strings.TrimSuffix(password, "\n")
}

func blockAdminCredentialHandoff(t *testing.T, dataDir string) {
	t.Helper()
	if err := os.Mkdir(AdminCredentialsPath(dataDir), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestFirstRunAdminPasswordUsesPrivateFileInsteadOfLog(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", "owner@example.com")
	f := newSecurityFixture(t)
	out := captureLog(t)

	if err := seedMinimalData(f.pb); err != nil {
		t.Fatal(err)
	}

	password := readTestAdminPassword(t, f.app.DataDir())
	if len(password) < 20 {
		t.Fatalf("generated password is only %d characters", len(password))
	}
	user, err := f.app.FindAuthRecordByEmail("users", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !user.ValidatePassword(password) {
		t.Fatal("password in credentials file does not sign in")
	}
	if user.GetBool("password_changed_from_default") {
		t.Fatal("first-run account is not marked as requiring a password change")
	}
	if strings.Contains(out.String(), password) || strings.Contains(out.String(), "Password:") {
		t.Fatal("startup log contains the admin password")
	}
	if !strings.Contains(out.String(), AdminCredentialsPath(f.app.DataDir())) {
		t.Fatal("startup log does not identify the credentials file")
	}
}

func TestSeedCredentialWriteFailureDoesNotCreateAdmin(t *testing.T) {
	f := newSecurityFixture(t)
	blockAdminCredentialHandoff(t, f.app.DataDir())
	out := captureLog(t)

	if err := seedMinimalData(f.pb); err == nil {
		t.Fatal("seed succeeded without a credentials file")
	}
	if count, _ := f.app.CountRecords("users"); count != 0 {
		t.Fatalf("failed seed created %d users", count)
	}
	if strings.Contains(out.String(), "Password:") {
		t.Fatal("failed seed logged a password")
	}
}

func TestResetCredentialWriteFailureLeavesAccountUntouched(t *testing.T) {
	f := newSecurityFixture(t)
	user, _ := f.createAdmin(t, "owner@example.com", "old-password-123", map[string]any{
		"totp_enabled":       true,
		"totp_secret":        "encrypted-secret-before-reset",
		"totp_session_nonce": "nonce-before-reset",
	})
	oldTokenKey := user.TokenKey()
	blockAdminCredentialHandoff(t, f.app.DataDir())

	password, err := ResetAdminPassword(f.app, "owner@example.com")
	if err == nil {
		t.Fatal("reset succeeded without a credentials file")
	}
	if password != "" {
		t.Fatal("failed reset returned a password")
	}

	fresh, err := f.app.FindAuthRecordByEmail("users", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.ValidatePassword("old-password-123") {
		t.Fatal("failed reset changed the account password")
	}
	if fresh.TokenKey() != oldTokenKey {
		t.Fatal("failed reset revoked existing sessions")
	}
	if fresh.GetString("totp_session_nonce") != "nonce-before-reset" {
		t.Fatal("failed reset changed the 2FA session")
	}
	if !fresh.GetBool("totp_enabled") || fresh.GetString("totp_secret") != "encrypted-secret-before-reset" {
		t.Fatal("failed reset changed 2FA settings")
	}
}

func TestResetAdminPasswordWritesCredentialsAndPreservesTwoFactor(t *testing.T) {
	f := newSecurityFixture(t)
	_, _ = f.createAdmin(t, "owner@example.com", "old-password-123", map[string]any{
		"totp_enabled": true,
		"totp_secret":  "encrypted-secret-before-reset",
	})

	password, err := ResetAdminPassword(f.app, "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if password == "" || readTestAdminPassword(t, f.app.DataDir()) != password {
		t.Fatal("reset password does not match the credentials file")
	}
	user, err := f.app.FindAuthRecordByEmail("users", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !user.GetBool("totp_enabled") || user.GetString("totp_secret") != "encrypted-secret-before-reset" {
		t.Fatal("password reset changed 2FA settings")
	}
}
