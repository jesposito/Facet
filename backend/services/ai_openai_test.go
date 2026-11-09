package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if got := body["reasoning_effort"]; got != "none" {
		t.Errorf("reasoning_effort = %v, want none", got)
	}
	if got := body["temperature"]; got != 0.7 {
		t.Errorf("temperature = %v, want 0.7 when reasoning is disabled", got)
	}
	if _, ok := body["max_tokens"]; ok {
		t.Errorf("max_tokens = %v, want omitted", body["max_tokens"])
	}
	if got := body["max_completion_tokens"]; got != float64(512) {
		t.Errorf("max_completion_tokens = %v, want 512", got)
	}
}

func TestOpenAIRequestUsesReasoningParametersForGPT5(t *testing.T) {
	body := captureOpenAIRequest(t, &AIProvider{Type: "openai", Model: "gpt-5.6-terra"})

	if _, ok := body["temperature"]; ok {
		t.Errorf("temperature = %v, want omitted for GPT-5 reasoning", body["temperature"])
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
		t.Errorf("max_completion_tokens = %v, want omitted for a custom non-reasoning model", body["max_completion_tokens"])
	}
}

func TestOpenAIRequestRejectsEmptyLengthLimitedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"choices":[{"message":{"content":""},"finish_reason":"length"}]}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	ai := &AIService{client: server.Client()}
	provider := &AIProvider{Type: "openai", BaseURL: server.URL}
	_, err := ai.ImproveContentWithTokens(context.Background(), provider, "Improve this", 512)
	if err == nil {
		t.Fatal("ImproveContentWithTokens() error = nil, want token-limit error")
	}
	if !strings.Contains(err.Error(), `finish_reason "length"`) {
		t.Fatalf("ImproveContentWithTokens() error = %q, want finish reason", err)
	}
}
