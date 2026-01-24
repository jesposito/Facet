package services

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestParseEnrichmentResponse(t *testing.T) {
	ai := &AIService{}

	tests := []struct {
		name     string
		response string
		wantErr  bool
		check    func(*EnrichmentResult) bool
	}{
		{
			name:     "plain JSON",
			response: `{"summary": "A test project", "bullets": ["Feature 1", "Feature 2"], "tags": ["go", "test"], "case_study": "Built for testing", "tech_highlights": ["Go", "Testing"]}`,
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "A test project" &&
					len(r.Bullets) == 2 &&
					len(r.Tags) == 2 &&
					r.CaseStudy == "Built for testing" &&
					len(r.TechHighlights) == 2
			},
		},
		{
			name:     "markdown json block",
			response: "```json\n" + `{"summary": "Markdown test", "bullets": ["One"], "tags": ["md"]}` + "\n```",
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "Markdown test" && len(r.Bullets) == 1
			},
		},
		{
			name:     "plain markdown block",
			response: "```\n" + `{"summary": "Plain block", "bullets": []}` + "\n```",
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "Plain block"
			},
		},
		{
			name:     "JSON with leading text",
			response: `Here is the response: {"summary": "With text", "bullets": []}`,
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "With text"
			},
		},
		{
			name:     "JSON with trailing text",
			response: `{"summary": "Trailing", "bullets": []} That's the output.`,
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "Trailing"
			},
		},
		{
			name:     "case_study as array",
			response: `{"summary": "Array case study", "bullets": [], "case_study": ["Point 1", "Point 2"]}`,
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "Array case study" &&
					r.CaseStudy == "• Point 1\n• Point 2"
			},
		},
		{
			name:     "missing fields",
			response: `{"summary": "Minimal"}`,
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "Minimal" &&
					len(r.Bullets) == 0 &&
					len(r.Tags) == 0
			},
		},
		{
			name:     "empty JSON object",
			response: `{}`,
			wantErr:  false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "" && len(r.Bullets) == 0
			},
		},
		{
			name:     "invalid JSON",
			response: `{invalid json`,
			wantErr:  true,
			check:    nil,
		},
		{
			name:     "no JSON at all",
			response: `This is just plain text with no JSON`,
			wantErr:  true,
			check:    nil,
		},
		{
			name: "whitespace around JSON",
			response: `

   {"summary": "Whitespace", "bullets": []}

`,
			wantErr: false,
			check: func(r *EnrichmentResult) bool {
				return r.Summary == "Whitespace"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ai.parseEnrichmentResponse(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEnrichmentResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && !tt.check(result) {
				t.Errorf("parseEnrichmentResponse() result check failed for %s", tt.name)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	ai := &AIService{}

	tests := []struct {
		name    string
		req     *EnrichmentRequest
		wantLen int
	}{
		{
			name: "basic request",
			req: &EnrichmentRequest{
				Title:       "Test Project",
				Description: "A test project",
				README:      "# Test\nThis is a test.",
				Languages:   map[string]int{"Go": 80, "Shell": 20},
				Topics:      []string{"testing", "go"},
				PrivacyMode: "full",
			},
			wantLen: 100,
		},
		{
			name: "empty readme",
			req: &EnrichmentRequest{
				Title:       "No README",
				Description: "Project without README",
				README:      "",
				Languages:   nil,
				Topics:      nil,
				PrivacyMode: "summary",
			},
			wantLen: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ai.buildPrompt(tt.req)
			if len(result) < tt.wantLen {
				t.Errorf("buildPrompt() length = %d, want at least %d", len(result), tt.wantLen)
			}
			if tt.req.Title != "" && !contains(result, tt.req.Title) {
				t.Errorf("buildPrompt() should contain title %q", tt.req.Title)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestEnrichProject_OpenAI(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	enrichmentJSON := `{"summary": "Test project", "bullets": ["Feature 1"], "tags": ["go"]}`
	mock.Handle("POST", "/chat/completions", mock.JSON(http.StatusOK, OpenAIResponse(enrichmentJSON)))

	ai := &AIService{client: &http.Client{Timeout: 5 * time.Second}}
	provider := MockAIProvider(mock.URL, "openai")

	result, err := ai.EnrichProject(context.Background(), provider, &EnrichmentRequest{
		Title:       "Test",
		Description: "A test project",
		PrivacyMode: "full",
	})

	if err != nil {
		t.Fatalf("EnrichProject() error = %v", err)
	}
	if result.Summary != "Test project" {
		t.Errorf("Expected summary 'Test project', got %q", result.Summary)
	}
	if len(result.Bullets) != 1 {
		t.Errorf("Expected 1 bullet, got %d", len(result.Bullets))
	}
}

func TestEnrichProject_OpenAI_APIError(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	mock.Handle("POST", "/chat/completions", mock.Error(http.StatusUnauthorized, "Invalid API key"))

	ai := &AIService{client: &http.Client{Timeout: 5 * time.Second}}
	provider := MockAIProvider(mock.URL, "openai")

	_, err := ai.EnrichProject(context.Background(), provider, &EnrichmentRequest{
		Title: "Test",
	})

	if err == nil {
		t.Fatal("Expected error for unauthorized request")
	}
}

func TestEnrichProject_Anthropic(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	enrichmentJSON := `{"summary": "Anthropic test", "bullets": ["AI powered"], "tags": ["claude"]}`
	mock.Handle("POST", "/messages", mock.JSON(http.StatusOK, AnthropicResponse(enrichmentJSON)))

	ai := &AIService{client: &http.Client{Timeout: 5 * time.Second}}
	provider := MockAIProvider(mock.URL, "anthropic")

	result, err := ai.EnrichProject(context.Background(), provider, &EnrichmentRequest{
		Title:       "Claude Test",
		Description: "Testing Anthropic integration",
		PrivacyMode: "summary",
	})

	if err != nil {
		t.Fatalf("EnrichProject() error = %v", err)
	}
	if result.Summary != "Anthropic test" {
		t.Errorf("Expected summary 'Anthropic test', got %q", result.Summary)
	}
}

func TestEnrichProject_Ollama(t *testing.T) {
	mock := NewMockServer(t)
	defer mock.Server.Close()

	enrichmentJSON := `{"summary": "Local model test", "bullets": ["Self-hosted"], "tags": ["ollama"]}`
	mock.Handle("POST", "/api/generate", mock.JSON(http.StatusOK, OllamaResponse(enrichmentJSON)))

	ai := &AIService{client: &http.Client{Timeout: 5 * time.Second}}
	provider := MockAIProvider(mock.URL, "ollama")

	result, err := ai.EnrichProject(context.Background(), provider, &EnrichmentRequest{
		Title:       "Ollama Test",
		Description: "Testing local model",
		PrivacyMode: "none",
	})

	if err != nil {
		t.Fatalf("EnrichProject() error = %v", err)
	}
	if result.Summary != "Local model test" {
		t.Errorf("Expected summary 'Local model test', got %q", result.Summary)
	}
}

func TestEnrichProject_UnsupportedProvider(t *testing.T) {
	ai := &AIService{client: &http.Client{}}
	provider := &AIProvider{Type: "unsupported"}

	_, err := ai.EnrichProject(context.Background(), provider, &EnrichmentRequest{Title: "Test"})

	if err == nil {
		t.Fatal("Expected error for unsupported provider")
	}
}
