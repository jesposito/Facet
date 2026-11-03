package hooks

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"facet/services"
)

// fakeGitHub records the token each request used and returns fixed metadata,
// so these tests never reach api.github.com.
type fakeGitHub struct {
	mu     sync.Mutex
	tokens []string
}

func (g *fakeGitHub) ParseRepoURL(input string) (string, string, error) {
	return services.NewGitHubService().ParseRepoURL(input)
}

func (g *fakeGitHub) FetchRepoMetadata(owner, repo, token string) (*services.RepoMetadata, error) {
	g.mu.Lock()
	g.tokens = append(g.tokens, token)
	g.mu.Unlock()
	return &services.RepoMetadata{
		Name:        repo,
		FullName:    owner + "/" + repo,
		Description: "From GitHub",
		HTMLURL:     "https://github.com/" + owner + "/" + repo,
		Languages:   map[string]int{"Go": 100},
	}, nil
}

func (g *fakeGitHub) lastToken(t *testing.T) string {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.tokens) == 0 {
		t.Fatal("GitHub was not called")
	}
	return g.tokens[len(g.tokens)-1]
}

// fakeEnricher fails with an error that mentions the provider key, the kind of
// detail a real provider error can carry and the response must not repeat.
type fakeEnricher struct {
	err    error
	result *services.EnrichmentResult
}

func (a *fakeEnricher) DecryptAPIKey(string) (string, error) { return "sk-provider-secret", nil }

func (a *fakeEnricher) EnrichProject(context.Context, *services.AIProvider, *services.EnrichmentRequest) (*services.EnrichmentResult, error) {
	return a.result, a.err
}

func newGitHubImportFixture(t *testing.T, ai projectEnricher) (*securityFixture, *fakeGitHub, http.Handler) {
	t.Helper()
	f := newSecurityFixture(t)
	gh := &fakeGitHub{}
	registerGitHubRoutes(f.app, gh, ai, f.crypto)
	return f, gh, f.serve(t)
}

func saveGitHubToken(t *testing.T, f *securityFixture, token string) {
	t.Helper()
	enc, err := f.crypto.Encrypt(token)
	if err != nil {
		t.Fatal(err)
	}
	f.create(t, "settings", map[string]any{"key": "github_token", "value": map[string]string{"token_encrypted": enc}})
}

