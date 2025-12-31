package hooks

import (
	"fmt"
	"net/http"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase/core"
)

// RateLimitMiddleware creates a rate-limiting wrapper for endpoint handlers
// tier: "strict", "moderate", or "normal"
func RateLimitMiddleware(rl *services.RateLimitService, tier string) func(func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			allowed, retryAfter := rl.Allow(e.Request, tier)

			if !allowed {
				// Set standard rate limit headers
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				e.Response.Header().Set("X-RateLimit-Limit", tierToLimit(tier))
				e.Response.Header().Set("X-RateLimit-Remaining", "0")

				// Return uniform 429 response (non-leaky)
				return e.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "too many requests",
				})
			}

			return handler(e)
		}
	}
}

// tierToLimit returns a human-readable limit description
func tierToLimit(tier string) string {
	switch tier {
	case "strict":
		return "5/min"
	case "moderate":
		return "10/min"
	case "normal":
		return "60/min"
	default:
		return "60/min"
	}
}
