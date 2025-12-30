package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubService handles GitHub API interactions
type GitHubService struct {
	client *http.Client
}

// RepoMetadata represents GitHub repository metadata
type RepoMetadata struct {
	Name            string            `json:"name"`
	FullName        string            `json:"full_name"`
	Description     string            `json:"description"`
	Homepage        string            `json:"homepage"`
	HTMLURL         string            `json:"html_url"`
	CloneURL        string            `json:"clone_url"`
	StargazersCount int               `json:"stargazers_count"`
	ForksCount      int               `json:"forks_count"`
	OpenIssuesCount int               `json:"open_issues_count"`
	DefaultBranch   string            `json:"default_branch"`
	Topics          []string          `json:"topics"`
	Languages       map[string]int    `json:"languages"`
	License         *LicenseInfo      `json:"license"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	PushedAt        time.Time         `json:"pushed_at"`
	README          string            `json:"readme"`
	Owner           string            `json:"owner_login"`
}

// LicenseInfo represents license information
type LicenseInfo struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	SPDXID string `json:"spdx_id"`
}

type ghRepoResponse struct {
	Name            string       `json:"name"`
	FullName        string       `json:"full_name"`
	Description     string       `json:"description"`
	Homepage        string       `json:"homepage"`
	HTMLURL         string       `json:"html_url"`
	CloneURL        string       `json:"clone_url"`
	StargazersCount int          `json:"stargazers_count"`
	ForksCount      int          `json:"forks_count"`
	OpenIssuesCount int          `json:"open_issues_count"`
	DefaultBranch   string       `json:"default_branch"`
	Topics          []string     `json:"topics"`
	License         *LicenseInfo `json:"license"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	PushedAt        time.Time    `json:"pushed_at"`
	Owner           struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type ghReadmeResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// NewGitHubService creates a new GitHub service
func NewGitHubService() *GitHubService {
	return &GitHubService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchRepoMetadata fetches complete metadata for a GitHub repository
func (g *GitHubService) FetchRepoMetadata(owner, repo, token string) (*RepoMetadata, error) {
	// Fetch basic repo info
	repoData, err := g.fetchRepo(owner, repo, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo: %w", err)
	}

	// Fetch languages
	languages, err := g.fetchLanguages(owner, repo, token)
	if err != nil {
		// Non-fatal, continue without languages
		languages = make(map[string]int)
	}

	// Fetch README
	readme, err := g.fetchREADME(owner, repo, token)
	if err != nil {
		// Non-fatal, continue without README
		readme = ""
	}

	// Truncate README if too long (max 50KB)
	if len(readme) > 50000 {
		readme = readme[:50000] + "\n\n[README truncated...]"
	}

	return &RepoMetadata{
		Name:            repoData.Name,
		FullName:        repoData.FullName,
		Description:     repoData.Description,
		Homepage:        repoData.Homepage,
		HTMLURL:         repoData.HTMLURL,
		CloneURL:        repoData.CloneURL,
		StargazersCount: repoData.StargazersCount,
		ForksCount:      repoData.ForksCount,
		OpenIssuesCount: repoData.OpenIssuesCount,
		DefaultBranch:   repoData.DefaultBranch,
		Topics:          repoData.Topics,
		Languages:       languages,
		License:         repoData.License,
		CreatedAt:       repoData.CreatedAt,
		UpdatedAt:       repoData.UpdatedAt,
		PushedAt:        repoData.PushedAt,
		README:          readme,
		Owner:           repoData.Owner.Login,
	}, nil
}

func (g *GitHubService) fetchRepo(owner, repo, token string) (*ghRepoResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var repoResp ghRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return nil, err
	}

	return &repoResp, nil
}

func (g *GitHubService) fetchLanguages(owner, repo, token string) (map[string]int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/languages", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch languages: %d", resp.StatusCode)
	}

	var languages map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&languages); err != nil {
		return nil, err
	}

	return languages, nil
}

func (g *GitHubService) fetchREADME(owner, repo, token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // No README is fine
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch README: %d", resp.StatusCode)
	}

	var readmeResp ghReadmeResponse
	if err := json.NewDecoder(resp.Body).Decode(&readmeResp); err != nil {
		return "", err
	}

	// Decode base64 content
	if readmeResp.Encoding == "base64" {
		// Remove newlines from base64
		content := strings.ReplaceAll(readmeResp.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	return readmeResp.Content, nil
}

// ParseRepoURL parses a GitHub URL or owner/repo string
func (g *GitHubService) ParseRepoURL(input string) (owner, repo string, err error) {
	input = strings.TrimSpace(input)

	// Handle full URLs
	if strings.HasPrefix(input, "https://github.com/") {
		input = strings.TrimPrefix(input, "https://github.com/")
	} else if strings.HasPrefix(input, "http://github.com/") {
		input = strings.TrimPrefix(input, "http://github.com/")
	} else if strings.HasPrefix(input, "github.com/") {
		input = strings.TrimPrefix(input, "github.com/")
	}

	// Remove trailing .git
	input = strings.TrimSuffix(input, ".git")

	// Remove trailing slashes and paths
	parts := strings.Split(input, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository format: expected owner/repo")
	}

	owner = parts[0]
	repo = parts[1]

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository format: owner and repo cannot be empty")
	}

	return owner, repo, nil
}
