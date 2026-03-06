package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServer_AppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server: ServerConfig{
			Name: "Weather Service",
		},
		Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
	}

	err := cfg.Validate()
	require.NoError(t, err)

	assert.Equal(t, DefaultServerTitle, cfg.Server.Title)
	assert.Equal(t, DefaultServerDescription, cfg.Server.Description)
	assert.Equal(t, "example.com/weather-service", cfg.Server.Module)
}

func TestValidateServer_InvalidModuleReturnsSentinel(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server: ServerConfig{
			Name:   "weather",
			Module: "invalid module",
		},
		Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrServerModuleInvalid)
}

func TestValidateTransport_DefaultsAndSentinels(t *testing.T) {
	t.Parallel()

	t.Run("empty values default", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server: ServerConfig{Name: "weather", Module: "example.com/weather"},
		}

		err := cfg.Validate()
		require.NoError(t, err)
		assert.Equal(t, DefaultTransport, cfg.Transport.Type)
		assert.Equal(t, DefaultHTTPPort, cfg.Transport.HTTPPort)
	})

	t.Run("invalid values return sentinels", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server: ServerConfig{Name: "weather", Module: "example.com/weather"},
			Transport: TransportConfig{
				Type:     "grpc",
				HTTPPort: 70000,
			},
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTransportTypeInvalid)
		assert.ErrorIs(t, err, ErrTransportPortInvalid)
		assert.Equal(t, DefaultTransport, cfg.Transport.Type)
	})
}

func TestValidateResource_DefaultsAndURISentinel(t *testing.T) {
	t.Parallel()

	t.Run("readme defaults are applied", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server:    ServerConfig{Name: "weather", Module: "example.com/weather"},
			Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
			Resources: []ResourceConfig{{ID: DefaultResourceID, URI: "file://readme"}},
		}

		err := cfg.Validate()
		require.NoError(t, err)
		require.Len(t, cfg.Resources, 1)

		assert.Equal(t, "Readme", cfg.Resources[0].Title)
		assert.Equal(t, defaultReadmeDescription, cfg.Resources[0].Description)
		assert.Equal(t, DefaultResourceText, cfg.Resources[0].Text)
	})

	t.Run("uri without scheme returns sentinel", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server:    ServerConfig{Name: "weather", Module: "example.com/weather"},
			Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
			Resources: []ResourceConfig{{ID: "docs", URI: "relative/path"}},
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrURIMissingScheme)
	})
}

func TestValidatePrompt_DefaultsAndArgumentValidation(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server:    ServerConfig{Name: "weather", Module: "example.com/weather"},
		Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
		Prompts: []PromptConfig{{
			ID:        "onboarding",
			Arguments: []PromptArgumentConfig{{Name: ""}},
		}},
	}

	err := cfg.Validate()
	require.Error(t, err)

	assert.Equal(t, "Onboarding", cfg.Prompts[0].Title)
	assert.Equal(t, "Prompt stub for onboarding.", cfg.Prompts[0].Description)
	assert.Equal(t, "Prompt onboarding stub", cfg.Prompts[0].Template)
	assert.Equal(t, defaultPromptRole, cfg.Prompts[0].Role)
	assert.Contains(t, err.Error(), "argument[0].name is required")
}

func TestValidateTool_DefaultSchemasAndDuplicateIDs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server:    ServerConfig{Name: "weather", Module: "example.com/weather"},
		Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
		Tools: []ToolConfig{
			{ID: "search"},
			{ID: "search"},
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")

	require.Len(t, cfg.Tools, 2)
	assert.Equal(t, defaultJSONSchemaObject, cfg.Tools[0].InputSchema)
	assert.Equal(t, defaultJSONSchemaObject, cfg.Tools[0].OutputSchema)
	assert.Equal(t, defaultJSONSchemaObject, cfg.Tools[1].InputSchema)
	assert.Equal(t, defaultJSONSchemaObject, cfg.Tools[1].OutputSchema)
}

func TestValidate_AccumulatesSentinelErrors(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server: ServerConfig{
			Name:   "",
			Module: "invalid module",
		},
		Transport: TransportConfig{Type: "tcp", HTTPPort: 99999},
		Resources: []ResourceConfig{{ID: "docs", URI: "docs/path"}},
	}

	err := cfg.Validate()
	require.Error(t, err)

	assert.ErrorIs(t, err, ErrServerNameRequired)
	assert.ErrorIs(t, err, ErrServerModuleInvalid)
	assert.ErrorIs(t, err, ErrTransportTypeInvalid)
	assert.ErrorIs(t, err, ErrTransportPortInvalid)
	assert.ErrorIs(t, err, ErrURIMissingScheme)
}
