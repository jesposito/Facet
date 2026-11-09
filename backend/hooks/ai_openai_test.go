package hooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFetchOpenAIModelsPrefersGPT6(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("request = %s %s, want GET /v1/models", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "gpt-4o-mini", "owned_by": "openai"},
				{"id": "gpt-6-luna", "owned_by": "openai"},
				{"id": "gpt-6.1-terra", "owned_by": "openai"},
				{"id": "text-embedding-3-small", "owned_by": "openai"},
			},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	models, err := fetchOpenAIModels(server.Client(), "test-key", server.URL+"/v1")
	if err != nil {
		t.Fatalf("fetchOpenAIModels() error = %v", err)
	}
	want := []string{"gpt-6-luna", "gpt-6.1-terra", "gpt-4o-mini"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("fetchOpenAIModels() = %v, want %v", models, want)
	}
}
