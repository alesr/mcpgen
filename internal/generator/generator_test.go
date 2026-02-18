package generator

import (
	"errors"
	"testing"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGenerator_validate(t *testing.T) {
	t.Parallel()

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
