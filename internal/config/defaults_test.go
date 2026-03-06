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
			DefaultServerModule,
		},
		{
			"only invalid characters",
			"!!!",
			DefaultServerModule,
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

	tests := []struct {
		name            string
		id              string
		expectedTitle   string
		expectedDesc    string
		containsOnly    bool
		expectedSnippet string
	}{
		{
			name:            "greet stub overrides",
			id:              "greet",
			expectedTitle:   "Greet",
			containsOnly:    true,
			expectedSnippet: "Greets a user",
		},
		{
			name:          "generic tool fallback",
			id:            "foo-bar",
			expectedTitle: "Foo Bar",
			expectedDesc:  "Tool stub for foo-bar.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedTitle, defaultToolTitle(tt.id))

			desc := defaultToolDescription(tt.id)
			if tt.containsOnly {
				assert.Contains(t, desc, tt.expectedSnippet)
				return
			}

			assert.Equal(t, tt.expectedDesc, desc)
		})
	}
}

func TestDefaultResourceLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		expectedTitle string
		expectedText  string
	}{
		{
			name:          "readme stub overrides",
			id:            "readme",
			expectedTitle: "Readme",
			expectedText:  DefaultResourceText,
		},
		{
			name:          "generic resource fallback",
			id:            "config-file",
			expectedTitle: "Config File",
			expectedText:  "This is the config-file stub.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedTitle, defaultResourceTitle(tt.id))
			assert.Equal(t, tt.expectedText, defaultResourceTextForID(tt.id))
		})
	}
}

func TestDefaultPromptLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		id               string
		expectedTitle    string
		expectedTemplate string
	}{
		{
			name:             "welcome stub overrides",
			id:               "welcome",
			expectedTitle:    "Welcome",
			expectedTemplate: DefaultPromptTemplate,
		},
		{
			name:             "generic prompt fallback",
			id:               "onboarding",
			expectedTitle:    "Onboarding",
			expectedTemplate: "Prompt onboarding stub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedTitle, defaultPromptTitle(tt.id))
			assert.Equal(t, tt.expectedTemplate, defaultPromptTemplateForID(tt.id))
		})
	}
}
