package hooks

import (
	"reflect"
	"testing"
)

func TestGetTopLanguagesSortsByBytesThenName(t *testing.T) {
	languages := map[string]int{
		"Shell":      100,
		"Go":         100,
		"TypeScript": 300,
		"Python":     300,
		"HTML":       200,
	}

	for i := 0; i < 100; i++ {
		got := getTopLanguages(languages, 4)
		want := []string{"Python", "TypeScript", "HTML", "Go"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getTopLanguages() = %v, want %v", got, want)
		}
	}

	if got := languages["Go"]; got != 100 {
		t.Fatalf("getTopLanguages changed Go byte count to %d", got)
	}
}

func TestGetTopLanguagesLimits(t *testing.T) {
	languages := map[string]int{"Go": 100, "Python": 300, "HTML": 200}

	for _, limit := range []int{0, -1} {
		if got := getTopLanguages(languages, limit); len(got) != 0 {
			t.Fatalf("getTopLanguages(limit %d) = %v, want empty", limit, got)
		}
	}

	got := getTopLanguages(languages, 10)
	want := []string{"Python", "HTML", "Go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("getTopLanguages(limit 10) = %v, want %v", got, want)
	}
}
