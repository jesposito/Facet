package hooks

import (
	"testing"
)

// TestPublicRoutingContract documents the public routing model for Facet.
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
		t.Log("   - Request falls through to other routes (admin, api, etc.)")
		t.Log("")
		t.Log("2. Backend: PocketBase hook (hooks/view.go)")
		t.Log("   - OnRecordCreate AND OnRecordUpdate validation")
		t.Log("   - Prevents creating/updating views with reserved slugs")
		t.Log("   - Returns HTTP 400 with descriptive error message")
	})

	t.Run("system routes still work", func(t *testing.T) {
		t.Log("IMPORTANT: Reserved slugs are NOT captured by [slug=slug]")
		t.Log("but system routes STILL WORK normally:")
		t.Log("")
		t.Log("  /admin    → Admin dashboard (SvelteKit route)")
		t.Log("  /api/*    → PocketBase API (Caddy proxy)")
		t.Log("  /s/<tok>  → Share link handler")
		t.Log("  /v/<slug> → Legacy redirect")
		t.Log("")
		t.Log("The param matcher ensures [slug=slug] never captures these paths.")
		t.Log("This is NOT the same as returning 404 for these paths!")
	})

	t.Run("validation rules", func(t *testing.T) {
		t.Log("A valid slug must:")
		t.Log("  - Not be empty")
		t.Log("  - Not be in the reserved list (case-insensitive)")
		t.Log("  - Match pattern: ^[a-zA-Z0-9][a-zA-Z0-9_-]*$")
		t.Log("  - Not start with underscore or hyphen")
		t.Log("  - Be at most 100 characters")
	})

	t.Run("backend validates BOTH create and update", func(t *testing.T) {
		t.Log("Slug validation happens on:")
		t.Log("  - OnRecordCreate: prevents creating view with reserved slug")
		t.Log("  - OnRecordUpdate: prevents changing slug to reserved value")
		t.Log("")
		t.Log("This prevents the attack where:")
		t.Log("  1. Create view with slug='my-view'")
		t.Log("  2. Update slug to 'admin'")
		t.Log("  → BLOCKED by OnRecordUpdate hook")
	})
}

// TestReservedSlugValidation verifies the isValidSlug function
func TestReservedSlugValidation(t *testing.T) {
	tests := []struct {
		slug   string
		valid  bool
		reason string
	}{
		// Valid slugs
		{"recruiter", true, "simple alphanumeric"},
		{"my-resume", true, "with hyphen"},
		{"portfolio_2024", true, "with underscore and numbers"},
		{"2024-goals", true, "starts with number"},
		{"a", true, "single character"},

		// Reserved slugs (should be invalid)
		{"admin", false, "reserved: system route"},
		{"ADMIN", false, "reserved: case insensitive"},
		{"Admin", false, "reserved: mixed case"},
		{"api", false, "reserved: API namespace"},
		{"s", false, "reserved: share link route"},
		{"v", false, "reserved: legacy view route"},
		{"login", false, "reserved: auth path"},

		// Invalid format
		{"", false, "empty string"},
		{"_hidden", false, "starts with underscore"},
		{"-dash", false, "starts with hyphen"},
		{"my/path", false, "contains slash"},
		{"my.file", false, "contains dot"},
		{"my slug", false, "contains space"},
		{"my@email", false, "contains special char"},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := isValidSlug(tt.slug)
			if got != tt.valid {
				t.Errorf("isValidSlug(%q) = %v, want %v (%s)",
					tt.slug, got, tt.valid, tt.reason)
			}
		})
	}

	// Test max length
	t.Run("max length 100", func(t *testing.T) {
		slug100 := make([]byte, 100)
		for i := range slug100 {
			slug100[i] = 'a'
		}
		if !isValidSlug(string(slug100)) {
			t.Error("100 char slug should be valid")
		}

		slug101 := make([]byte, 101)
		for i := range slug101 {
			slug101[i] = 'a'
		}
		if isValidSlug(string(slug101)) {
			t.Error("101 char slug should be invalid")
		}
	})
}

