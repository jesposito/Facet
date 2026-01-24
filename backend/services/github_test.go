package services

import (
	"net/http"
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

func TestFetchRepoMetadata_Success(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	mock.Handle("GET", "/repos/owner/repo", mock.JSON(http.StatusOK, GitHubRepoResponse("owner", "repo", "A test repository")))
	mock.Handle("GET", "/repos/owner/repo/languages", mock.JSON(http.StatusOK, GitHubLanguagesResponse(nil)))
	mock.Handle("GET", "/repos/owner/repo/readme", mock.JSON(http.StatusOK, GitHubReadmeResponse("# Test\nThis is a test.")))

	gh := NewMockGitHubService(mock.URL)

	meta, err := gh.FetchRepoMetadata("owner", "repo", "test-token")
	if err != nil {
		t.Fatalf("FetchRepoMetadata() error = %v", err)
	}

	if meta.Name != "repo" {
		t.Errorf("Expected name 'repo', got %q", meta.Name)
	}
	if meta.FullName != "owner/repo" {
		t.Errorf("Expected full_name 'owner/repo', got %q", meta.FullName)
	}
	if meta.Description != "A test repository" {
		t.Errorf("Expected description 'A test repository', got %q", meta.Description)
	}
	if meta.StargazersCount != 100 {
		t.Errorf("Expected 100 stars, got %d", meta.StargazersCount)
	}
}

func TestFetchRepoMetadata_RepoNotFound(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	mock.Handle("GET", "/repos/owner/nonexistent", mock.Error(http.StatusNotFound, "Not Found"))

	gh := NewMockGitHubService(mock.URL)

	_, err := gh.FetchRepoMetadata("owner", "nonexistent", "")
	if err == nil {
		t.Fatal("Expected error for non-existent repo")
	}
}

func TestFetchRepoMetadata_NoReadme(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	mock.Handle("GET", "/repos/owner/repo", mock.JSON(http.StatusOK, GitHubRepoResponse("owner", "repo", "No readme here")))
	mock.Handle("GET", "/repos/owner/repo/languages", mock.JSON(http.StatusOK, GitHubLanguagesResponse(map[string]int{"Python": 50000})))
	mock.Handle("GET", "/repos/owner/repo/readme", mock.Error(http.StatusNotFound, "Not Found"))

	gh := NewMockGitHubService(mock.URL)

	meta, err := gh.FetchRepoMetadata("owner", "repo", "")
	if err != nil {
		t.Fatalf("FetchRepoMetadata() should not fail without README, got error = %v", err)
	}

	if meta.README != "" {
		t.Errorf("Expected empty README, got %q", meta.README)
	}
	if meta.Languages["Python"] != 50000 {
		t.Errorf("Expected Python: 50000, got %v", meta.Languages)
	}
}

func TestFetchRepoMetadata_Unauthorized(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	mock.Handle("GET", "/repos/private/repo", mock.Error(http.StatusUnauthorized, "Bad credentials"))

	gh := NewMockGitHubService(mock.URL)

	_, err := gh.FetchRepoMetadata("private", "repo", "invalid-token")
	if err == nil {
		t.Fatal("Expected error for unauthorized request")
	}
}

func TestFetchRepoMetadata_RateLimited(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	mock.Handle("GET", "/repos/owner/repo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1234567890")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	})

	gh := NewMockGitHubService(mock.URL)

	_, err := gh.FetchRepoMetadata("owner", "repo", "")
	if err == nil {
		t.Fatal("Expected error for rate limited request")
	}
}
