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

// Small models ignore the passive rules above, so non-English content must
// additionally pin an explicit directive to the first and last lines.
func TestAIWritingPromptsPinExplicitLanguageDirective(t *testing.T) {
	german := "Ich entwickle zuverlässige Softwareplattformen für mittelständische Unternehmen und betreue deren Betrieb."

	prompts := map[string]string{
		"improve":  buildImprovementPrompt("description", german, nil, "improve"),
		"rewrite":  buildRewritePrompt(german, "description", nil, "professional"),
		"critique": buildCritiquePrompt(german, "description", nil),
	}

	for name, prompt := range prompts {
		t.Run(name, func(t *testing.T) {
			if !strings.HasPrefix(prompt, "CRITICAL LANGUAGE REQUIREMENT: The source content is written in German.") {
				t.Fatalf("prompt must START with the German directive:\n%s", prompt[:200])
			}
			tail := prompt[len(prompt)-160:]
			if !strings.Contains(tail, "German") {
				t.Fatalf("prompt must END with a German reminder, got tail:\n%s", tail)
			}
		})
	}

	// English content: no directive, prompts unchanged.
	english := buildImprovementPrompt("description", "I build reliable software platforms for mid-sized companies.", nil, "improve")
	if strings.Contains(english, "CRITICAL LANGUAGE REQUIREMENT") {
		t.Fatal("English content must not get a language directive")
	}
}
