package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func captureOpenAIRequest(t *testing.T, provider *AIProvider) map[string]any {
	t.Helper()

	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("request = %s %s, want POST /chat/completions", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(OpenAIResponse("OK")); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	provider.BaseURL = server.URL
	ai := &AIService{client: server.Client()}
	result, err := ai.ImproveContentWithTokens(context.Background(), provider, "Improve this", 512)
	if err != nil {
		t.Fatalf("ImproveContentWithTokens() error = %v", err)
	}
	if result != "OK" {
		t.Fatalf("ImproveContentWithTokens() = %q, want OK", result)
	}
	return body
}

func TestOpenAIRequestUsesReasoningParametersForGPT6(t *testing.T) {
	body := captureOpenAIRequest(t, &AIProvider{Type: "openai"})

	if got := body["model"]; got != "gpt-6-luna" {
		t.Errorf("model = %v, want gpt-6-luna", got)
	}
	if _, ok := body["temperature"]; ok {
		t.Errorf("temperature = %v, want omitted for GPT-6 reasoning", body["temperature"])
	}
	if _, ok := body["max_tokens"]; ok {
		t.Errorf("max_tokens = %v, want omitted", body["max_tokens"])
	}
	if got := body["max_completion_tokens"]; got != float64(512) {
		t.Errorf("max_completion_tokens = %v, want 512", got)
	}
}

func TestOpenAIRequestKeepsLegacyParametersForCustomModels(t *testing.T) {
	body := captureOpenAIRequest(t, &AIProvider{Type: "custom", Model: "custom-chat"})

	if got := body["model"]; got != "custom-chat" {
		t.Errorf("model = %v, want custom-chat", got)
	}
	if got := body["temperature"]; got != 0.7 {
		t.Errorf("temperature = %v, want 0.7", got)
	}
	if got := body["max_tokens"]; got != float64(512) {
		t.Errorf("max_tokens = %v, want 512", got)
	}
	if _, ok := body["max_completion_tokens"]; ok {
		t.Errorf("max_completion_tokens = %v, want omitted for a custom non-GPT-6 model", body["max_completion_tokens"])
	}
}
