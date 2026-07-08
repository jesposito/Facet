package hooks

import (
	"testing"

	"facet/buildinfo"
)

func TestVersionIdentityPayload(t *testing.T) {
	originalVersion := buildinfo.Version
	originalSHA := buildinfo.SHA
	originalBuildDate := buildinfo.BuildDate
	defer func() {
		buildinfo.Version = originalVersion
		buildinfo.SHA = originalSHA
		buildinfo.BuildDate = originalBuildDate
	}()

	buildinfo.Version = "v1.2.3"
	buildinfo.SHA = "abcdef1234567890"
	buildinfo.BuildDate = "2026-07-08T10:09:00Z"

	t.Setenv("FACET_VERSION", "v1.2.3")
	t.Setenv("BUILD_DATE", "2026-07-08T10:09:00Z")
	t.Setenv("GIT_COMMIT", "abcdef1234567890")

	got := versionIdentityPayload()

	tests := map[string]string{
		"version":        "v1.2.3",
		"sha":            "abcdef1",
		"build_date":     "2026-07-08T10:09:00Z",
		"env":            "v1.2.3",
		"env_version":    "v1.2.3",
		"env_build_date": "2026-07-08T10:09:00Z",
		"env_git_commit": "abcdef1234567890",
	}

	for key, want := range tests {
		if got[key] != want {
			t.Fatalf("payload[%q] = %q, want %q", key, got[key], want)
		}
	}
}
