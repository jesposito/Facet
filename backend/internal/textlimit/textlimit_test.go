package textlimit

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLimits(t *testing.T) {
	for _, input := range []string{"", "plain text", "Māori 日本語 🐈", strings.Repeat("🌱", 20001)} {
		for _, max := range []int{-1, 0, 1, 2, 3, 4, 5, 100, 20000, len(input), len(input) + 1} {
			for name, fn := range map[string]func(string, int) string{"bytes": Bytes, "runes": Runes} {
				got := fn(input, max)
				if !utf8.ValidString(got) || !strings.HasPrefix(input, got) {
					t.Fatalf("%s(%q, %d) invalid prefix", name, input, max)
				}
				length := len(got)
				if name == "runes" {
					length = utf8.RuneCountInString(got)
				}
				if max <= 0 {
					if got != "" {
						t.Fatal("non-positive limit must return empty")
					}
					continue
				}
				if length > max {
					t.Fatalf("%s exceeded %d", name, max)
				}
				if len(got) < len(input) {
					_, size := utf8.DecodeRuneInString(input[len(got):])
					next := size
					if name == "runes" {
						next = 1
					}
					if length+next <= max {
						t.Fatalf("%s cut before the last complete character", name)
					}
				}
			}
		}
	}
}
