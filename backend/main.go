package main

import (
	"log"
	"os"
	"strings"

	"facet/hooks"
	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "facet/migrations"
)

func main() {
	app := pocketbase.New()

	// Initialize services
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Fatal("FATAL: ENCRYPTION_KEY environment variable is required.\n" +
			"This key is used to encrypt sensitive data (API tokens, credentials).\n" +
			"Generate one with: openssl rand -hex 32")
	}
	if len(encryptionKey) < 32 {
		log.Fatal("FATAL: ENCRYPTION_KEY must be at least 32 characters.\n" +
			"Generate one with: openssl rand -hex 32")
	}

	cryptoService := services.NewCryptoService(encryptionKey)
	githubService := services.NewGitHubService()
	aiService := services.NewAIService(cryptoService)
	shareService := services.NewShareService(cryptoService)
	rateLimitService := services.NewRateLimitService()

	// Register migrations
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: strings.HasPrefix(os.Args[0], os.TempDir()),
	})

	// Register custom hooks
	// Note: PocketBase v0.23+ has a built-in /api/health endpoint
	hooks.RegisterCollectionRules(app) // Ensure proper access rules on all collections
	hooks.RegisterAdminAuth(app)
	hooks.RegisterGitHubHooks(app, githubService, aiService, cryptoService)
	hooks.RegisterAIHooks(app, aiService, cryptoService)
	hooks.RegisterShareHooks(app, shareService, cryptoService, rateLimitService)
	hooks.RegisterPasswordHooks(app, cryptoService, rateLimitService)
	hooks.RegisterSiteSettingsHooks(app)
	hooks.RegisterMediaHooks(app)
	hooks.RegisterViewHooks(app, cryptoService, shareService, rateLimitService)
	hooks.RegisterOAuthEnvConfig(app)
	hooks.RegisterAdminHooks(app)
	hooks.RegisterExportHooks(app)
	hooks.RegisterResumeHooks(app, cryptoService)
	hooks.RegisterSeedHook(app)
	hooks.RegisterDemoHooks(app)

	// Security enhancements
	// hooks.RegisterSecurityHeaders(app)
	hooks.CheckHTTPS(app)
	// hooks.RegisterAuditLogging(app)

	// Note: Trusted proxy headers are handled by Caddy in the Docker setup.
	// For standalone deployments, configure your reverse proxy to set
	// X-Forwarded-For, X-Forwarded-Proto, and X-Forwarded-Host headers.

	// Start the server
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