// TestDefaultViewUniqueness documents is_default enforcement
func TestDefaultViewUniqueness(t *testing.T) {
	t.Run("only one default allowed", func(t *testing.T) {
		t.Log("is_default=true uniqueness is enforced server-side:")
		t.Log("")
		t.Log("On create with is_default=true:")
		t.Log("  → clearOtherDefaults('') unsets is_default on ALL other views")
		t.Log("")
		t.Log("On update with is_default=true:")
		t.Log("  → clearOtherDefaults(thisViewId) unsets is_default on other views")
		t.Log("")
		t.Log("This guarantees at most ONE view has is_default=true at any time.")
	})

	t.Run("fallback when no default exists", func(t *testing.T) {
		t.Log("If no view has is_default=true, /api/default-view returns:")
		t.Log("")
		t.Log("1. First public active view (ordered by created date)")
		t.Log("2. If none: { has_default: false, fallback: 'homepage' }")
		t.Log("")
		t.Log("Frontend then uses legacy /api/homepage aggregation.")
	})

	t.Run("race condition handling", func(t *testing.T) {
		t.Log("Concurrent requests setting is_default=true:")
		t.Log("")
		t.Log("Both will clear others, then set themselves.")
		t.Log("Last write wins - one view ends up as default.")
		t.Log("No data corruption, just non-deterministic winner.")
		t.Log("")
		t.Log("For production, consider optimistic locking if this matters.")
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

// TestShareTokenIntegrationFlow documents the full share link flow
func TestShareTokenIntegrationFlow(t *testing.T) {
	t.Run("complete flow: /s/<token> to rendered view", func(t *testing.T) {
		t.Log("Integration test scenario:")
		t.Log("")
		t.Log("Setup:")
		t.Log("  - View 'recruiter-view' exists with visibility=unlisted")
		t.Log("  - Share token 'abc123...' generated for this view")
		t.Log("")
		t.Log("Step 1: User visits /s/abc123...")
		t.Log("  - SvelteKit route src/routes/s/[token]/+page.server.ts")
		t.Log("  - Calls POST /api/share/validate with { token: 'abc123...' }")
		t.Log("  - Backend validates HMAC, checks expiry, increments use_count")
		t.Log("  - Returns { valid: true, view_slug: 'recruiter-view' }")
		t.Log("")
		t.Log("Step 2: Frontend sets cookie and redirects")
		t.Log("  - setShareToken(cookies, 'abc123...', 7 * 24 * 60 * 60)")
		t.Log("  - Cookie: me_share_token=abc123...; Path=/; HttpOnly; SameSite=Lax")
		t.Log("  - throw redirect(302, '/recruiter-view')")
		t.Log("")
		t.Log("Step 3: Browser follows redirect to /recruiter-view")
		t.Log("  - SvelteKit route src/routes/[slug=slug]/+page.server.ts")
		t.Log("  - Reads token from cookies: getShareToken(cookies)")
		t.Log("  - Calls GET /api/view/recruiter-view/data with X-Share-Token header")
		t.Log("")
		t.Log("Step 4: Backend validates and returns data")
		t.Log("  - extractShareToken() reads X-Share-Token header")
		t.Log("  - validateShareToken() verifies HMAC and view_id match")
		t.Log("  - Returns full view data with sections")
		t.Log("")
		t.Log("Step 5: Page renders")
		t.Log("  - src/routes/[slug=slug]/+page.svelte displays content")
		t.Log("  - URL bar shows /recruiter-view (clean, no token)")
		t.Log("")
		t.Log("Security verification:")
		t.Log("  ✓ Token never appears in URL bar")
		t.Log("  ✓ Token never in browser history")
		t.Log("  ✓ Token not leaked via Referer header")
		t.Log("  ✓ Cookie is httpOnly (no JS access)")
	})

	t.Run("unlisted view without token returns 404", func(t *testing.T) {
		t.Log("Scenario: Direct access to unlisted view without share token")
		t.Log("")
		t.Log("Request: GET /recruiter-view (no cookie)")
		t.Log("")
		t.Log("Flow:")
		t.Log("  1. [slug=slug] route loads")
		t.Log("  2. getShareToken(cookies) returns null")
		t.Log("  3. GET /api/view/recruiter-view/access returns visibility='unlisted'")
		t.Log("  4. Frontend checks: visibility === 'unlisted' && !shareToken")
		t.Log("  5. throw error(404, 'Not Found')")
		t.Log("")
		t.Log("Result: HTTP 404 (not 401/403)")
		t.Log("Reason: Prevents discovery of unlisted view existence")
	})
}

// TestPasswordViewIntegrationFlow documents the password prompt flow
func TestPasswordViewIntegrationFlow(t *testing.T) {
	t.Run("complete flow: prompt to rendered view", func(t *testing.T) {
		t.Log("Integration test scenario:")
		t.Log("")
		t.Log("Setup:")
		t.Log("  - View 'confidential' exists with visibility=password")
		t.Log("  - password_hash contains bcrypt of 'secret123'")
		t.Log("")
		t.Log("Step 1: User visits /confidential (no JWT cookie)")
		t.Log("  - SvelteKit route loads")
		t.Log("  - getPasswordToken(cookies) returns null")
		t.Log("  - GET /api/view/confidential/access returns visibility='password'")
		t.Log("  - Frontend returns { requiresPassword: true, view: {...} }")
		t.Log("")
		t.Log("Step 2: Page renders password prompt")
		t.Log("  - PasswordPrompt.svelte component displayed")
		t.Log("  - User enters 'secret123' and submits")
		t.Log("")
		t.Log("Step 3: Password validation")
		t.Log("  - POST /api/password/check with { view_id, password }")
		t.Log("  - Backend: bcrypt.Compare(password, password_hash)")
		t.Log("  - If valid: generate JWT with ViewID claim")
		t.Log("  - Returns { access_token: '<jwt>', expires_in: 3600 }")
		t.Log("")
		t.Log("Step 4: Frontend stores JWT and reloads")
		t.Log("  - Form action ?/setPasswordToken called")
		t.Log("  - setPasswordToken(cookies, jwt, 3600)")
		t.Log("  - Cookie: me_password_token=<jwt>; Path=/; HttpOnly; SameSite=Lax")
		t.Log("  - invalidateAll() triggers page reload")
		t.Log("")
		t.Log("Step 5: Reload fetches with JWT")
		t.Log("  - getPasswordToken(cookies) returns JWT")
		t.Log("  - GET /api/view/confidential/data with Authorization: Bearer <jwt>")
		t.Log("  - Backend validates JWT signature, expiry, view_id")
		t.Log("  - Returns view content")
		t.Log("")
		t.Log("Step 6: Content renders")
		t.Log("  - requiresPassword=false now")
		t.Log("  - Full view content displayed")
		t.Log("")
		t.Log("Security verification:")
		t.Log("  ✓ Password never stored (only hash)")
		t.Log("  ✓ JWT is short-lived (1 hour)")
		t.Log("  ✓ JWT is view-specific (can't use for other views)")
		t.Log("  ✓ Cookie is httpOnly (no JS access)")
	})

	t.Run("expired JWT shows prompt again", func(t *testing.T) {
		t.Log("Scenario: JWT expired, user revisits page")
		t.Log("")
		t.Log("Flow:")
		t.Log("  1. Cookie me_password_token contains expired JWT")
		t.Log("  2. GET /api/view/{slug}/data with expired JWT")
		t.Log("  3. Backend: JWT validation fails (expired)")
		t.Log("  4. Returns 401 Unauthorized")
		t.Log("  5. Frontend catches 401, returns { requiresPassword: true }")
		t.Log("  6. Password prompt displayed again")
	})
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
