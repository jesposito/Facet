package services

import (
	"fmt"
	"unicode/utf8"
)

const CustomCSSMaxChars = 20000

func CheckCustomCSSLength(css string) error {
	if count := utf8.RuneCountInString(css); count > CustomCSSMaxChars {
		return fmt.Errorf("custom CSS is %d characters; the limit is %d characters", count, CustomCSSMaxChars)
	}
	return nil
}
