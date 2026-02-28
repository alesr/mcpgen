package generator

import (
	"errors"
	"os"
	"path/filepath"
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
