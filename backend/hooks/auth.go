package hooks

import (
	"net/http"
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

// RegisterPasswordChangeEndpoint registers password change endpoints
func RegisterPasswordChangeEndpoint(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// GET /api/auth/check-default-password - Check if user has default password
		se.Router.GET("/api/auth/check-default-password", func(e *core.RequestEvent) error {
			// Verify user is authenticated
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Check if password matches "changeme123"
			hasDefaultPassword := user.ValidatePassword("changeme123")

			return e.JSON(http.StatusOK, map[string]interface{}{
				"has_default_password": hasDefaultPassword,
			})
		})

		// POST /api/auth/change-password - Change user password
		se.Router.POST("/api/auth/change-password", func(e *core.RequestEvent) error {
			// Verify user is authenticated
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Parse request body
			var data struct {
				CurrentPassword string `json:"currentPassword"`
				NewPassword     string `json:"newPassword"`
			}
			if err := e.BindBody(&data); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid request body",
				})
			}

			// Validate new password
			if len(data.NewPassword) < 8 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "New password must be at least 8 characters",
				})
			}

			// Verify current password
			if !user.ValidatePassword(data.CurrentPassword) {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Current password is incorrect",
				})
			}

			// Set new password
			user.SetPassword(data.NewPassword)

			// Refresh the token key (required after password change)
			user.RefreshTokenKey()

			// Save user record
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to save password change",
					"details": err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Password changed successfully",
			})
		})

		return se.Next()
	})
}
