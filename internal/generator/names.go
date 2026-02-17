package generator

import (
	"strings"
	"unicode"
)

// sanitizes an arbitrary string into a valid, exported Go PascalCase identifier,
// ensuring it starts with a letter (prefixed with 'N' if the input begins with a digit).
func goIdent(input string) string {
	parts := splitIdentifier(input)
	if len(parts) == 0 {
		return "Item"
	}

	var b strings.Builder
	for _, p := range parts {
		for i, r := range p {
			if i == 0 {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
		}
	}

	ident := b.String()
	if len(ident) > 0 && unicode.IsDigit(rune(ident[0])) {
		return "N" + ident
	}
	return ident
}

func goFileName(input string) string {
	parts := splitIdentifier(strings.ToLower(input))
	if len(parts) == 0 {
		return "item"
	}
	return strings.Join(parts, "_") // snake_case
}

func splitIdentifier(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
