package hooks

import (
	"reflect"
	"testing"
)

func TestOllamaTagURLs(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    []string
	}{
		{
			name:    "explicit base URL",
			baseURL: "http://example.local:11434/",
			want:    []string{"http://example.local:11434/api/tags"},
		},
		{
			name:    "docker fallback",
			baseURL: "",
			want: []string{
				"http://localhost:11434/api/tags",
				"http://ollama:11434/api/tags",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ollamaTagURLs(tt.baseURL); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ollamaTagURLs(%q) = %v, want %v", tt.baseURL, got, tt.want)
			}
		})
	}
}
