package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultModulePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"standard name",
			"my-server",
			"example.com/my-server",
		},
		{
			"mixed case and spaces",
			" My Server ",
			"example.com/my-server",
		},
		{
			"invalid characters",
			"my_server@123!",
			"example.com/my-server-123",
		},
		{
			"empty input fallback",
			"",
			"example.com/mcp-server",
		},
		{
			"only invalid characters",
			"!!!",
			"example.com/mcp-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := defaultModulePath(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDefaultToolTitlesAndDescriptions(t *testing.T) {
	t.Parallel()

	t.Run("greet stub overrides", func(t *testing.T) {
		id := "greet"
		assert.Equal(t, "Greet", defaultToolTitle(id))
		assert.Contains(t, defaultToolDescription(id), "Greets a user")
	})

	t.Run("generic tool fallback", func(t *testing.T) {
		id := "foo-bar"
		assert.Equal(t, "Foo Bar", defaultToolTitle(id))
		assert.Equal(t, "Tool stub for foo-bar.", defaultToolDescription(id))
	})
}

func TestDefaultResourceLogic(t *testing.T) {
	t.Parallel()

	t.Run("readme stub overrides", func(t *testing.T) {
		id := "readme"
		assert.Equal(t, "Readme", defaultResourceTitle(id))
		assert.Equal(t, DefaultResourceText, defaultResourceTextForID(id))
	})

	t.Run("generic resource fallback", func(t *testing.T) {
		id := "config-file"
		assert.Equal(t, "Config File", defaultResourceTitle(id))
		assert.Equal(t, "This is the config-file stub.", defaultResourceTextForID(id))
	})
}

func TestDefaultPromptLogic(t *testing.T) {
	t.Parallel()

	t.Run("welcome stub overrides", func(t *testing.T) {
		id := "welcome"
		assert.Equal(t, "Welcome", defaultPromptTitle(id))
		assert.Equal(t, DefaultPromptTemplate, defaultPromptTemplateForID(id))
	})

	t.Run("generic prompt fallback", func(t *testing.T) {
		id := "onboarding"
		assert.Equal(t, "Onboarding", defaultPromptTitle(id))
		assert.Equal(t, "Prompt onboarding stub", defaultPromptTemplateForID(id))
	})
}
