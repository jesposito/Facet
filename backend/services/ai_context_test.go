package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEnrichProjectCancellationReachesProvider(t *testing.T) {
	for _, providerType := range []string{"openai", "anthropic", "ollama"} {
		t.Run(providerType, func(t *testing.T) {
			started := make(chan struct{})
			providerCanceled := make(chan struct{})
			stopHandler := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// HTTP/1 only watches for a disconnected client after reading
				// the request body, as a real provider would before processing it.
				_, _ = io.Copy(io.Discard, r.Body)
				close(started)
				select {
				case <-r.Context().Done():
					close(providerCanceled)
				case <-stopHandler:
				}
			}))
			defer server.Close()
			defer close(stopHandler)

			ai := &AIService{client: server.Client()}
			provider := &AIProvider{Type: providerType, BaseURL: server.URL, Model: "test"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := ai.EnrichProject(ctx, provider, &EnrichmentRequest{Title: "Test"})
				done <- err
			}()

			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("provider request did not start")
			}
			cancel()

			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("EnrichProject() error = %v, want context.Canceled", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("EnrichProject did not return after cancellation")
			}

			select {
			case <-providerCanceled:
			case <-time.After(2 * time.Second):
				t.Fatal("provider HTTP request context was not canceled")
			}
		})
	}
}
