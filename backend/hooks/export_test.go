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
