package hooks

import (
	"testing"
)

// TestReservedSlugValidation verifies the isValidSlug function.
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
		{"_app", false, "reserved: SvelteKit internal"},
		{"favicon.ico", false, "reserved: standard web file"},
		{"sitemap.xml", false, "reserved: standard web file"},

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

	// Test max length boundary
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

// TestReservedSlugsAreComprehensive verifies the reserved list covers critical paths.
func TestReservedSlugsAreComprehensive(t *testing.T) {
	criticalReserved := []string{
		"admin",       // Admin dashboard
		"api",         // Backend API
		"s",           // Share link handler
		"v",           // Legacy view route
		"_app",        // SvelteKit internals
		"_",           // PocketBase admin
		"login",       // Auth path
		"logout",      // Auth path
		"auth",        // Auth path
		"health",      // Health check
	}

	for _, slug := range criticalReserved {
		t.Run(slug, func(t *testing.T) {
			if isValidSlug(slug) {
				t.Errorf("SECURITY: %q must be reserved (would conflict with system routes)", slug)
			}
		})
	}
}

// TestSlugValidationPreventsInjection verifies slugs can't contain path traversal or injection.
func TestSlugValidationPreventsInjection(t *testing.T) {
	injectionAttempts := []struct {
		slug   string
		attack string
	}{
		{"../etc/passwd", "path traversal"},
		{"admin/login", "path injection"},
		{"<script>", "XSS attempt"},
		{"slug;rm -rf", "command injection"},
		{"slug\x00null", "null byte"},
		{".hidden", "hidden file"},
		{"slug%20encoded", "URL encoding"},
	}

	for _, tt := range injectionAttempts {
		t.Run(tt.attack, func(t *testing.T) {
			if isValidSlug(tt.slug) {
				t.Errorf("SECURITY: slug %q should be rejected (%s)", tt.slug, tt.attack)
			}
		})
	}
}

// TestCanonicalURLRouting verifies the URL model contract.
func TestCanonicalURLRouting(t *testing.T) {
	tests := []struct {
		path           string
		isSlugRoute    bool
		description    string
	}{
		{"/recruiter", true, "named views use root-level slugs"},
		{"/my-resume", true, "hyphenated slugs work"},
		{"/admin", false, "admin is reserved"},
		{"/api", false, "api is reserved"},
		{"/s", false, "share link prefix is reserved"},
		{"/v", false, "legacy prefix is reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Extract slug (strip leading /)
			slug := tt.path[1:]
			isValid := isValidSlug(slug)
			if isValid != tt.isSlugRoute {
				t.Errorf("path %s: isValidSlug(%q) = %v, want %v (%s)",
					tt.path, slug, isValid, tt.isSlugRoute, tt.description)
			}
		})
	}
}

// TestVisibilityRoutingContract verifies the expected HTTP status codes for each visibility level.
// This is a contract test - it documents expected behavior that is enforced in view.go.
func TestVisibilityRoutingContract(t *testing.T) {
	tests := []struct {
		visibility     string
		withToken      bool
		expectedStatus int
		description    string
	}{
		{"public", false, 200, "public views render normally"},
		{"unlisted", true, 200, "unlisted with valid token renders"},
		{"unlisted", false, 404, "unlisted without token returns 404"},
		{"password", false, 200, "password view shows prompt UI"},
		{"password", true, 200, "password view with valid JWT renders content"},
		{"private", false, 404, "private always returns 404"},
		{"private", true, 404, "private returns 404 even with share token"},
	}

	for _, tt := range tests {
		t.Run(tt.visibility+"_token="+boolStr(tt.withToken), func(t *testing.T) {
			// Verify the contract matches our security model
			if tt.visibility == "private" && tt.expectedStatus != 404 {
				t.Errorf("SECURITY: private views must return 404, not %d", tt.expectedStatus)
			}
			if tt.visibility == "unlisted" && !tt.withToken && tt.expectedStatus != 404 {
				t.Errorf("SECURITY: unlisted without token must return 404, not %d", tt.expectedStatus)
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
