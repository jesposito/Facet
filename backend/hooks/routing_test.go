package hooks

import (
	"testing"
)

// TestPublicRoutingContract documents the public routing model for Me.yaml.
//
// URL MODEL (LinkedIn-style):
// - /           → Default public profile view
// - /<slug>     → Named view (canonical URL)
// - /s/<token>  → Share link entry point (redirects to /<slug>)
// - /v/<slug>   → Legacy route (301 redirects to /<slug>)
//
// Reserved slugs are protected at both frontend (param matcher) and backend (hook).

func TestCanonicalURLs(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		canonicalURL string
		description  string
	}{
		{
			name:         "homepage",
			path:         "/",
			canonicalURL: "/",
			description:  "Root path is the default profile view",
		},
		{
			name:         "named view",
			path:         "/recruiter",
			canonicalURL: "/recruiter",
			description:  "Named views use root-level slugs",
		},
		{
			name:         "legacy view path",
			path:         "/v/recruiter",
			canonicalURL: "/recruiter",
			description:  "Legacy /v/ prefix redirects to root-level slug",
		},
		{
			name:         "share link",
			path:         "/s/abc123...",
			canonicalURL: "/<view-slug>",
			description:  "Share links redirect to canonical URL after setting cookie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Path: %s → Canonical: %s", tt.path, tt.canonicalURL)
			t.Logf("Description: %s", tt.description)
		})
	}
}

func TestLegacyRouteRedirects(t *testing.T) {
	tests := []struct {
		name           string
		legacyPath     string
		redirectTo     string
		statusCode     int
		preserveCookie bool
		reason         string
	}{
		{
			name:           "/v/slug redirects to /slug",
			legacyPath:     "/v/my-resume",
			redirectTo:     "/my-resume",
			statusCode:     301,
			preserveCookie: true,
			reason:         "Permanent redirect; cookies preserved for token access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%s → %s (HTTP %d)", tt.legacyPath, tt.redirectTo, tt.statusCode)
			t.Logf("Cookies preserved: %v", tt.preserveCookie)
			t.Logf("Reason: %s", tt.reason)
		})
	}
}

func TestShareLinkFlow(t *testing.T) {
	t.Run("share link entry and redirect", func(t *testing.T) {
		t.Log("Flow: /s/<token>")
		t.Log("1. Validate token server-side (POST /api/share/validate)")
		t.Log("2. If valid: set httpOnly cookie (me_share_token, SameSite=Lax)")
		t.Log("3. Redirect 302 to /<slug> (canonical URL)")
		t.Log("4. Token is NOT in the final URL")
		t.Log("")
		t.Log("Security benefits:")
		t.Log("  - Token not in browser history")
		t.Log("  - Token not in Referer headers")
		t.Log("  - Token not visible in URL bar")
		t.Log("  - Cookie is httpOnly (no JS access)")
	})

	t.Run("legacy query param cleanup", func(t *testing.T) {
		t.Log("Flow: /<slug>?t=<token>")
		t.Log("1. Validate token")
		t.Log("2. Set httpOnly cookie")
		t.Log("3. Redirect to /<slug> (clean URL)")
		t.Log("")
		t.Log("This handles bookmarks/links created before the new routing model")
	})
}

func TestDefaultViewResolution(t *testing.T) {
	t.Run("default view lookup order", func(t *testing.T) {
		t.Log("When requesting /, the default view is resolved:")
		t.Log("1. Find view where is_default=true AND is_active=true AND visibility='public'")
		t.Log("2. If not found: find first public active view by created date")
		t.Log("3. If not found: fallback to legacy /api/homepage behavior")
		t.Log("")
		t.Log("Note: Only public views can be the default (unlisted/password/private cannot)")
	})

	t.Run("is_default enforcement", func(t *testing.T) {
		t.Log("Only ONE view can have is_default=true at a time")
		t.Log("Setting is_default=true on a view clears it from all other views")
		t.Log("This is enforced by OnRecordCreate/OnRecordUpdate hooks")
	})
}

