package hooks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
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

// assertNoTemporaryCredentials fails if a finished operation left a
// temporary credentials file in the data directory.
func assertNoTemporaryCredentials(t *testing.T, dataDir string) {
	t.Helper()
	leftovers, err := filepath.Glob(filepath.Join(dataDir, adminCredentialsFilename+"-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("%d temporary credentials files left in the data directory", len(leftovers))
	}
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
	assertNoTemporaryCredentials(t, f.app.DataDir())
}

// adminState is what a failed reset must leave unchanged.
type adminState struct {
	tokenKey string
	nonce    string
}

func readAdminState(t *testing.T, f *securityFixture, email string) (*core.Record, adminState) {
	t.Helper()
	user, err := f.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		t.Fatal(err)
	}
	return user, adminState{tokenKey: user.TokenKey(), nonce: user.GetString("totp_session_nonce")}
}

func assertAdminUnchanged(t *testing.T, f *securityFixture, email, password string, before adminState) {
	t.Helper()
	user, after := readAdminState(t, f, email)
	if !user.ValidatePassword(password) {
		t.Fatalf("failed reset changed the password for %s", email)
	}
	if after != before {
		t.Fatalf("failed reset changed sessions for %s", email)
	}
}

func TestResetRefusesPendingCredentialsForAnotherAccount(t *testing.T) {
	f := newSecurityFixture(t)
	_, _ = f.createAdmin(t, "first@example.com", "first-password-123", nil)
	_, _ = f.createAdmin(t, "second@example.com", "second-password-123", map[string]any{
		"totp_session_nonce": "second-nonce",
	})

	firstPassword, err := ResetAdminPassword(f.app, "first@example.com")
	if err != nil {
		t.Fatal(err)
	}
	path := AdminCredentialsPath(f.app.DataDir())
	pending, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, before := readAdminState(t, f, "second@example.com")

	password, err := ResetAdminPassword(f.app, "second@example.com")
	if err == nil {
		t.Fatal("reset replaced an unread credentials file")
	}
	if password != "" {
		t.Fatal("refused reset returned a password")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error does not name the pending credentials file: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(pending) {
		t.Fatal("refused reset changed the pending credentials file")
	}
	first, err := f.app.FindAuthRecordByEmail("users", "first@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !first.ValidatePassword(firstPassword) || readTestAdminPassword(t, f.app.DataDir()) != firstPassword {
		t.Fatal("pending credentials no longer sign in to their account")
	}
	assertAdminUnchanged(t, f, "second@example.com", "second-password-123", before)
	assertNoTemporaryCredentials(t, f.app.DataDir())

	// Once the operator has read and deleted the file, the reset works.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	secondPassword, err := ResetAdminPassword(f.app, "second@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if readTestAdminPassword(t, f.app.DataDir()) != secondPassword {
		t.Fatal("reset password does not match the credentials file")
	}
}

func TestConcurrentResetsChangeThePasswordOnce(t *testing.T) {
	f := newSecurityFixture(t)
	_, _ = f.createAdmin(t, "owner@example.com", "old-password-123", nil)

	const attempts = 8
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		winners  []string
		failures int
		start    = make(chan struct{})
	)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			password, err := ResetAdminPassword(f.app, "owner@example.com")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if password != "" {
					t.Error("failed reset returned a password")
				}
				failures++
				return
			}
			winners = append(winners, password)
		}()
	}
	close(start)
	wg.Wait()

	if len(winners) != 1 || failures != attempts-1 {
		t.Fatalf("%d resets succeeded and %d failed, want 1 and %d", len(winners), failures, attempts-1)
	}
	if readTestAdminPassword(t, f.app.DataDir()) != winners[0] {
		t.Fatal("credentials file does not hold the successful reset's password")
	}
	user, err := f.app.FindAuthRecordByEmail("users", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !user.ValidatePassword(winners[0]) {
		t.Fatal("stored password does not match the credentials file")
	}
	assertNoTemporaryCredentials(t, f.app.DataDir())
}

func TestResetSaveFailureRemovesOnlyItsOwnCredentials(t *testing.T) {
	f := newSecurityFixture(t)
	_, _ = f.createAdmin(t, "owner@example.com", "old-password-123", map[string]any{
		"totp_session_nonce": "nonce-before-reset",
	})
	_, before := readAdminState(t, f, "owner@example.com")
	path := AdminCredentialsPath(f.app.DataDir())

	var replaceHandoff bool
	other := []byte("Email: other@example.com\nPassword: other-password\n")
	f.app.OnRecordUpdate("users").BindFunc(func(e *core.RecordEvent) error {
		if replaceHandoff {
			// The operator deletes this reset's file and another reset
			// writes its own before the save fails.
			if err := os.Remove(path); err != nil {
				return err
			}
			if err := os.WriteFile(path, other, 0o600); err != nil {
				return err
			}
		}
		return errors.New("simulated database failure")
	})

	if _, err := ResetAdminPassword(f.app, "owner@example.com"); err == nil {
		t.Fatal("reset succeeded although the account was not saved")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("failed reset left its credentials file behind")
	}
	assertAdminUnchanged(t, f, "owner@example.com", "old-password-123", before)

	replaceHandoff = true
	if _, err := ResetAdminPassword(f.app, "owner@example.com"); err == nil {
		t.Fatal("reset succeeded although the account was not saved")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reset removed another operation's credentials file: %v", err)
	}
	if string(contents) != string(other) {
		t.Fatal("failed reset changed another operation's credentials file")
	}
	assertAdminUnchanged(t, f, "owner@example.com", "old-password-123", before)
	assertNoTemporaryCredentials(t, f.app.DataDir())
}
