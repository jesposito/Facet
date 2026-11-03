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
func RegisterSMTPEnvConfig(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		settings := app.Settings()

		// Always ensure AppName is set for PocketBase's built-in auth emails
		// (login notifications, password resets, etc.). PocketBase defaults
		// this to "Acme" which is not useful for anyone.
		needsSave := false
		if appName := strings.TrimSpace(os.Getenv("APP_NAME")); appName != "" {
			if settings.Meta.AppName != appName {
				settings.Meta.AppName = appName
				needsSave = true
			}
		} else if settings.Meta.AppName == "" || settings.Meta.AppName == "Acme" {
			settings.Meta.AppName = "Facet"
			needsSave = true
		}

		if appURL := strings.TrimSpace(os.Getenv("APP_URL")); appURL != "" && settings.Meta.AppURL != appURL {
			settings.Meta.AppURL = appURL
			needsSave = true
		}

		host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
		if host == "" {
			if needsSave {
				if err := app.Save(settings); err != nil {
					app.Logger().Error("Failed to update app settings from environment", "error", err)
				} else {
					app.Logger().Info("App settings updated from environment", "appName", settings.Meta.AppName, "appURL", settings.Meta.AppURL)
				}
			}
			app.Logger().Info("SMTP env config: SMTP_HOST not set; leaving SMTP settings unchanged")
			return se.Next()
		}

		settings.SMTP.Enabled = true
		settings.SMTP.Host = host

		// Port (default 587)
		if portStr := strings.TrimSpace(os.Getenv("SMTP_PORT")); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				settings.SMTP.Port = port
			} else {
				app.Logger().Warn("SMTP env config: invalid SMTP_PORT, using default 587", "value", portStr)
				settings.SMTP.Port = 587
			}
		} else {
			settings.SMTP.Port = 587
		}

		if username := strings.TrimSpace(os.Getenv("SMTP_USERNAME")); username != "" {
			settings.SMTP.Username = username
		}

		if password := os.Getenv("SMTP_PASSWORD"); password != "" {
			settings.SMTP.Password = password
		}

		// TLS. PocketBase's mailer treats TLS=true as implicit TLS (encrypted from
		// the first byte, as on port 465). With TLS=false it connects in plain
		// text and upgrades with STARTTLS when the server offers it, which is
		// what port 587 expects. So when SMTP_TLS is unset, only port 465 gets
		// implicit TLS. An explicit value always wins.
		if tlsStr := strings.TrimSpace(os.Getenv("SMTP_TLS")); tlsStr != "" {
			settings.SMTP.TLS = tlsStr == "true" || tlsStr == "1"
			if settings.SMTP.TLS && settings.SMTP.Port == 587 {
				app.Logger().Warn("SMTP env config: SMTP_TLS=true uses implicit TLS, but port 587 normally expects STARTTLS; unset SMTP_TLS or set it to false unless your provider requires implicit TLS on 587",
					"port", settings.SMTP.Port)
			}
		} else {
			settings.SMTP.TLS = settings.SMTP.Port == 465
		}

		// Auth method (default PLAIN)
		if authMethod := strings.TrimSpace(os.Getenv("SMTP_AUTH_METHOD")); authMethod != "" {
			settings.SMTP.AuthMethod = strings.ToUpper(authMethod)
		} else {
			settings.SMTP.AuthMethod = "PLAIN"
		}

		// Sender identity
		if senderName := strings.TrimSpace(os.Getenv("SMTP_SENDER_NAME")); senderName != "" {
			settings.Meta.SenderName = senderName
		} else if settings.Meta.SenderName == "" || settings.Meta.SenderName == "Support" {
			settings.Meta.SenderName = "Facet"
		}

		if senderAddress := strings.TrimSpace(os.Getenv("SMTP_SENDER_ADDRESS")); senderAddress != "" {
			settings.Meta.SenderAddress = senderAddress
		}

		if err := app.Save(settings); err != nil {
			app.Logger().Error("SMTP env config: failed to save settings", "error", err)
			return se.Next()
		}

		app.Logger().Info("SMTP env config: SMTP enabled",
			"host", host,
			"port", settings.SMTP.Port,
			"tls", settings.SMTP.TLS,
			"auth", settings.SMTP.AuthMethod,
		)
		return se.Next()
	})
}
