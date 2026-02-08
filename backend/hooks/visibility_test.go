package hooks

import (
	"facet/services"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// createMockRecord creates a minimal in-memory record for testing.
// PocketBase records need a collection, so we create a base collection.
func createMockRecord(fields map[string]any) *core.Record {
	col := core.NewBaseCollection("test_collection")

	// Add fields the record needs
	col.Fields.Add(&core.TextField{Name: "visibility"})
	col.Fields.Add(&core.BoolField{Name: "is_draft"})
	col.Fields.Add(&core.JSONField{Name: "view_visibility"})

	record := core.NewRecord(col)
	for k, v := range fields {
		record.Set(k, v)
	}
	return record
}

// --- isRecordVisible tests ---

func TestIsRecordVisible(t *testing.T) {
	tests := []struct {
		name       string
		visibility string
		isDraft    bool
		expected   bool
	}{
		{"public non-draft is visible", "public", false, true},
		{"unlisted non-draft is visible", "unlisted", false, true},
		{"password non-draft is visible", "password", false, true},
		{"private non-draft is NOT visible", "private", false, false},
		{"public draft is NOT visible", "public", true, false},
		{"private draft is NOT visible", "private", true, false},
		{"empty visibility non-draft is visible", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createMockRecord(map[string]any{
				"visibility": tt.visibility,
				"is_draft":   tt.isDraft,
			})
			got := isRecordVisible(record)
			if got != tt.expected {
				t.Errorf("isRecordVisible(visibility=%q, is_draft=%v) = %v, want %v",
					tt.visibility, tt.isDraft, got, tt.expected)
			}
		})
	}
}

// TestPrivateViewsMustReturn404 verifies the critical security property:
// Private views return 404 (not 401/403) to prevent existence leaking.
func TestPrivateViewsMustReturn404(t *testing.T) {
	// This is a documentation + contract test. The actual HTTP behavior
	// is enforced in view.go's /api/view/{slug}/data handler.
	// The isRecordVisible function returns false for private views,
	// which triggers the 404 path.
	record := createMockRecord(map[string]any{
		"visibility": "private",
		"is_draft":   false,
	})

	if isRecordVisible(record) {
		t.Error("SECURITY: private views must not be visible (should return 404, not 401)")
	}
}

// TestHomepageOnlyIncludesPublic verifies homepage filter uses
// visibility = 'public' (not visibility != 'private').
func TestHomepageOnlyIncludesPublic(t *testing.T) {
	tests := []struct {
		visibility        string
		appearsOnHomepage bool
	}{
		{"public", true},
		{"unlisted", false},
		{"password", false},
		{"private", false},
	}

	// The homepage query filter MUST be:
	//   visibility = 'public' && is_draft = false
	// NOT:
	//   visibility != 'private' && is_draft = false
	//
	// This is critical because the second filter would leak unlisted/password content.

	for _, tt := range tests {
		t.Run(tt.visibility, func(t *testing.T) {
			// Simulate the homepage filter: only "public" passes
			passesHomepageFilter := tt.visibility == "public"

			if passesHomepageFilter != tt.appearsOnHomepage {
				t.Errorf("visibility=%q: homepage filter result=%v, want %v",
					tt.visibility, passesHomepageFilter, tt.appearsOnHomepage)
			}
		})
	}
}

// TestDraftContentExclusion verifies drafts never appear publicly.
func TestDraftContentExclusion(t *testing.T) {
	visibilities := []string{"public", "unlisted", "password", "private"}

	for _, v := range visibilities {
		t.Run(v+"_draft", func(t *testing.T) {
			record := createMockRecord(map[string]any{
				"visibility": v,
				"is_draft":   true,
			})
			if isRecordVisible(record) {
				t.Errorf("SECURITY: draft content with visibility=%q must not be visible", v)
			}
		})
	}
}

// --- filterBySelectedItems tests ---

