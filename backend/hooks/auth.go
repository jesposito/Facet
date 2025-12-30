package hooks

import (
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterAdminAuth registers admin authentication hooks with email allowlist
func RegisterAdminAuth(app *pocketbase.PocketBase) {
	// Get admin allowlist from environment
	allowlistEnv := os.Getenv("ADMIN_EMAILS")
	var allowedEmails []string
	if allowlistEnv != "" {
		for _, email := range strings.Split(allowlistEnv, ",") {
			allowedEmails = append(allowedEmails, strings.TrimSpace(strings.ToLower(email)))
		}
	}

	// Hook into OAuth authentication
	app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		// If no allowlist configured, check if this is the first user
		if len(allowedEmails) == 0 {
			count, _ := app.CountRecords("users")
			if count > 0 {
				return &AdminDeniedError{Message: "Admin access denied. Configure ADMIN_EMAILS environment variable."}
			}
			// Allow first user registration
			return e.Next()
		}

		// Check if the OAuth email is in the allowlist
		email := strings.ToLower(e.OAuth2User.Email)
		for _, allowed := range allowedEmails {
			if email == allowed {
				return e.Next()
			}
		}

		return &AdminDeniedError{Message: "Admin access denied. Your email is not authorized."}
	})

	// Hook into password authentication
	app.OnRecordAuthWithPasswordRequest("users").BindFunc(func(e *core.RecordAuthWithPasswordRequestEvent) error {
		if len(allowedEmails) == 0 {
			return e.Next()
		}

		email := strings.ToLower(e.Record.Email())
		for _, allowed := range allowedEmails {
			if email == allowed {
				return e.Next()
			}
		}

		return &AdminDeniedError{Message: "Admin access denied. Your email is not authorized."}
	})
}

// AdminDeniedError represents an admin access denial
type AdminDeniedError struct {
	Message string
}

func (e *AdminDeniedError) Error() string {
	return e.Message
}

// RequireAuth returns a middleware that requires authentication
func RequireAuth(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		info, _ := e.RequestInfo()
		if info != nil && info.Auth != nil {
			e.Auth = info.Auth
		}
		return e.Next()
	}
}
