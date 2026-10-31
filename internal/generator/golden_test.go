package generator

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseGo(src []byte) (any, error) {
	return parser.ParseFile(token.NewFileSet(), "gen.go", src, parser.ParseComments)
}

// goldenCases are the template permutations snapshot-tested in
// testdata/golden. Regenerate with: UPDATE_GOLDEN=1 go test ./internal/generator -run Golden
var goldenCases = map[string]func() *config.Config{
	"stdio-default": func() *config.Config {
		cfg := config.DefaultConfig("stdio", 8080, true, true, true)
		cfg.Server.Name = "weather"
		cfg.Server.Module = "github.com/acme/weather"
		return cfg
	},
	"http-full": func() *config.Config {
		cfg := config.DefaultConfig("http", 9090, true, true, true)
		cfg.Server.Name = "weather"
		cfg.Server.Module = "github.com/acme/weather"
		cfg.Tools = append(cfg.Tools, config.Tool{
			ID:          "get_forecast",
			Description: "Get the forecast for a city.",
			InputSchema: []byte(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
			ReadOnly:    true, Idempotent: true,
		})
		cfg.Prompts = append(cfg.Prompts, config.Prompt{
			ID: "report", Description: "Build a report.",
			Arguments: []config.PromptArgument{{Name: "topic", Required: true, Description: "Report topic"}},
		})
		cfg.Resources = append(cfg.Resources, config.Resource{
			ID: "config", URITemplate: "file:///config/{name}", MIMEType: "application/json", Text: "{}",
		})
		cfg.Features = config.Features{Tasks: true, Skills: true, OAuth: true}
		return cfg
	},
	"http-latest-only-empty": func() *config.Config {
		cfg := config.DefaultConfig("http", 8080, false, false, false)
		cfg.Server.Name = "bare"
		cfg.Server.Module = "example.com/bare"
		cfg.Protocol = config.ProtocolLatestOnly
		return cfg
	},
}

func TestGolden(t *testing.T) {
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	for name, build := range goldenCases {
		t.Run(name, func(t *testing.T) {
			cfg := build()
			require.NoError(t, cfg.Validate())

			dir := filepath.Join(t.TempDir(), "proj")
			gen := &Generator{Config: cfg, OutDir: dir, Version: "golden"}
			_, err := gen.Run()
			require.NoError(t, err)

			goldenDir := filepath.Join("testdata", "golden", name)
			if update {
				require.NoError(t, os.RemoveAll(goldenDir))
				require.NoError(t, copyTree(dir, goldenDir))
			}

			want := listFiles(t, goldenDir)
			got := listFiles(t, dir)
			require.Equal(t, want, got, "file tree mismatch (run UPDATE_GOLDEN=1 to regenerate)")
			for _, f := range want {
				wc, err := os.ReadFile(filepath.Join(goldenDir, f))
				require.NoError(t, err)
				gc, err := os.ReadFile(filepath.Join(dir, f))
				require.NoError(t, err)
				assert.Equal(t, string(wc), string(gc), f)
			}
		})
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// TestTemplateLint_ForbiddenSpecIdentifiers guards the PRD rule: generated
// code must never use methods removed or deprecated in MCP 2026-07-28
// (initialize handling, ping, logging/setLevel, resources/subscribe, roots,
// sampling, logging).
func TestTemplateLint_ForbiddenSpecIdentifiers(t *testing.T) {
	t.Parallel()
	forbidden := []string{
		"SubscribeHandler",
		"UnsubscribeHandler",
		"RootsListChangedHandler",
		"SetLevel",
		"setLevel",
		"resources/subscribe",
		"sampling/createMessage",
		"notifications/message",
		"PingHandler",
		"logging/setLevel",
	}
	entries, err := templateFS.ReadDir("templates")
	require.NoError(t, err)
	for _, e := range entries {
		raw, err := templateFS.ReadFile("templates/" + e.Name())
		require.NoError(t, err)
		for _, f := range forbidden {
			assert.Falsef(t, bytes.Contains(raw, []byte(f)), "template %s contains forbidden identifier %q", e.Name(), f)
		}
	}
}

// TestTemplates_RenderAll ensures every template renders without error for
// both a full and an empty config.
func TestTemplates_RenderAll(t *testing.T) {
	t.Parallel()
	full := config.DefaultConfig("http", 9090, true, true, true)
	full.Server.Name = "weather"
	full.Server.Module = "github.com/acme/weather"
	full.Tools = append(full.Tools, config.Tool{ID: "get_forecast", InputSchema: []byte(`{"type":"object"}`)})
	full.Prompts = append(full.Prompts, config.Prompt{ID: "report", Arguments: []config.PromptArgument{{Name: "topic", Required: true}}})
	full.Resources = append(full.Resources, config.Resource{ID: "cfg", URITemplate: "file:///c/{name}"})
	full.Features = config.Features{Tasks: true, Skills: true, OAuth: true}
	require.NoError(t, full.Validate())

	empty := config.DefaultConfig("stdio", 8080, false, false, false)
	empty.Server.Name = "bare"
	empty.Server.Module = "example.com/bare"
	require.NoError(t, empty.Validate())

	for _, cfg := range []*config.Config{full, empty} {
		data, err := BuildTemplateData(cfg, "test")
		require.NoError(t, err)
		entries, err := templateFS.ReadDir("templates")
		require.NoError(t, err)
		for _, e := range entries {
			name := e.Name()
			t.Run(name, func(t *testing.T) {
				var tmplData any = data
				switch name {
				case "tool_handler.go.gotmpl", "tool_handler_test.go.gotmpl", "stmt_tool.go.gotmpl":
					if len(data.Tools) == 0 {
						t.Skip("no tools")
					}
					tmplData = data.Tools[0]
				case "prompt.go.gotmpl", "prompt_test.go.gotmpl", "stmt_prompt.go.gotmpl":
					if len(data.Prompts) == 0 {
						t.Skip("no prompts")
					}
					tmplData = data.Prompts[0]
				case "resource.go.gotmpl", "resource_test.go.gotmpl", "stmt_resource.go.gotmpl":
					if len(data.Resources) == 0 {
						t.Skip("no resources")
					}
					tmplData = data.Resources[0]
				}
				raw, err := RenderTemplate(name, tmplData)
				require.NoError(t, err)
				assert.NotEmpty(t, raw)
			})
		}
	}
}
