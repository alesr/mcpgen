package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig_Features(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		addTool     bool
		addResource bool
		addPrompt   bool
	}{
		{name: "all enabled", addTool: true, addResource: true, addPrompt: true},
		{name: "all disabled", addTool: false, addResource: false, addPrompt: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, _ := DefaultConfig("./generated", "stdio", 8080, tt.addTool, tt.addResource, tt.addPrompt)

			if tt.addTool {
				assert.NotNil(t, cfg.Tool)
			} else {
				assert.Nil(t, cfg.Tool)
			}

			if tt.addResource {
				assert.NotNil(t, cfg.Resource)
			} else {
				assert.Nil(t, cfg.Resource)
			}

			if tt.addPrompt {
				assert.NotNil(t, cfg.Prompt)
			} else {
				assert.Nil(t, cfg.Prompt)
			}
		})
	}
}
