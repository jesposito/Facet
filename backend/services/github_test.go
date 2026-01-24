package services

import (
	"testing"
)

func TestParseRepoURL(t *testing.T) {
	g := NewGitHubService()

	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https URL",
			input:     "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "http URL",
			input:     "http://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL without protocol",
			input:     "github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "owner/repo format",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with .git suffix",
			input:     "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with extra path segments",
			input:     "https://github.com/owner/repo/tree/main/src",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with trailing slash",
			input:     "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "whitespace around input",
			input:     "  owner/repo  ",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "real repo - facebook/react",
			input:     "https://github.com/facebook/react",
			wantOwner: "facebook",
			wantRepo:  "react",
			wantErr:   false,
		},
		{
			name:      "just owner no repo",
			input:     "owner",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "empty string",
			input:     "",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "only slash",
			input:     "/",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "empty owner",
			input:     "/repo",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "empty repo",
			input:     "owner/",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := g.ParseRepoURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepoURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if owner != tt.wantOwner {
				t.Errorf("ParseRepoURL(%q) owner = %q, want %q", tt.input, owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("ParseRepoURL(%q) repo = %q, want %q", tt.input, repo, tt.wantRepo)
			}
		})
	}
}
