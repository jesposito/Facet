package hooks

import (
	"fmt"
	"net/http"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase/core"
)

// RateLimitMiddleware creates a rate-limiting wrapper for endpoint handlers
// tier: "strict", "moderate", or "normal"
//
// Sets standard rate limit headers on ALL responses (RFC 6585, draft-ietf-httpapi-ratelimit-headers):
//   - X-RateLimit-Limit: Maximum requests allowed per window
//   - X-RateLimit-Remaining: Remaining requests in current window
//   - X-RateLimit-Reset: Unix timestamp when the rate limit resets
//   - Retry-After: Seconds until next request allowed (only on 429)
func RateLimitMiddleware(rl *services.RateLimitService, tier string) func(func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			info := rl.AllowWithInfo(e.Request, tier)

			// Always set rate limit headers (allows clients to monitor quota)
			e.Response.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
			e.Response.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
			e.Response.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.Reset))

			if !info.Allowed {
				// Additional header for 429 responses
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", info.RetryAfter))

				// Return uniform 429 response (non-leaky)
				return e.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "too many requests",
				})
			}

			return handler(e)
		}
	}
}

