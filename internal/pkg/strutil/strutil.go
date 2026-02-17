package strutil

import (
	"strings"
	"unicode"
)

func SplitIdentifier(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
