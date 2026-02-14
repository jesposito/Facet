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
// Only activates when SMTP_HOST is set.
func RegisterSMTPEnvConfig(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
		if host == "" {
			app.Logger().Info("SMTP env config: SMTP_HOST not set; leaving existing settings unchanged")
			return se.Next()
		}

		settings := app.Settings()

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

		// TLS (default true)
		if tlsStr := strings.TrimSpace(os.Getenv("SMTP_TLS")); tlsStr != "" {
			settings.SMTP.TLS = tlsStr == "true" || tlsStr == "1"
		} else {
			settings.SMTP.TLS = true
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
