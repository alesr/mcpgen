package generator

import (
	"bytes"
	"embed"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*.gotmpl
var templateFS embed.FS

func RenderTemplate(name string, data TemplateData) ([]byte, error) {
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

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"quote":           strconv.Quote,
		"join":            strings.Join,
		"hasRequiredArgs": hasRequiredArgs,
	}
}

func hasRequiredArgs(prompts []PromptData) bool {
	return slices.ContainsFunc(prompts, func(p PromptData) bool {
		return len(p.RequiredArgs) > 0
	})
}
