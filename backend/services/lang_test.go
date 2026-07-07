package services

import (
	"strings"
	"testing"
)

func TestDetectContentLanguage(t *testing.T) {
	german := "Ich entwickle zuverlässige Softwareplattformen für mittelständische Unternehmen und betreue deren Betrieb."
	english := "I build reliable software platforms for mid-sized companies and run their operations."

	tests := []struct {
		name  string
		texts []string
		want  string
	}{
		{"german content", []string{german}, "German"},
		{"english content", []string{english}, ""},
		{"empty", []string{""}, ""},
		{"too short", []string{"Hallo Welt"}, ""},
		{"content wins over context", []string{german, english}, "German"},
		{"english content stops fallback", []string{english, german}, ""},
		{"falls back to context when content empty", []string{"", german}, "German"},
		{"french", []string{"Je développe des plateformes logicielles fiables pour les entreprises de taille moyenne."}, "French"},
		{"spanish", []string{"Desarrollo plataformas de software confiables para empresas medianas y gestiono sus operaciones."}, "Spanish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectContentLanguage(tt.texts...); got != tt.want {
				t.Fatalf("DetectContentLanguage(%q) = %q, want %q", tt.texts, got, tt.want)
			}
		})
	}
}

func TestLanguagePromptHelpers(t *testing.T) {
	if LanguagePromptDirective("") != "" || LanguagePromptReminder("") != "" {
		t.Fatal("empty lang must produce empty directive and reminder")
	}
	d := LanguagePromptDirective("German")
	if !strings.Contains(d, "German") || !strings.Contains(d, "ENTIRE response") {
		t.Fatalf("directive malformed: %q", d)
	}
	if r := LanguagePromptReminder("German"); r != " Respond ONLY in German." {
		t.Fatalf("reminder malformed: %q", r)
	}
}
