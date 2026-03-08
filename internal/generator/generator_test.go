package generator

import (
	"errors"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
	"github.com/alesr/mcpgen/internal/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_validate(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Name: "foo",
		},
	}

	outDir := "/bar"

	tests := []struct {
		name        string
		givenCfg    *config.Config
		givenOutDir string
		expected    error
	}{
		{
			name:        "valid",
			givenCfg:    cfg,
			givenOutDir: outDir,
		},

		{
			name:        "no config",
			givenCfg:    nil,
			givenOutDir: outDir,
			expected:    errConfigIsNil,
		},
		{
			name:        "no out dir",
			givenCfg:    cfg,
			givenOutDir: "",
			expected:    errOutDirEmpty,
		},
		{
			name:        "out dir is current directory",
			givenCfg:    cfg,
			givenOutDir: ".",
			expected:    errOutDirUnsafe,
		},
		{
			name:        "out dir is absolute current directory",
			givenCfg:    cfg,
			givenOutDir: cwd,
			expected:    errOutDirUnsafe,
		},
		{
			name:        "out dir is filesystem root",
			givenCfg:    cfg,
			givenOutDir: string(filepath.Separator),
			expected:    errOutDirUnsafe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := Generator{
				Config: tt.givenCfg,
				OutDir: tt.givenOutDir,
			}

			err := gen.validate()

			assert.True(t, errors.Is(err, tt.expected))
		})
	}
}

func TestRenderTemplate_Handlers(t *testing.T) {
	t.Parallel()

	data := TemplateData{
		Module: "example.com/test",
		Tools: []ToolData{{
			ID:     "greet",
			GoName: "Greet",
		}},
	}

	content, err := RenderTemplate("handlers.go.gotmpl", data)
	require.NoError(t, err)

	_, err = format.Source(content)
	require.NoError(t, err)

	out := string(content)
	assert.NotContains(t, out, "req.Session.Elicit")
	assert.True(t, strings.Contains(out, "ToolNameFallbackGreet"))
}

func TestGenerator_Run_ConditionalFeatureFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		addTool         bool
		addResource     bool
		addPrompt       bool
		expectedPresent []string
		expectedMissing []string
		contains        []string
		notContains     []string
	}{
		{
			name:        "no optional features",
			addTool:     false,
			addResource: false,
			addPrompt:   false,
			expectedMissing: []string{
				"internal/mcpapp/tools",
				"internal/mcpapp/prompts",
				"internal/mcpapp/resources",
				"internal/mcpapp/stubs",
			},
			notContains: []string{
				"internal/mcpapp/tools/handlers",
				"tools.Register(server",
				"prompts.Register(server)",
				"resources.Register(server)",
				"handlers.New(logger)",
			},
		},
		{
			name:        "prompt only",
			addTool:     false,
			addResource: false,
			addPrompt:   true,
			expectedPresent: []string{
				"internal/mcpapp/prompts/prompts.go",
				"internal/mcpapp/prompts/prompts_test.go",
				"internal/mcpapp/stubs/stubs.go",
			},
			expectedMissing: []string{
				"internal/mcpapp/tools",
				"internal/mcpapp/resources",
			},
			contains: []string{
				"internal/mcpapp/prompts",
				"prompts.Register(server)",
			},
			notContains: []string{
				"internal/mcpapp/tools/handlers",
				"tools.Register(server",
				"resources.Register(server)",
				"handlers.New(logger)",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			outDir := filepath.Join(t.TempDir(), "generated")
			cfg, _ := scaffold.DefaultConfig(
				outDir,
				config.DefaultTransport,
				config.DefaultHTTPPort,
				tt.addTool,
				tt.addResource,
				tt.addPrompt,
			)

			require.NoError(t, cfg.Validate())

			gen := Generator{Config: cfg, OutDir: outDir}
			require.NoError(t, gen.Run())

			for _, relPath := range tt.expectedPresent {
				_, err := os.Stat(filepath.Join(outDir, relPath))
				require.NoError(t, err, "expected path to exist: %s", relPath)
			}

			for _, relPath := range tt.expectedMissing {
				_, err := os.Stat(filepath.Join(outDir, relPath))
				require.True(t, os.IsNotExist(err), "expected path to be missing: %s", relPath)
			}

			serverName := utils.DefaultServerName(cfg.Server.Name)
			mainPath := filepath.Join(outDir, "cmd", serverName, "main.go")
			appPath := filepath.Join(outDir, "internal", "mcpapp", "mcpapp.go")

			mainContent, err := os.ReadFile(mainPath)
			require.NoError(t, err)
			appContent, err := os.ReadFile(appPath)
			require.NoError(t, err)

			combined := string(mainContent) + "\n" + string(appContent)
			for _, s := range tt.contains {
				assert.Contains(t, combined, s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, combined, s)
			}
		})
	}
}
