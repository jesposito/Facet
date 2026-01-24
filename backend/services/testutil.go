package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockServer wraps httptest.Server with convenient handler registration and response helpers.
type MockServer struct {
	*httptest.Server
	t        *testing.T
	handlers map[string]http.HandlerFunc
}

func NewMockServer(t *testing.T) *MockServer {
	t.Helper()
	m := &MockServer{
		t:        t,
		handlers: make(map[string]http.HandlerFunc),
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if handler, ok := m.handlers[key]; ok {
			handler(w, r)
			return
		}
		if handler, ok := m.handlers[r.URL.Path]; ok {
			handler(w, r)
			return
		}
		t.Logf("MockServer: no handler for %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))

	return m
}

func (m *MockServer) Handle(method, path string, handler http.HandlerFunc) {
	key := method + " " + path
	m.handlers[key] = handler
}

func (m *MockServer) HandlePath(path string, handler http.HandlerFunc) {
	m.handlers[path] = handler
}

func (m *MockServer) JSON(statusCode int, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(data); err != nil {
			m.t.Fatalf("MockServer: failed to encode JSON: %v", err)
		}
	}
}

func (m *MockServer) JSONString(statusCode int, jsonStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(jsonStr))
	}
}

func (m *MockServer) Error(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, message, statusCode)
	}
}

func OpenAIResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   "gpt-4",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
		},
	}
}

func AnthropicResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"id":            "msg_test",
		"type":          "message",
		"role":          "assistant",
		"model":         "claude-3-sonnet",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  10,
			"output_tokens": 20,
		},
	}
}

func OllamaResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"model":                "llama2",
		"created_at":           "2024-01-01T00:00:00Z",
		"response":             content,
		"done":                 true,
		"done_reason":          "stop",
		"total_duration":       1000000000,
		"load_duration":        100000000,
		"prompt_eval_count":    10,
		"prompt_eval_duration": 100000000,
		"eval_count":           20,
		"eval_duration":        800000000,
	}
}

func GitHubRepoResponse(owner, name, description string) map[string]interface{} {
	return map[string]interface{}{
		"name":              name,
		"full_name":         owner + "/" + name,
		"description":       description,
		"homepage":          "",
		"html_url":          "https://github.com/" + owner + "/" + name,
		"clone_url":         "https://github.com/" + owner + "/" + name + ".git",
		"stargazers_count":  100,
		"forks_count":       10,
		"open_issues_count": 5,
		"default_branch":    "main",
		"topics":            []string{"go", "testing"},
		"license": map[string]interface{}{
			"key":     "mit",
			"name":    "MIT License",
			"spdx_id": "MIT",
		},
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-15T00:00:00Z",
		"pushed_at":  "2024-01-15T00:00:00Z",
		"owner": map[string]interface{}{
			"login": owner,
		},
	}
}

func GitHubLanguagesResponse(languages map[string]int) map[string]int {
	if languages == nil {
		return map[string]int{"Go": 80000, "Shell": 5000}
	}
	return languages
}

func GitHubReadmeResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"name":     "README.md",
		"path":     "README.md",
		"sha":      "abc123",
		"size":     len(content),
		"encoding": "base64",
		"content":  base64Encode(content),
	}
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func NewMockAIService(mockURL string) *AIService {
	return &AIService{
		crypto: nil,
		client: &http.Client{},
	}
}

func NewMockGitHubService(mockURL string) *GitHubService {
	return &GitHubService{
		client:  &http.Client{},
		baseURL: mockURL,
	}
}

func MockAIProvider(mockURL, providerType string) *AIProvider {
	return &AIProvider{
		ID:       "test-provider",
		Name:     "Test Provider",
		Type:     providerType,
		APIKey:   "test-api-key",
		BaseURL:  mockURL,
		Model:    "test-model",
		IsActive: true,
	}
}
