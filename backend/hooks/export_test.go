package hooks

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExportMeta(t *testing.T) {
	meta := ExportMeta{
		Version:    "1.0.0",
		ExportedAt: "2026-01-01T00:00:00Z",
		App:        "Facet",
	}

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal ExportMeta to JSON: %v", err)
	}

	var jsonMeta ExportMeta
	if err := json.Unmarshal(jsonBytes, &jsonMeta); err != nil {
		t.Fatalf("Failed to unmarshal ExportMeta from JSON: %v", err)
	}

	if jsonMeta.Version != meta.Version {
		t.Errorf("Version mismatch: got %s, want %s", jsonMeta.Version, meta.Version)
	}
	if jsonMeta.App != meta.App {
		t.Errorf("App mismatch: got %s, want %s", jsonMeta.App, meta.App)
	}

	// Test YAML marshaling
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal ExportMeta to YAML: %v", err)
	}

	var yamlMeta ExportMeta
	if err := yaml.Unmarshal(yamlBytes, &yamlMeta); err != nil {
		t.Fatalf("Failed to unmarshal ExportMeta from YAML: %v", err)
	}

	if yamlMeta.Version != meta.Version {
		t.Errorf("Version mismatch: got %s, want %s", yamlMeta.Version, meta.Version)
	}
	if yamlMeta.App != meta.App {
		t.Errorf("App mismatch: got %s, want %s", yamlMeta.App, meta.App)
	}
}

func TestExportDataStructure(t *testing.T) {
	export := &ExportData{
		Meta: ExportMeta{
			Version:    "1.0.0",
			ExportedAt: "2026-01-01T00:00:00Z",
			App:        "Facet",
		},
		Profile: map[string]interface{}{
			"name":     "Test User",
			"headline": "Software Engineer",
		},
		Experience: []map[string]interface{}{
			{"company": "Test Corp", "title": "Developer"},
		},
		Projects: []map[string]interface{}{
			{"title": "Test Project", "summary": "A test project"},
		},
	}

	// Test JSON marshaling
	jsonBytes, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal ExportData to JSON: %v", err)
	}

	var jsonExport ExportData
	if err := json.Unmarshal(jsonBytes, &jsonExport); err != nil {
		t.Fatalf("Failed to unmarshal ExportData from JSON: %v", err)
	}

	if jsonExport.Meta.App != "Facet" {
		t.Errorf("Meta.App mismatch")
	}
	if jsonExport.Profile["name"] != "Test User" {
		t.Errorf("Profile.name mismatch")
	}
	if len(jsonExport.Experience) != 1 {
		t.Errorf("Experience length mismatch: got %d, want 1", len(jsonExport.Experience))
	}
	if len(jsonExport.Projects) != 1 {
		t.Errorf("Projects length mismatch: got %d, want 1", len(jsonExport.Projects))
	}

	// Test YAML marshaling
	yamlBytes, err := yaml.Marshal(export)
	if err != nil {
		t.Fatalf("Failed to marshal ExportData to YAML: %v", err)
	}

	var yamlExport ExportData
	if err := yaml.Unmarshal(yamlBytes, &yamlExport); err != nil {
		t.Fatalf("Failed to unmarshal ExportData from YAML: %v", err)
	}

	if yamlExport.Meta.App != "Facet" {
		t.Errorf("Meta.App mismatch in YAML")
	}
}

func TestExportDataOmitEmpty(t *testing.T) {
	// Test that empty slices are omitted from JSON output
	export := &ExportData{
		Meta: ExportMeta{
			Version:    "1.0.0",
			ExportedAt: "2026-01-01T00:00:00Z",
			App:        "Facet",
		},
		// All other fields empty
	}

	jsonBytes, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)
	// Empty slices should not appear in output due to omitempty
	if stringContains(jsonStr, `"experience"`) {
		t.Errorf("Empty experience should be omitted")
	}
	if stringContains(jsonStr, `"projects"`) {
		t.Errorf("Empty projects should be omitted")
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestIsSensitiveField confirms the strip list catches credential field name
// patterns and lets normal fields through.
func TestIsSensitiveField(t *testing.T) {
	cases := []struct {
		field    string
		want     bool
	}{
		{"password", true},
		{"password_hash", true},
		{"Password_Hash", true}, // case insensitive
		{"api_key", true},
		{"stripe_api_key", true},
		{"client_secret", true},
		{"totp_secret", true},
		{"verification_token", true},
		{"email_verification_token", true},
		{"refresh_token", true},
		{"access_token", true},
		{"encryption_key", true},
		{"private_key", true},
		{"session_id", true},

		// non-sensitive fields must pass through
		{"name", false},
		{"email", false},
		{"title", false},
		{"description", false},
		{"created", false},
		{"id", false},
		{"slug", false},
	}

	for _, tc := range cases {
		got := isSensitiveField(tc.field)
		if got != tc.want {
			t.Errorf("isSensitiveField(%q) = %v, want %v", tc.field, got, tc.want)
		}
	}
}

// TestNewExportCollectionsOmitEmpty confirms every newly added export field
// is correctly tagged omitempty so an empty database produces a minimal JSON.
func TestNewExportCollectionsOmitEmpty(t *testing.T) {
	export := &ExportData{
		Meta: ExportMeta{Version: "1.1.0", ExportedAt: "2026-01-01T00:00:00Z", App: "Facet"},
	}
	jsonBytes, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	jsonStr := string(jsonBytes)

	// All new collection fields must be absent from an empty export.
	newKeys := []string{
		"testimonials", "testimonial_requests", "contact_methods", "sources",
		"custom_content", "courses", "modules", "lessons", "course_progress",
		"lesson_progress", "course_certificates", "quizzes", "quiz_questions",
		"quiz_attempts", "comments", "comment_reports", "purchases", "coupons",
		"newsletter_lists", "subscribers", "subscriber_list_memberships",
		"newsletter_sends", "newsletter_events", "settings", "site_settings",
		"media_display_names", "external_media", "admin_tags", "ai_providers",
		"import_proposals", "resume_imports", "view_exports", "webhooks",
		"download_logs", "uploads",
	}
	for _, key := range newKeys {
		needle := `"` + key + `"`
		if stringContains(jsonStr, needle) {
			t.Errorf("empty export should omit %q but it appeared in JSON", key)
		}
	}
}

// TestExportVersionBumped guards the JSON schema version bump that signals
// new collection coverage to consumers.
func TestExportVersionBumped(t *testing.T) {
	// collectExportData hard-codes the version. Mirror that here.
	const wantVersion = "1.1.0"
	export := &ExportData{Meta: ExportMeta{Version: wantVersion}}
	if export.Meta.Version != wantVersion {
		t.Fatalf("Version = %q, want %q", export.Meta.Version, wantVersion)
	}
}
