package hooks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"facet/services"
)

type cancellationEnricher struct {
	started chan struct{}
}

func (a *cancellationEnricher) DecryptAPIKey(string) (string, error) {
	return "sk-provider-secret", nil
}

func (a *cancellationEnricher) EnrichProject(ctx context.Context, _ *services.AIProvider, _ *services.EnrichmentRequest) (*services.EnrichmentResult, error) {
	close(a.started)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return nil, errors.New("provider context was not canceled")
	}
}

func TestGitHubImportCancellationCreatesNoRecords(t *testing.T) {
	enricher := &cancellationEnricher{started: make(chan struct{})}
	f, _, h := newGitHubImportFixture(t, enricher)
	_, adminToken := f.createAdmin(t, "owner@example.com", "password123", nil)
	provider := f.create(t, "ai_providers", map[string]any{"name": "Test", "type": "openai", "api_key_encrypted": "enc", "is_active": true})

	ctx, cancel := context.WithCancel(context.Background())
	body := `{"repo_url":"octo/demo","ai_enrich":true,"ai_provider_id":"` + provider.Id + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/github/import", strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminToken)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-enricher.started:
	case <-time.After(2 * time.Second):
		t.Fatal("AI enrichment did not start")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("canceled GitHub import did not return")
	}
	if rec.Code != http.StatusRequestTimeout {
		t.Fatalf("canceled import status = %d, want %d; body: %s", rec.Code, http.StatusRequestTimeout, rec.Body.String())
	}
	for _, collection := range []string{"sources", "import_proposals"} {
		records, err := f.app.FindAllRecords(collection)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 0 {
			t.Fatalf("canceled import created %d %s records", len(records), collection)
		}
	}
}
