// Package textlimit bounds text without splitting UTF-8 characters.
package textlimit

import "unicode/utf8"

func Runes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	count := 0
	for offset := range value {
		if count == max {
			return value[:offset]
		}
		count++
	}
	return value
}

// Bytes preserves byte budgets used for prompts and transport metadata.
// Input is expected to be valid UTF-8; existing invalid input is not repaired.
func Bytes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(value) <= max {
		return value
	}
	for max > 0 && !utf8.RuneStart(value[max]) {
		max--
	}
	return value[:max]
}
