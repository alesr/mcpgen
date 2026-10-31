package generator

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*.gotmpl
var templateFS embed.FS

// RenderTemplate renders the named embedded template with data.
func RenderTemplate(name string, data any) ([]byte, error) {
	tmpl, err := template.New(name).Funcs(templateFuncs()).ParseFS(templateFS, "templates/*.gotmpl")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

// RenderStatement renders one of the stmt_*.gotmpl templates (a single
// registration statement for `mcpgen add` insertion).
func RenderStatement(name string, data any) (string, error) {
	raw, err := RenderTemplate(name, data)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(raw), "\n"), nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"quote": strconv.Quote,
		"join":  strings.Join,
	}
}
