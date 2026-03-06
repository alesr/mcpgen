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
			Resource:  &ResourceConfig{ID: DefaultResourceID, URI: "file://readme"},
		}

		err := cfg.Validate()
		require.NoError(t, err)

		require.NotNil(t, cfg.Resource)
		assert.Equal(t, "Readme", cfg.Resource.Title)
		assert.Equal(t, defaultReadmeDescription, cfg.Resource.Description)
		assert.Equal(t, DefaultResourceText, cfg.Resource.Text)
	})

	t.Run("uri without scheme returns sentinel", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Server:    ServerConfig{Name: "weather", Module: "example.com/weather"},
			Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
			Resource:  &ResourceConfig{ID: "docs", URI: "relative/path"},
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
		Prompt: &PromptConfig{
			ID:        "onboarding",
			Arguments: []PromptArgumentConfig{{Name: ""}},
		},
	}

	err := cfg.Validate()
	require.Error(t, err)

	require.NotNil(t, cfg.Prompt)
	assert.Equal(t, "Onboarding", cfg.Prompt.Title)
	assert.Equal(t, "Prompt stub for onboarding.", cfg.Prompt.Description)
	assert.Equal(t, "Prompt onboarding stub", cfg.Prompt.Template)
	assert.Equal(t, defaultPromptRole, cfg.Prompt.Role)
	assert.Contains(t, err.Error(), "argument[0].name is required")
}

func TestValidateElicitation_RequiresTool(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server:      ServerConfig{Name: "weather", Module: "example.com/weather"},
		Transport:   TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
		Elicitation: ElicitationConfig{Enabled: true},
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrElicitationNeedsTool)
}

func TestValidateTool_DefaultSchemas(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server:    ServerConfig{Name: "weather", Module: "example.com/weather"},
		Transport: TransportConfig{Type: "stdio", HTTPPort: DefaultHTTPPort},
		Tool:      &ToolConfig{ID: "search"},
	}

	err := cfg.Validate()
	require.NoError(t, err)

	require.NotNil(t, cfg.Tool)
	assert.Equal(t, defaultJSONSchemaObject, cfg.Tool.InputSchema)
	assert.Equal(t, defaultJSONSchemaObject, cfg.Tool.OutputSchema)
}

func TestValidate_AccumulatesSentinelErrors(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Server: ServerConfig{
			Name:   "",
			Module: "invalid module",
		},
		Transport: TransportConfig{Type: "tcp", HTTPPort: 99999},
		Resource:  &ResourceConfig{ID: "docs", URI: "docs/path"},
	}

	err := cfg.Validate()
	require.Error(t, err)

	assert.ErrorIs(t, err, ErrServerNameRequired)
	assert.ErrorIs(t, err, ErrServerModuleInvalid)
	assert.ErrorIs(t, err, ErrTransportTypeInvalid)
	assert.ErrorIs(t, err, ErrTransportPortInvalid)
	assert.ErrorIs(t, err, ErrURIMissingScheme)
}
