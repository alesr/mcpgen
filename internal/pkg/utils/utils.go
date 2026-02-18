package utils

import (
	"strings"
	"unicode"

	"github.com/alesr/strcase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const defaultServerName = "mcp"

func SplitIdentifier(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func DefaultServerName(name string) string {
	parts := SplitIdentifier(strings.ToLower(name))
	if len(parts) == 0 {
		return defaultServerName
	}
	return strings.Join(parts, "_")
}

func DefaultIfEmpty(value string, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

// GoIdent converts a string to an exported PascalCase identifier
// "foo-server" to "FooServer"
func GoIdent(input string) string {
	ident := strcase.ToCamel(input)
	if ident == "" {
		return "Item"
	}
	if unicode.IsDigit(rune(ident[0])) { // go identifiers cannot start with a digit
		return "N" + ident
	}
	return ident
}

// GoFileName converts a string to a snake_case filename
func GoFileName(input string) string {
	filename := strcase.ToSnake(input)
	if filename == "" {
		return "item"
	}
	return filename
}

func TitleCaseID(id string) string {
	spaced := strcase.ToDelimited(id, ' ')
	caser := cases.Title(language.AmericanEnglish)
	return caser.String(spaced)
}
