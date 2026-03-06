package generator

import (
	"errors"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alesr/mcpgen/internal/config"
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

func TestRenderTemplate_HandlersWithElicitation(t *testing.T) {
	t.Parallel()

	data := TemplateData{
		Module:             "example.com/test",
		ElicitationEnabled: true,
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
	assert.True(t, strings.Contains(out, "req.Session.Elicit"))
	assert.True(t, strings.Contains(out, "ToolNameFallbackGreet"))
}
