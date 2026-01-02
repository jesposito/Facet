package hooks

import (
	"log/slog"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterOAuthEnvConfig wires OAuth providers from environment variables into the
// PocketBase users collection so admins can enable Google/GitHub login without
// touching the PocketBase UI.
func RegisterOAuthEnvConfig(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		providers := buildOAuthProvidersFromEnv(app.Logger())
		appURL := strings.TrimSpace(os.Getenv("APP_URL"))

		if len(providers) == 0 {
			app.Logger().Info("OAuth env config: no providers configured; leaving existing settings unchanged")
			return se.Next()
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			app.Logger().Error("OAuth env config: failed to load users collection", "error", err)
			return se.Next()
		}

		users.OAuth2.Enabled = true
		users.OAuth2.Providers = providers

		if err := app.Save(users); err != nil {
			app.Logger().Error("OAuth env config: failed to save users collection", "error", err)
			return se.Next()
		}

		// Keep APP_URL in sync for downstream consumers (login redirect helpers, docs).
		if appURL != "" && app.Settings().Meta.AppURL != appURL {
			settings := app.Settings()
			settings.Meta.AppURL = appURL
			if err := app.Save(settings); err != nil {
				app.Logger().Warn("OAuth env config: failed to persist APP_URL to settings", "error", err)
			}
		} else if appURL == "" {
			app.Logger().Warn("OAuth env config: APP_URL is empty; confirm redirect URIs match your deployment")
		}

		app.Logger().Info("OAuth env config: providers enabled", "providers", providerNames(providers))
		return se.Next()
	})
}

func buildOAuthProvidersFromEnv(logger *slog.Logger) []core.OAuth2ProviderConfig {
	var providers []core.OAuth2ProviderConfig

	googleID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	googleSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	githubID := strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID"))
	githubSecret := strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET"))

	if googleID != "" || googleSecret != "" {
		if googleID == "" || googleSecret == "" {
			logger.Warn("OAuth env config: partial Google credentials; ignoring provider until both are set")
		} else {
			providers = append(providers, core.OAuth2ProviderConfig{
				Name:         "google",
				ClientId:     googleID,
				ClientSecret: googleSecret,
				DisplayName:  "Google",
			})
		}
	}

	if githubID != "" || githubSecret != "" {
		if githubID == "" || githubSecret == "" {
			logger.Warn("OAuth env config: partial GitHub credentials; ignoring provider until both are set")
		} else {
			providers = append(providers, core.OAuth2ProviderConfig{
				Name:         "github",
				ClientId:     githubID,
				ClientSecret: githubSecret,
				DisplayName:  "GitHub",
			})
		}
	}

	return providers
}

func providerNames(providers []core.OAuth2ProviderConfig) []string {
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name)
	}
	return names
}