func TestReservedSlugsProtection(t *testing.T) {
	reservedSlugs := []string{
		// Existing routes
		"admin", "api", "s", "v",
		// SvelteKit internal
		"_app", "_",
		// Static assets
		"assets", "static",
		// Standard web files
		"favicon.ico", "robots.txt", "sitemap.xml",
		// System endpoints
		"health", "healthz", "ready",
		// Common reserved paths
		"login", "logout", "auth", "oauth", "callback",
		// Prevent confusion
		"home", "index", "default", "profile",
	}

	t.Run("reserved slugs list", func(t *testing.T) {
		t.Logf("Protected slugs: %v", reservedSlugs)
		t.Log("")
		t.Log("These slugs CANNOT be used for views because they would")
		t.Log("conflict with system routes or cause user confusion.")
	})

	t.Run("protection layers", func(t *testing.T) {
		t.Log("Reserved slug protection has TWO layers:")
		t.Log("1. Frontend: SvelteKit param matcher (frontend/src/params/slug.ts)")
		t.Log("   - Invalid slugs don't match the /[slug=slug] route")
		t.Log("   - Prevents frontend rendering")
		t.Log("")
		t.Log("2. Backend: PocketBase hook (hooks/view.go)")
		t.Log("   - OnRecordCreate and OnRecordUpdate validation")
		t.Log("   - Prevents creating/updating views with reserved slugs")
		t.Log("   - Returns HTTP 400 with descriptive error message")
	})

	t.Run("validation rules", func(t *testing.T) {
		t.Log("A valid slug must:")
		t.Log("  - Not be empty")
		t.Log("  - Not be in the reserved list (case-insensitive)")
		t.Log("  - Match pattern: ^[a-zA-Z0-9][a-zA-Z0-9_-]*$")
		t.Log("  - Not start with underscore or hyphen")
		t.Log("  - Be at most 100 characters")
	})
}

func TestVisibilityRouting(t *testing.T) {
	tests := []struct {
		visibility     string
		withToken      bool
		expectedStatus int
		description    string
	}{
		{
			visibility:     "public",
			withToken:      false,
			expectedStatus: 200,
			description:    "Public views render normally",
		},
		{
			visibility:     "unlisted",
			withToken:      true,
			expectedStatus: 200,
			description:    "Unlisted with valid token renders",
		},
		{
			visibility:     "unlisted",
			withToken:      false,
			expectedStatus: 404,
			description:    "Unlisted without token returns 404 (not discoverable)",
		},
		{
			visibility:     "password",
			withToken:      false,
			expectedStatus: 200,
			description:    "Password view shows prompt UI",
		},
		{
			visibility:     "password",
			withToken:      true,
			expectedStatus: 200,
			description:    "Password view with valid JWT renders content",
		},
		{
			visibility:     "private",
			withToken:      false,
			expectedStatus: 404,
			description:    "Private always returns 404 (never discoverable)",
		},
		{
			visibility:     "private",
			withToken:      true,
			expectedStatus: 404,
			description:    "Private returns 404 even with share token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.visibility+"_token="+boolStr(tt.withToken), func(t *testing.T) {
			t.Logf("Visibility: %s, Token: %v → HTTP %d",
				tt.visibility, tt.withToken, tt.expectedStatus)
			t.Logf("Description: %s", tt.description)
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestAPIEndpoints(t *testing.T) {
	endpoints := []struct {
		method      string
		path        string
		auth        string
		rateLimit   string
		description string
	}{
		{
			method:      "GET",
			path:        "/api/default-view",
			auth:        "None",
			rateLimit:   "normal (60/min)",
			description: "Returns default view slug or fallback indicator",
		},
		{
			method:      "GET",
			path:        "/api/view/{slug}/access",
			auth:        "None",
			rateLimit:   "normal (60/min)",
			description: "Returns view access requirements (visibility, etc.)",
		},
		{
			method:      "GET",
			path:        "/api/view/{slug}/data",
			auth:        "Token/Password",
			rateLimit:   "normal (60/min)",
			description: "Returns full view data with sections",
		},
		{
			method:      "GET",
			path:        "/api/homepage",
			auth:        "None",
			rateLimit:   "normal (60/min)",
			description: "DEPRECATED - Returns aggregated public content",
		},
		{
			method:      "POST",
			path:        "/api/share/validate",
			auth:        "None",
			rateLimit:   "moderate (10/min)",
			description: "Validates share token, returns view slug if valid",
		},
		{
			method:      "POST",
			path:        "/api/password/check",
			auth:        "None",
			rateLimit:   "strict (5/min)",
			description: "Checks password, returns JWT if correct",
		},
	}

	t.Log("Public API Endpoints for Views:")
	t.Log("")

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			t.Logf("%s %s", ep.method, ep.path)
			t.Logf("  Auth: %s", ep.auth)
			t.Logf("  Rate limit: %s", ep.rateLimit)
			t.Logf("  Description: %s", ep.description)
		})
	}
}