func TestGitHubRoutesRequireAdmin(t *testing.T) {
	f, gh, h := newGitHubImportFixture(t, &fakeEnricher{})
	buyerJWT, _, err := f.purchase.GeneratePurchaseJWT([]string{"purchase1"}, "buyer@example.com")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"repo_url":"octo/demo"}`
	for _, path := range []string{"/api/github/preview", "/api/github/import", "/api/github/refresh/missing"} {
		for name, headers := range map[string]map[string]string{
			"anonymous":     nil,
			"buyer bearer":  {"Authorization": "Bearer " + buyerJWT},
			"buyer raw":     {"Authorization": buyerJWT},
			"buyer cookie":  {"Cookie": services.PurchaseCookieName + "=" + buyerJWT},
			"garbage token": {"Authorization": "Bearer not-a-token"},
		} {
			rec := do(t, h, testRequest{method: http.MethodPost, url: path, body: body, headers: headers})
			if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s: status = %d, want 401/403; body: %s", name, path, rec.Code, rec.Body.String())
			}
		}
	}
	if len(gh.tokens) != 0 {
		t.Fatalf("unauthenticated requests reached GitHub %d times", len(gh.tokens))
	}
}

func TestGitHubRequestTokenTakesPrecedenceAndIsNotStored(t *testing.T) {
	f, gh, h := newGitHubImportFixture(t, &fakeEnricher{})
	_, adminToken := f.createAdmin(t, "owner@example.com", "password123", nil)
	saveGitHubToken(t, f, "ghp_saved")
	auth := map[string]string{"Authorization": adminToken}

	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/github/preview", body: `{"repo_url":"octo/demo","token":"ghp_once"}`, headers: auth})
	assertStatus(t, rec, http.StatusOK)
	if got := gh.lastToken(t); got != "ghp_once" {
		t.Fatalf("preview used token %q, want the one-import token", got)
	}

	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/github/preview", body: `{"repo_url":"octo/demo","token":"  "}`, headers: auth})
	assertStatus(t, rec, http.StatusOK)
	if got := gh.lastToken(t); got != "ghp_saved" {
		t.Fatalf("blank request token used %q, want the saved token", got)
	}

	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/github/import", body: `{"repo_url":"octo/demo","token":"ghp_once"}`, headers: auth})
	assertStatus(t, rec, http.StatusOK)
	assertNotContains(t, rec, "ghp_once")
	assertNotContains(t, rec, "ai_error")
	if got := gh.lastToken(t); got != "ghp_once" {
		t.Fatalf("import used token %q, want the one-import token", got)
	}

	// The one-import token must not land in any record the import touched.
	for _, collection := range []string{"settings", "sources", "import_proposals"} {
		records, err := f.app.FindAllRecords(collection)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range records {
			for k, v := range r.FieldsData() {
				if s, ok := v.(string); ok && strings.Contains(s, "ghp_once") {
					t.Fatalf("%s.%s stored the one-import token", collection, k)
				}
			}
			if raw := r.GetString("value"); strings.Contains(raw, "ghp_once") {
				t.Fatalf("%s.value stored the one-import token", collection)
			}
		}
	}
	saved := getGitHubToken(f.app, f.crypto)
	if saved != "ghp_saved" {
		t.Fatalf("saved token changed to %q", saved)
	}
}

func TestGitHubImportReportsAIFailure(t *testing.T) {
	providerErr := errors.New("401 from provider: invalid key sk-provider-secret")
	f, _, h := newGitHubImportFixture(t, &fakeEnricher{err: providerErr})
	_, adminToken := f.createAdmin(t, "owner@example.com", "password123", nil)
	provider := f.create(t, "ai_providers", map[string]any{"name": "Test", "type": "openai", "api_key_encrypted": "enc", "is_active": true})
	auth := map[string]string{"Authorization": adminToken}

	rec := do(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/github/import",
		body:    `{"repo_url":"octo/demo","ai_enrich":true,"ai_provider_id":"` + provider.Id + `","privacy_mode":"none"}`,
		headers: auth,
	})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"ai_enriched":false`)
	assertContains(t, rec, `"ai_error":"`+githubAIError+`"`)
	assertNotContains(t, rec, "sk-provider-secret")
	assertContains(t, rec, `"summary":"From GitHub"`)

	// A provider that no longer exists is also a failed enrichment, not a
	// silent success.
	rec = do(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/github/import",
		body:    `{"repo_url":"octo/demo","ai_enrich":true,"ai_provider_id":"missingprovider"}`,
		headers: auth,
	})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"ai_error"`)

	// Without enrichment requested there's nothing to warn about.
	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/github/import", body: `{"repo_url":"octo/demo"}`, headers: auth})
	assertStatus(t, rec, http.StatusOK)
	assertNotContains(t, rec, "ai_error")
}

func TestGitHubImportAppliesSuccessfulEnrichment(t *testing.T) {
	f, _, h := newGitHubImportFixture(t, &fakeEnricher{result: &services.EnrichmentResult{Summary: "Polished summary"}})
	_, adminToken := f.createAdmin(t, "owner@example.com", "password123", nil)
	provider := f.create(t, "ai_providers", map[string]any{"name": "Test", "type": "openai", "api_key_encrypted": "enc", "is_active": true})

	rec := do(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/github/import",
		body:    `{"repo_url":"octo/demo","ai_enrich":true,"ai_provider_id":"` + provider.Id + `"}`,
		headers: map[string]string{"Authorization": adminToken},
	})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"ai_enriched":true`)
	assertContains(t, rec, "Polished summary")
	assertNotContains(t, rec, "ai_error")
}
