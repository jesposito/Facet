package hooks

import (
	"strings"
	"testing"
)

func TestAIWritingPromptsPreserveSourceLanguage(t *testing.T) {
	prompts := map[string]string{
		"improve": buildImprovementPrompt("description", "Ich baue zuverlässige Plattformen.", map[string]string{
			"role": "Softwareentwickler",
		}, "improve"),
		"generate": buildImprovementPrompt("summary", "", map[string]string{
			"role":    "Softwareentwickler",
			"company": "Mittelstand",
		}, "generate"),
		"rewrite":  buildRewritePrompt("Ich baue zuverlässige Plattformen.", "description", nil, "professional"),
		"critique": buildCritiquePrompt("Ich baue zuverlässige Plattformen.", "description", nil),
	}

	for name, prompt := range prompts {
		t.Run(name, func(t *testing.T) {
			for _, want := range []string{
				"Detect the primary language from the user's current content first.",
				"If current content is empty, use the primary language from the provided context.",
				"Keep the response in that same language. Do not translate to English",
				"English examples are illustrative only.",
			} {
				if !strings.Contains(prompt, want) {
					t.Fatalf("prompt missing language rule %q:\n%s", want, prompt)
				}
			}
		})
	}
}
