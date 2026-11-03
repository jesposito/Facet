package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func newSMTPEnvTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

func triggerServe(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()
	err := app.OnServe().Trigger(&core.ServeEvent{App: app}, func(*core.ServeEvent) error { return nil })
	if err != nil {
		t.Fatalf("serve hooks: %v", err)
	}
}

// APP_URL must reach Meta.AppURL even when neither OAuth nor SMTP is configured,
// because newsletter and auth email links are built from it.
func TestSMTPEnvConfigPersistsAppURLWithoutOAuthOrSMTP(t *testing.T) {
	for _, name := range []string{"SMTP_HOST", "GOOGLE_CLIENT_ID", "GITHUB_CLIENT_ID", "APP_NAME"} {
		t.Setenv(name, "")
	}
	t.Setenv("APP_URL", " https://facet.example.com ")

	app := newSMTPEnvTestApp(t)
	RegisterSMTPEnvConfig(app)
	triggerServe(t, app)

	if got := app.Settings().Meta.AppURL; got != "https://facet.example.com" {
		t.Fatalf("Meta.AppURL = %q, want trimmed APP_URL", got)
	}
	if got := app.Settings().Meta.AppName; got != "Facet" {
		t.Fatalf("Meta.AppName = %q, want Facet", got)
	}
	if app.Settings().SMTP.Enabled {
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

			app := newSMTPEnvTestApp(t)
			RegisterSMTPEnvConfig(app)
			triggerServe(t, app)

			// Reload from the database so the test checks what was saved,
			// not only the in-memory settings.
			if err := app.ReloadSettings(); err != nil {
				t.Fatalf("reload settings: %v", err)
			}
			smtp := app.Settings().SMTP
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

	app := newSMTPEnvTestApp(t)
	settings := app.Settings()
	settings.Meta.AppURL = "https://saved.example.com"
	if err := app.Save(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	RegisterSMTPEnvConfig(app)
	triggerServe(t, app)

	if got := app.Settings().Meta.AppURL; got != "https://saved.example.com" {
		t.Fatalf("Meta.AppURL = %q, want saved value preserved", got)
	}
}
