package hooks

import (
	"os"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterSMTPEnvConfig applies SMTP settings from environment variables
// into PocketBase's built-in SMTP configuration on startup.
// Always sets Meta.AppName (used in PocketBase's built-in auth emails) and
// applies APP_URL to Meta.AppURL when set, so email links and OAuth redirects
// use the public address whether or not SMTP or OAuth is configured.
// SMTP settings only activate when SMTP_HOST is set.
//
// This runs during bootstrap, straight after PocketBase loads the stored
// settings, rather than in OnServe. Saving settings makes PocketBase reload
// them into the live app settings, and PocketBase's logger reads those fields
// without a lock. At bootstrap nothing else is running yet: PocketBase starts
// its own background goroutine (the sqlite version check, which logs) only
// after the rest of the bootstrap chain returns, and the server, cron and
// Facet's OnServe goroutines start later still. In OnServe that goroutine
// could still be logging while the reload rewrote the settings.
//
// The hook must be registered before app.Bootstrap() runs. The normal
// app.Start() path does that.
func RegisterSMTPEnvConfig(app *pocketbase.PocketBase) {
	app.OnBootstrap().BindFunc(func(be *core.BootstrapEvent) error {
		if err := be.Next(); err != nil {
			return err
		}
		applySMTPEnvConfig(be.App)
		return nil
	})
}

// applySMTPEnvConfig edits a copy of the settings and saves only when an
// environment value differs from what is stored. A restart with unchanged
// environment variables therefore leaves the settings row untouched and
// skips the reload entirely.
func applySMTPEnvConfig(app core.App) {
	settings, err := app.Settings().Clone()
	if err != nil {
		app.Logger().Error("SMTP env config: failed to copy settings", "error", err)
		return
	}
	changed := false
	setString := func(field *string, value string) {
		if *field != value {
			*field = value
			changed = true
		}
	}

	// Always ensure AppName is set for PocketBase's built-in auth emails
	// (login notifications, password resets, etc.). PocketBase defaults
	// this to "Acme" which is not useful for anyone.
	if appName := strings.TrimSpace(os.Getenv("APP_NAME")); appName != "" {
		setString(&settings.Meta.AppName, appName)
	} else if settings.Meta.AppName == "" || settings.Meta.AppName == "Acme" {
		setString(&settings.Meta.AppName, "Facet")
	}

	if appURL := strings.TrimSpace(os.Getenv("APP_URL")); appURL != "" {
		setString(&settings.Meta.AppURL, appURL)
	}

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		if changed {
			if err := app.Save(settings); err != nil {
				app.Logger().Error("Failed to update app settings from environment", "error", err)
			} else {
				app.Logger().Info("App settings updated from environment", "appName", settings.Meta.AppName, "appURL", settings.Meta.AppURL)
			}
		}
		app.Logger().Info("SMTP env config: SMTP_HOST not set; leaving SMTP settings unchanged")
		return
	}

	smtp := settings.SMTP

	smtp.Enabled = true
	smtp.Host = host

	// Port (default 587)
	if portStr := strings.TrimSpace(os.Getenv("SMTP_PORT")); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			smtp.Port = port
		} else {
			app.Logger().Warn("SMTP env config: invalid SMTP_PORT, using default 587", "value", portStr)
			smtp.Port = 587
		}
	} else {
		smtp.Port = 587
	}

	if username := strings.TrimSpace(os.Getenv("SMTP_USERNAME")); username != "" {
		smtp.Username = username
	}

	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		smtp.Password = password
	}

	// TLS. PocketBase's mailer treats TLS=true as implicit TLS (encrypted from
	// the first byte, as on port 465). With TLS=false it connects in plain
	// text and upgrades with STARTTLS when the server offers it, which is
	// what port 587 expects. So when SMTP_TLS is unset, only port 465 gets
	// implicit TLS. An explicit value always wins.
	if tlsStr := strings.TrimSpace(os.Getenv("SMTP_TLS")); tlsStr != "" {
		smtp.TLS = tlsStr == "true" || tlsStr == "1"
		if smtp.TLS && smtp.Port == 587 {
			app.Logger().Warn("SMTP env config: SMTP_TLS=true uses implicit TLS, but port 587 normally expects STARTTLS; unset SMTP_TLS or set it to false unless your provider requires implicit TLS on 587",
				"port", smtp.Port)
		}
	} else {
		smtp.TLS = smtp.Port == 465
	}

	// Auth method (default PLAIN)
	if authMethod := strings.TrimSpace(os.Getenv("SMTP_AUTH_METHOD")); authMethod != "" {
		smtp.AuthMethod = strings.ToUpper(authMethod)
	} else {
		smtp.AuthMethod = "PLAIN"
	}

	if smtp != settings.SMTP {
		settings.SMTP = smtp
		changed = true
	}

	// Sender identity
	if senderName := strings.TrimSpace(os.Getenv("SMTP_SENDER_NAME")); senderName != "" {
		setString(&settings.Meta.SenderName, senderName)
	} else if settings.Meta.SenderName == "" || settings.Meta.SenderName == "Support" {
		setString(&settings.Meta.SenderName, "Facet")
	}

	if senderAddress := strings.TrimSpace(os.Getenv("SMTP_SENDER_ADDRESS")); senderAddress != "" {
		setString(&settings.Meta.SenderAddress, senderAddress)
	}

	if changed {
		if err := app.Save(settings); err != nil {
			app.Logger().Error("SMTP env config: failed to save settings", "error", err)
			return
		}
	}

	app.Logger().Info("SMTP env config: SMTP enabled",
		"host", host,
		"port", settings.SMTP.Port,
		"tls", settings.SMTP.TLS,
		"auth", settings.SMTP.AuthMethod,
	)
}
