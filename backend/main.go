package main

import (
	"log"
	"os"
	"strings"

	"ownprofile/hooks"
	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "ownprofile/migrations"
)

func main() {
	app := pocketbase.New()

	// Initialize services
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Println("WARNING: ENCRYPTION_KEY not set. API tokens will not be encrypted properly.")
		encryptionKey = "default-dev-key-change-in-prod!!" // 32 bytes for dev
	}

	cryptoService := services.NewCryptoService(encryptionKey)
	githubService := services.NewGitHubService()
	aiService := services.NewAIService(cryptoService)
	shareService := services.NewShareService(cryptoService)

	// Register migrations
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: strings.HasPrefix(os.Args[0], os.TempDir()),
	})

	// Register custom hooks
	hooks.RegisterHealthCheck(app)
	hooks.RegisterAdminAuth(app)
	hooks.RegisterGitHubHooks(app, githubService, aiService, cryptoService)
	hooks.RegisterAIHooks(app, aiService, cryptoService)
	hooks.RegisterShareHooks(app, shareService, cryptoService)
	hooks.RegisterPasswordHooks(app, cryptoService)
	hooks.RegisterViewHooks(app)
	hooks.RegisterSeedHook(app)

	// Note: Trusted proxy headers are handled by Caddy in the Docker setup.
	// For standalone deployments, configure your reverse proxy to set
	// X-Forwarded-For, X-Forwarded-Proto, and X-Forwarded-Host headers.

	// Start the server
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