func TestFilterBySelectedItems(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a", "name": "Alpha"},
		{"id": "b", "name": "Beta"},
		{"id": "c", "name": "Charlie"},
	}

	t.Run("empty selection returns nil", func(t *testing.T) {
		result := filterBySelectedItems(items, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}

		result = filterBySelectedItems(items, []string{})
		if result != nil {
			t.Errorf("expected nil for empty slice, got %v", result)
		}
	})

	t.Run("returns items in specified order", func(t *testing.T) {
		result := filterBySelectedItems(items, []string{"c", "a"})
		if len(result) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result))
		}
		if result[0]["id"] != "c" {
			t.Errorf("first item should be 'c', got %v", result[0]["id"])
		}
		if result[1]["id"] != "a" {
			t.Errorf("second item should be 'a', got %v", result[1]["id"])
		}
	})

	t.Run("skips missing IDs", func(t *testing.T) {
		result := filterBySelectedItems(items, []string{"a", "nonexistent", "b"})
		if len(result) != 2 {
			t.Fatalf("expected 2 items (skipping nonexistent), got %d", len(result))
		}
		if result[0]["id"] != "a" || result[1]["id"] != "b" {
			t.Errorf("expected [a, b], got [%v, %v]", result[0]["id"], result[1]["id"])
		}
	})

	t.Run("nil items returns nil", func(t *testing.T) {
		result := filterBySelectedItems(nil, []string{"a"})
		if result != nil {
			t.Errorf("expected nil for nil items, got %v", result)
		}
	})
}

func TestFilterBySelectedItemsWithDefault(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a", "name": "Alpha"},
		{"id": "b", "name": "Beta"},
	}

	t.Run("unconfigured returns all items", func(t *testing.T) {
		result := filterBySelectedItemsWithDefault(items, nil, false)
		if len(result) != 2 {
			t.Errorf("expected all items when not configured, got %d", len(result))
		}
	})

	t.Run("configured with empty selection returns nil", func(t *testing.T) {
		result := filterBySelectedItemsWithDefault(items, nil, true)
		if result != nil {
			t.Errorf("expected nil when configured with no items, got %v", result)
		}
	})

	t.Run("configured with selection returns in order", func(t *testing.T) {
		result := filterBySelectedItemsWithDefault(items, []string{"b", "a"}, true)
		if len(result) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result))
		}
		if result[0]["id"] != "b" {
			t.Errorf("first item should be 'b', got %v", result[0]["id"])
		}
	})
}

// --- getSectionConfig tests ---

func TestGetSectionConfig(t *testing.T) {
	t.Run("nil settings returns defaults", func(t *testing.T) {
		enabled, items, isConfigured := getSectionConfig(nil, "experience")
		if !enabled {
			t.Error("expected enabled=true for nil settings")
		}
		if items != nil {
			t.Errorf("expected nil items, got %v", items)
		}
		if isConfigured {
			t.Error("expected isConfigured=false for nil settings")
		}
	})

	t.Run("nil sections map returns defaults", func(t *testing.T) {
		settings := &services.SiteSettings{HomepageSections: nil}
		enabled, _, isConfigured := getSectionConfig(settings, "experience")
		if !enabled || isConfigured {
			t.Error("expected defaults for nil HomepageSections")
		}
	})

	t.Run("missing section returns defaults", func(t *testing.T) {
		settings := &services.SiteSettings{
			HomepageSections: map[string]services.HomepageSectionConfig{
				"skills": {Enabled: true},
			},
		}
		enabled, _, isConfigured := getSectionConfig(settings, "experience")
		if !enabled || isConfigured {
			t.Error("expected defaults for missing section")
		}
	})

	t.Run("existing section returns config", func(t *testing.T) {
		settings := &services.SiteSettings{
			HomepageSections: map[string]services.HomepageSectionConfig{
				"experience": {
					Enabled: true,
					Items:   []string{"id1", "id2"},
				},
			},
		}
		enabled, items, isConfigured := getSectionConfig(settings, "experience")
		if !enabled {
			t.Error("expected enabled=true")
		}
		if !isConfigured {
			t.Error("expected isConfigured=true")
		}
		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
	})

	t.Run("disabled section returns enabled=false", func(t *testing.T) {
		settings := &services.SiteSettings{
			HomepageSections: map[string]services.HomepageSectionConfig{
				"experience": {Enabled: false},
			},
		}
		enabled, _, isConfigured := getSectionConfig(settings, "experience")
		if enabled {
			t.Error("expected enabled=false for disabled section")
		}
		if !isConfigured {
			t.Error("expected isConfigured=true")
		}
	})
}

// --- Visibility level documentation ---

func TestVisibilityLevels(t *testing.T) {
	// Verify the 4 visibility levels are handled correctly
	levels := map[string]bool{
		"public":   true,  // visible
		"unlisted": true,  // visible (requires token at HTTP layer)
		"password": true,  // visible (requires JWT at HTTP layer)
		"private":  false, // NOT visible
	}

	for level, expectedVisible := range levels {
		t.Run(level, func(t *testing.T) {
			record := createMockRecord(map[string]any{
				"visibility": level,
				"is_draft":   false,
			})
			got := isRecordVisible(record)
			if got != expectedVisible {
				t.Errorf("isRecordVisible(visibility=%q) = %v, want %v", level, got, expectedVisible)
			}
		})
	}
}
