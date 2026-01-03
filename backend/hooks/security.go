package hooks

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
)

// CheckHTTPS logs a warning if the application is running in production
// without HTTPS. This is informational only and does not block requests.
func CheckHTTPS(app *pocketbase.PocketBase) {
	appURL := os.Getenv("APP_URL")
	publicAppURL := os.Getenv("PUBLIC_APP_URL")

	// Check if we're likely in production (not localhost/dev)
	isDev := appURL == "" ||
		appURL == "http://localhost:8080" ||
		appURL == "http://localhost:5173" ||
		os.Getenv("DEV_MODE") == "true"

	if isDev {
		log.Println("[SECURITY] Running in development mode, HTTPS check skipped")
		return
	}

	// Check if either URL uses HTTPS
	usesHTTPS := false
	if appURL != "" && len(appURL) > 8 && appURL[:8] == "https://" {
		usesHTTPS = true
	}
	if publicAppURL != "" && len(publicAppURL) > 8 && publicAppURL[:8] == "https://" {
		usesHTTPS = true
	}

	if !usesHTTPS {
		log.Println("⚠️  [SECURITY WARNING] Application appears to be running in production without HTTPS")
		log.Println("⚠️  [SECURITY WARNING] APP_URL or PUBLIC_APP_URL should start with https://")
		log.Println("⚠️  [SECURITY WARNING] Running without HTTPS exposes user data and credentials")
	} else {
		log.Println("[SECURITY] HTTPS detected, connection security: OK")
	}
}
