package main

import (
	"log"
	"os"
	"strings"

	"ownprofile/hooks"
	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "ownprofile/migrations"
)

func main() {
	app := pocketbase.New()

	// Configure data directory from environment
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./pb_data"
	}

	// Initialize services
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Println("WARNING: ENCRYPTION_KEY not set. API tokens will not be encrypted properly.")
		encryptionKey = "default-dev-key-change-in-prod!!" // 32 bytes for dev
	}

	cryptoService := services.NewCryptoService(encryptionKey)
	githubService := services.NewGitHubService()
	aiService := services.NewAIService(cryptoService)
	shareService := services.NewShareService()

	// Register migrations
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: strings.HasPrefix(os.Args[0], os.TempDir()),
	})

	// Register custom hooks
	hooks.RegisterHealthCheck(app)
	hooks.RegisterGitHubHooks(app, githubService, aiService)
	hooks.RegisterAIHooks(app, aiService, cryptoService)
	hooks.RegisterShareHooks(app, shareService)
	hooks.RegisterPasswordHooks(app)
	hooks.RegisterViewHooks(app)

	// Serve static files and SvelteKit app
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Trust proxy headers for reverse proxy setups
		if os.Getenv("TRUST_PROXY") == "true" {
			se.Router.Use(apis.TrustedProxyHeaders())
		}

		return se.Next()
	})

	// Start the server
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
