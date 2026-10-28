package services

import (
	"strings"
	"testing"
)

func TestCustomCSSLimit(t *testing.T) {
	for _, char := range []string{"x", "ā", "🌱"} {
		for _, size := range []int{0, 19999, 20000, 20001} {
			err := CheckCustomCSSLength(strings.Repeat(char, size))
			if (err != nil) != (size > 20000) {
				t.Fatalf("%d %q characters: %v", size, char, err)
			}
		}
	}
}
