package hooks

import (
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// newSMTPEnvTestApp registers the env hook before bootstrapping, the same
// order main.go uses with app.Start().
func newSMTPEnvTestApp(t *testing.T, dataDir string) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	RegisterSMTPEnvConfig(app)
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

// seedStoredSettings opens dataDir without the env hook, lets edit change the
// settings, saves them and closes the app again. It stands in for settings an
// operator saved before a restart.
//
// The save happens inside bootstrap for the same reason the env hook's does:
// once Bootstrap returns, PocketBase's version-check goroutine may be logging,
// and a settings save reloads fields its logger reads without a lock.
func seedStoredSettings(t *testing.T, dataDir string, edit func(*core.Settings)) {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	app.OnBootstrap().BindFunc(func(be *core.BootstrapEvent) error {
		if err := be.Next(); err != nil {
			return err
		}
		settings, err := be.App.Settings().Clone()
		if err != nil {
			return err
		}
		edit(settings)
		return be.App.Save(settings)
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := app.ResetBootstrapState(); err != nil {
		t.Fatalf("close seeded app: %v", err)
	}
}

// storedSettings decodes the settings row straight from the database, so a
// test checks what was persisted without reloading the app's live settings.
func storedSettings(t *testing.T, app core.App) (core.MetaConfig, core.SMTPConfig) {
	t.Helper()
	param := &core.Param{}
	if err := app.ModelQuery(param).Model("settings", param); err != nil {
		t.Fatalf("load stored settings: %v", err)
	}
	var stored struct {
		Meta core.MetaConfig `json:"meta"`
		SMTP core.SMTPConfig `json:"smtp"`
	}
	if err := json.Unmarshal(param.Value, &stored); err != nil {
		t.Fatalf("decode stored settings: %v", err)
	}
	return stored.Meta, stored.SMTP
}

// APP_URL must reach Meta.AppURL even when neither OAuth nor SMTP is configured,
// because newsletter and auth email links are built from it.
func TestSMTPEnvConfigPersistsAppURLWithoutOAuthOrSMTP(t *testing.T) {
	for _, name := range []string{"SMTP_HOST", "GOOGLE_CLIENT_ID", "GITHUB_CLIENT_ID", "APP_NAME"} {
		t.Setenv(name, "")
	}
	t.Setenv("APP_URL", " https://facet.example.com ")

	app := newSMTPEnvTestApp(t, t.TempDir())

	meta, smtp := storedSettings(t, app)
	if meta.AppURL != "https://facet.example.com" || app.Settings().Meta.AppURL != meta.AppURL {
		t.Fatalf("Meta.AppURL stored %q, live %q, want trimmed APP_URL", meta.AppURL, app.Settings().Meta.AppURL)
	}
	if meta.AppName != "Facet" {
		t.Fatalf("Meta.AppName = %q, want Facet", meta.AppName)
	}
	if smtp.Enabled {
		t.Fatal("SMTP must stay disabled when SMTP_HOST is empty")
	}
}

// PocketBase's mailer uses TLS=true for implicit TLS and TLS=false for a plain
// connection upgraded with STARTTLS. Port 587 must therefore default to false.
func TestSMTPEnvConfigTLSDefaultsFollowPort(t *testing.T) {
	cases := []struct {
		name     string
		port     string
		tls      string
		wantPort int
		wantTLS  bool
	}{
		{name: "host only defaults to 587 with STARTTLS", wantPort: 587, wantTLS: false},
		{name: "port 465 defaults to implicit TLS", port: "465", wantPort: 465, wantTLS: true},
		{name: "explicit true on 587 is honoured", port: "587", tls: "true", wantPort: 587, wantTLS: true},
		{name: "explicit false on 465 stays false", port: "465", tls: "false", wantPort: 465, wantTLS: false},
		{name: "explicit false on default port stays false", tls: "false", wantPort: 587, wantTLS: false},
		{name: "invalid port falls back to 587 with STARTTLS", port: "smtp", wantPort: 587, wantTLS: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_AUTH_METHOD", "SMTP_SENDER_NAME", "SMTP_SENDER_ADDRESS", "APP_NAME", "APP_URL"} {
				t.Setenv(name, "")
			}
			t.Setenv("SMTP_HOST", "smtp.example.com")
			t.Setenv("SMTP_PORT", tc.port)
			t.Setenv("SMTP_TLS", tc.tls)

			app := newSMTPEnvTestApp(t, t.TempDir())

			// Read the database row so the test checks what was saved,
			// not only the in-memory settings.
			_, smtp := storedSettings(t, app)
			if !smtp.Enabled || smtp.Host != "smtp.example.com" {
				t.Fatalf("SMTP not enabled for host: %+v", smtp)
			}
			if smtp.Port != tc.wantPort || smtp.TLS != tc.wantTLS {
				t.Fatalf("port/TLS = %d/%v, want %d/%v", smtp.Port, smtp.TLS, tc.wantPort, tc.wantTLS)
			}

			client, ok := app.NewMailClient().(*mailer.SMTPClient)
			if !ok {
				t.Fatalf("mail client = %T, want *mailer.SMTPClient", app.NewMailClient())
			}
			if client.Port != tc.wantPort || client.TLS != tc.wantTLS {
				t.Fatalf("mailer port/TLS = %d/%v, want %d/%v", client.Port, client.TLS, tc.wantPort, tc.wantTLS)
			}
		})
	}
}

// An empty APP_URL keeps the URL an operator saved in PocketBase settings.
func TestSMTPEnvConfigEmptyAppURLKeepsSavedSetting(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("APP_URL", "")

	dataDir := t.TempDir()
	seedStoredSettings(t, dataDir, func(s *core.Settings) {
		s.Meta.AppURL = "https://saved.example.com"
	})

	app := newSMTPEnvTestApp(t, dataDir)

	if got := app.Settings().Meta.AppURL; got != "https://saved.example.com" {
		t.Fatalf("Meta.AppURL = %q, want saved value preserved", got)
	}
}

// SMTP fields the environment leaves unset keep their stored values, and a
// restart with the same environment does not rewrite the settings row.
func TestSMTPEnvConfigPreservesStoredValuesAndSkipsNoOpSave(t *testing.T) {
	for _, name := range []string{"SMTP_PORT", "SMTP_TLS", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_AUTH_METHOD", "SMTP_SENDER_NAME", "SMTP_SENDER_ADDRESS", "APP_NAME"} {
		t.Setenv(name, "")
	}
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("APP_URL", "https://facet.example.com")

	dataDir := t.TempDir()
	seedStoredSettings(t, dataDir, func(s *core.Settings) {
		s.SMTP.Username = "stored-user"
		s.SMTP.Password = "stored-password"
		s.Meta.SenderAddress = "hello@example.com"
		s.Meta.SenderName = "Studio"
	})

	first := newSMTPEnvTestApp(t, dataDir)
	meta, smtp := storedSettings(t, first)
	if !smtp.Enabled || smtp.Host != "smtp.example.com" || smtp.Port != 587 || smtp.TLS {
		t.Fatalf("SMTP = enabled:%v %s:%d tls:%v, want enabled smtp.example.com:587 with STARTTLS",
			smtp.Enabled, smtp.Host, smtp.Port, smtp.TLS)
	}
	if smtp.Username != "stored-user" || smtp.Password != "stored-password" {
		t.Fatal("stored SMTP credentials were not preserved")
	}
	if meta.SenderAddress != "hello@example.com" || meta.SenderName != "Studio" {
		t.Fatalf("sender = %q <%s>, want stored values", meta.SenderName, meta.SenderAddress)
	}
	if meta.AppURL != "https://facet.example.com" {
		t.Fatalf("Meta.AppURL = %q, want APP_URL", meta.AppURL)
	}
	if err := first.ResetBootstrapState(); err != nil {
		t.Fatalf("close first app: %v", err)
	}

	// Restart with the same environment: nothing differs, so nothing is saved.
	restarted := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	var saves atomic.Int32
	restarted.OnModelAfterUpdateSuccess("_params").Bind(&hook.Handler[*core.ModelEvent]{
		Func: func(e *core.ModelEvent) error {
			saves.Add(1)
			return e.Next()
		},
	})
	RegisterSMTPEnvConfig(restarted)
	if err := restarted.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = restarted.ResetBootstrapState() })
	if n := saves.Load(); n != 0 {
		t.Fatalf("settings saved %d times on an unchanged restart, want 0", n)
	}
	if restarted.Settings().SMTP.Password != "stored-password" {
		t.Fatal("stored SMTP password lost on restart")
	}
}
