package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config returns no error", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server: ServerConfig{
				Name:    "test-server",
				Version: "1.0.0",
				Module:  "example.com/test",
			},
			Transport: TransportConfig{
				Type:     "stdio",
				HTTPPort: 8080,
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("accumulates multiple errors", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server: ServerConfig{
				Name: "",
			},
			Transport: TransportConfig{
				Type:     "invalid",
				HTTPPort: 99999,
			},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrServerNameRequired)
		assert.ErrorIs(t, err, ErrTransportTypeInvalid)
		assert.ErrorIs(t, err, ErrTransportPortInvalid)
	})
}

func TestValidateTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tools     []ToolConfig
		errsCount int
	}{
		{
			"valid tool",
			[]ToolConfig{{ID: "my-tool"}},
			0,
		},
		{
			"missing tool ID",
			[]ToolConfig{{ID: ""}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{Tools: tt.tools}

			errs := cfg.validateTool()
			assert.Len(t, errs, tt.errsCount)
		})
	}
}
