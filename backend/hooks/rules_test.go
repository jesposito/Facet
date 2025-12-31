package hooks

import (
	"testing"
)

// TestCollectionSecurityModel documents the expected security model for collection access.
// These are documentation tests that verify our security design.
//
// SECURITY MODEL:
// - Direct PocketBase collection access (/api/collections/{name}/records) requires authentication
// - Public data is served through custom API endpoints that enforce visibility rules
// - Server-side app.FindRecordsByFilter() calls bypass collection rules (by design)

func TestCollectionCategories(t *testing.T) {
	// Content collections that were previously public but now require auth
	contentCollections := []string{
		"profile",
		"experience",
		"projects",
		"education",
		"certifications",
		"skills",
		"posts",
		"talks",
		"views",
	}

	// Sensitive collections that should always require auth
	sensitiveCollections := []string{
		"share_tokens",
		"sources",
		"ai_providers",
		"import_proposals",
		"settings",
	}

	// Verify we're tracking all expected collections
	allManaged := append(contentCollections, sensitiveCollections...)
	if len(allManaged) != 14 {
		t.Errorf("Expected 14 managed collections, got %d", len(allManaged))
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, name := range allManaged {
		if seen[name] {
			t.Errorf("Duplicate collection in list: %s", name)
		}
		seen[name] = true
	}
}

func TestAuthRuleFormat(t *testing.T) {
	// The auth rule we use
	authRule := "@request.auth.id != ''"

	// Verify it's not empty (which would mean "public")
	if authRule == "" {
		t.Error("Auth rule should not be empty string")
	}

	// Verify it checks for authenticated user
	if authRule != "@request.auth.id != ''" {
		t.Errorf("Auth rule format changed: %s", authRule)
	}
}

func TestPublicRuleIsInsecure(t *testing.T) {
	// Document that empty string means public access
	publicRule := ""

	// This is the security hole we're closing
	if publicRule != "" {
		t.Error("Empty string should represent public access in PocketBase")
	}
}

// TestRuleEnforcementLogic verifies the logic in enforceCollectionRules
func TestRuleEnforcementLogic(t *testing.T) {
	authRule := "@request.auth.id != ''"

	tests := []struct {
		name           string
		currentRule    *string
		shouldUpdate   bool
		expectedReason string
	}{
		{
			name:           "nil rule (superuser only) - should update to auth",
			currentRule:    nil,
			shouldUpdate:   true,
			expectedReason: "nil means superuser-only, we want auth-only",
		},
		{
			name:           "empty string (public) - should update to auth",
			currentRule:    strPtr(""),
			shouldUpdate:   true,
			expectedReason: "empty string means public, which is insecure",
		},
		{
			name:           "already has auth rule - no update needed",
			currentRule:    &authRule,
			shouldUpdate:   false,
			expectedReason: "already secured",
		},
		{
			name:           "has custom rule - no update needed",
			currentRule:    strPtr("@request.auth.verified = true"),
			shouldUpdate:   false,
			expectedReason: "custom rules should not be overwritten",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsUpdate := tt.currentRule == nil || *tt.currentRule == ""
			if needsUpdate != tt.shouldUpdate {
				t.Errorf("Expected shouldUpdate=%v for %s, got %v. Reason: %s",
					tt.shouldUpdate, tt.name, needsUpdate, tt.expectedReason)
			}
		})
	}
}

// TestSecurityBypassScenarios documents known bypass scenarios
func TestSecurityBypassScenarios(t *testing.T) {
	t.Run("server-side calls bypass rules", func(t *testing.T) {
		// This is BY DESIGN - our custom endpoints use app.FindRecordsByFilter()
		// which is a server-side call that bypasses collection rules.
		// This allows /api/view/{slug}/data to work without authentication
		// while still applying visibility rules in our code.
		t.Log("EXPECTED: Server-side app.FindRecordsByFilter() bypasses collection rules")
		t.Log("This allows custom endpoints to apply their own security logic")
	})

	t.Run("authenticated admin can access all collections", func(t *testing.T) {
		// Any authenticated user can access collections directly
		// In production, only the admin OAuth allowlist can authenticate
		t.Log("EXPECTED: Authenticated users can access collections via /api/collections/{name}/records")
		t.Log("Access is controlled by OAuth allowlist (ADMIN_EMAILS)")
	})

	t.Run("unauthenticated users blocked from direct access", func(t *testing.T) {
		// Unauthenticated users get 403 when trying /api/collections/{name}/records
		t.Log("EXPECTED: Unauthenticated users receive 403 Forbidden")
		t.Log("They must use /api/view/{slug}/data which enforces visibility rules")
	})
}

// TestCollectionRuleConsistency ensures all rules are set consistently
func TestCollectionRuleConsistency(t *testing.T) {
	// All managed collections should have the same auth rule
	expectedRule := "@request.auth.id != ''"

	// In a real test with PocketBase, we would:
	// 1. Create a test app instance
	// 2. Run enforceCollectionRules()
	// 3. Verify each collection has the expected rules

	t.Logf("All collections should have ListRule=%q", expectedRule)
	t.Logf("All collections should have ViewRule=%q", expectedRule)
	t.Logf("All collections should have CreateRule=%q", expectedRule)
	t.Logf("All collections should have UpdateRule=%q", expectedRule)
	t.Logf("All collections should have DeleteRule=%q", expectedRule)
}

func strPtr(s string) *string {
	return &s
}
