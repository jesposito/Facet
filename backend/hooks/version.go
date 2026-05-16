package hooks

import (
	"net/http"
	"sync"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// versionCache is a tiny in-memory cache for the upstream version lookup.
// It exists to avoid hitting GitHub on every browser tab load and to prevent
// the frontend from needing third-party connect-src permissions (CSP).
type versionCache struct {
	mu        sync.RWMutex
	latest    string
	fetchedAt time.Time
	ttl       time.Duration
}

func (c *versionCache) get() (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.latest == "" {
		return "", false
	}
	if time.Since(c.fetchedAt) > c.ttl {
		return "", false
	}
	return c.latest, true
}

func (c *versionCache) set(v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latest = v
	c.fetchedAt = time.Now()
}

// RegisterVersionHooks exposes a backend-proxied version-check endpoint.
// The frontend hits /api/version-check (same-origin, CSP-friendly) and the
// backend handles the upstream GitHub fetch with a 1-hour in-memory cache.
func RegisterVersionHooks(app *pocketbase.PocketBase, github *services.GitHubService) {
	cache := &versionCache{ttl: 1 * time.Hour}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/version-check", func(e *core.RequestEvent) error {
			// Serve from cache when fresh
			if v, ok := cache.get(); ok {
				return e.JSON(http.StatusOK, map[string]string{"latest": v})
			}

			latest, err := github.FetchLatestVersion("jesposito", "Facet")
			if err != nil || latest == "" {
				app.Logger().Warn("version check upstream failed", "error", err)
				// Return empty string rather than an error so the frontend can
				// silently fail (matches existing client behavior).
				return e.JSON(http.StatusOK, map[string]string{"latest": ""})
			}

			cache.set(latest)
			return e.JSON(http.StatusOK, map[string]string{"latest": latest})
		})

		return se.Next()
	})
}
