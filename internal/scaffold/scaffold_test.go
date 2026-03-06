package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig_ElicitationFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		addTool        bool
		addElicitation bool
		expected       bool
	}{
		{
			name:           "enabled",
			addTool:        true,
			addElicitation: true,
			expected:       true,
		},
		{
			name:           "disabled",
			addTool:        true,
			addElicitation: false,
			expected:       false,
		},
		{
			name:           "requires tool",
			addTool:        false,
			addElicitation: true,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, _ := DefaultConfig("./generated", "stdio", 8080, tt.addTool, true, true, tt.addElicitation)
			assert.Equal(t, tt.expected, cfg.Elicitation.Enabled)
		})
	}
}
