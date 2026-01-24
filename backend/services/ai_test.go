package services

import (
	"testing"
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
