package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"facet/hooks"
	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

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
	testimonialService := services.NewTestimonialService(cryptoService)
	rateLimitService := services.NewRateLimitService()

	// Register migrations
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: strings.HasPrefix(os.Args[0], os.TempDir()),
	})

	// Register custom hooks
	// Note: PocketBase v0.23+ has a built-in /api/health endpoint
	hooks.RegisterCollectionRules(app) // Ensure proper access rules on all collections
	hooks.RegisterAdminAuth(app)
	hooks.RegisterPasswordChangeEndpoint(app, rateLimitService) // Password change endpoint for first-time setup
	hooks.RegisterGitHubHooks(app, githubService, aiService, cryptoService)
	hooks.RegisterAIHooks(app, aiService, cryptoService)
	hooks.RegisterShareHooks(app, shareService, cryptoService, rateLimitService)
	hooks.RegisterPasswordHooks(app, cryptoService, rateLimitService)
	hooks.RegisterSiteSettingsHooks(app)
	hooks.RegisterMediaHooks(app)
	hooks.RegisterViewHooks(app, cryptoService, shareService, rateLimitService)
	hooks.RegisterOAuthEnvConfig(app)
	hooks.RegisterExportHooks(app)
	hooks.RegisterResumeHooks(app, cryptoService)
	hooks.RegisterResumeUploadHooks(app, cryptoService) // Resume upload & parsing
	hooks.RegisterSeedHook(app)
	hooks.RegisterDemoHandlers(app)
	hooks.RegisterTestimonialHooks(app, testimonialService, rateLimitService)

	// Security enhancements
	// hooks.RegisterSecurityHeaders(app)
	hooks.CheckHTTPS(app)
	// hooks.RegisterAuditLogging(app)

	// Note: Trusted proxy headers are handled by Caddy in the Docker setup.
	// For standalone deployments, configure your reverse proxy to set
	// X-Forwarded-For, X-Forwarded-Proto, and X-Forwarded-Host headers.

	// Add custom command for resetting admin password
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "reset-admin-password [email]",
		Short: "Reset an admin user's password to the default (changeme123)",
		Long: `Resets the specified admin user's password to 'changeme123' and marks
it as requiring a password change on next login.

If no email is provided, resets the password for admin@example.com.

Example:
  ./facet reset-admin-password
  ./facet reset-admin-password admin@mydomain.com`,
		Run: func(cmd *cobra.Command, args []string) {
			email := "admin@example.com"
			if len(args) > 0 {
				email = args[0]
			}

			// Find the user
			user, err := app.FindAuthRecordByEmail("users", email)
			if err != nil {
				log.Fatalf("ERROR: User with email '%s' not found: %v", email, err)
			}

			// Hash default password
			defaultPassword := "changeme123"
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("ERROR: Failed to hash password: %v", err)
			}

			// Update password and reset the flag
			user.SetRaw("password", string(passwordHash))
			user.Set("password_changed_from_default", false)

			if err := app.Save(user); err != nil {
				log.Fatalf("ERROR: Failed to save password reset: %v", err)
			}

			fmt.Println("========================================")
			fmt.Printf("Password reset successful for: %s\n", email)
			fmt.Println("========================================")
			fmt.Println("New password: changeme123")
			fmt.Println("")
			fmt.Println("⚠️  The user will be prompted to change")
			fmt.Println("   this password on next login.")
			fmt.Println("========================================")
		},
	})

	// Start the server
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
