package hooks

import (
	"log"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// defaultAdminPassword is the published password older releases seeded on
// every install. Only SEED_DATA=dev still seeds it, for local testing. Any
// account still using it must change it before reaching admin APIs.
const defaultAdminPassword = "changeme123"

// Routes an admin can still use before finishing sign-in. auth-refresh keeps
// the token alive (the admin layout calls it before anything else) and the
// TOTP routes complete the second factor. None of them read or change content.
var secondFactorExemptPaths = map[string]bool{
	"/api/collections/users/auth-refresh": true,
	"/api/auth/check-default-password":    true,
	"/api/totp/status":                    true,
	"/api/totp/verify":                    true,
	"/api/totp/clear-session":             true,
}

// The default-password gate also allows the password change itself.
var defaultPasswordExemptPaths = map[string]bool{
	"/api/auth/change-password": true,
}

// RegisterAdminSessionGuard makes the server enforce two parts of admin
// sign-in that the admin UI used to enforce alone:
//
//   - An admin with 2FA enabled must hold the session cookie from
//     /api/totp/verify. A password-only token isn't enough.
//   - An admin still on the default password must change it first.
//
// A request that fails either check is treated as anonymous rather than
// rejected outright. Admin APIs then refuse it as they would any visitor,
// while public pages still render. That matters because the frontend's
// server-side loaders forward the admin token without the TOTP cookie.
// API keys, purchase tokens and view-password tokens don't set e.Auth, so
// they're unaffected. PocketBase superusers use their own MFA settings.
func RegisterAdminSessionGuard(app *pocketbase.PocketBase) {
	defaults := &defaultPasswordCache{}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil || user.Collection().Name != "users" {
				return e.Next()
			}

			path := e.Request.URL.Path
			if secondFactorExemptPaths[path] {
				return e.Next()
			}

			hasDefaultPassword := defaults.check(user)

			if user.GetBool("totp_enabled") && !hasValidTOTPSession(e, user) {
				// The admin UI asks for the new password before the TOTP code.
				// Letting that one request through gives an attacker nothing:
				// it only works with the published default password.
				if !(hasDefaultPassword && defaultPasswordExemptPaths[path]) {
					e.Auth = nil
					return e.Next()
				}
			}

			if hasDefaultPassword && !defaultPasswordExemptPaths[path] {
				e.Auth = nil
			}

			return e.Next()
		})
		return se.Next()
	})
}

// defaultPasswordCache remembers whether each user's current password is the
// default, so each request doesn't pay for a bcrypt comparison. Any save of
// the user record changes "updated", and a password change also replaces
// tokenKey, so a stale answer can't survive a password change.
type defaultPasswordCache struct {
	mu      sync.Mutex
	results map[string]bool
}

func (c *defaultPasswordCache) check(user *core.Record) bool {
	key := user.Id + "|" + user.TokenKey() + "|" + user.GetString("updated")

	c.mu.Lock()
	result, ok := c.results[key]
	c.mu.Unlock()
	if ok {
		return result
	}

	result = user.ValidatePassword(defaultAdminPassword)

	c.mu.Lock()
	if c.results == nil {
		c.results = make(map[string]bool)
	}
	// A handful of admins at most; drop old entries rather than track them.
	if len(c.results) > 256 {
		c.results = make(map[string]bool)
	}
	c.results[key] = result
	c.mu.Unlock()
	return result
}

// authRateLimitRules throttle PocketBase's own sign-in and recovery routes.
// Facet's custom endpoints keep their own limiter. These rules leave out
// PocketBase's catch-all "/api/" default, which would throttle the frontend's
// server-side requests.
var authRateLimitRules = []core.RateLimitRule{
	{Label: "*:auth", MaxRequests: 5, Duration: 30},
	{Label: "*:requestPasswordReset", MaxRequests: 2, Duration: 60},
	{Label: "*:confirmPasswordReset", MaxRequests: 5, Duration: 60},
	{Label: "*:requestVerification", MaxRequests: 2, Duration: 60},
	{Label: "*:requestOTP", MaxRequests: 2, Duration: 60},
}

// RegisterAuthRateLimits turns on PocketBase's built-in rate limiter for
// sign-in routes, which is off by default. If an operator has already enabled
// the limiter, their rules are kept.
func RegisterAuthRateLimits(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := enableAuthRateLimits(app); err != nil {
			log.Printf("ERROR: Failed to enable sign-in rate limits: %v", err)
		}
		return se.Next()
	})
}

func enableAuthRateLimits(app core.App) error {
	settings := app.Settings()
	if settings.RateLimits.Enabled {
		return nil
	}
	settings.RateLimits.Rules = append([]core.RateLimitRule(nil), authRateLimitRules...)
	settings.RateLimits.Enabled = true
	return app.Save(settings)
}
