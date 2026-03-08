package app

import (
	"bytes"
	"testing"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRunOptions(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		opts, err := parseRunOptions(nil, out)
		require.NoError(t, err)

		assert.False(t, opts.ShowHelp)
		assert.False(t, opts.HasCLIInput)
		assert.Equal(t, config.DefaultServerName, opts.Name)
		assert.Equal(t, config.DefaultTransport, opts.Transport)
		assert.True(t, opts.WithTools)
		assert.True(t, opts.WithPrompts)
		assert.True(t, opts.WithResources)
		assert.False(t, opts.WithElicitation)
		assert.False(t, opts.NoInspector)
	})

	t.Run("custom flags", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		opts, err := parseRunOptions([]string{
			"--name", "weather",
			"--transport", "http",
			"--with-tools=true",
			"--with-prompts=false",
			"--with-resources=true",
			"--with-elicitation=true",
			"--no-inspector",
		}, out)
		require.NoError(t, err)

		assert.True(t, opts.HasCLIInput)
		assert.Equal(t, "weather", opts.Name)
		assert.Equal(t, "http", opts.Transport)
		assert.True(t, opts.WithTools)
		assert.False(t, opts.WithPrompts)
		assert.True(t, opts.WithResources)
		assert.True(t, opts.WithElicitation)
		assert.True(t, opts.NoInspector)
	})

	t.Run("invalid transport", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		_, err := parseRunOptions([]string{"--transport", "tcp"}, out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --transport")
	})

	t.Run("elicitation requires tools", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		_, err := parseRunOptions([]string{"--with-tools=false", "--with-elicitation=true"}, out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--with-elicitation requires --with-tools=true")
	})

	t.Run("help", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		opts, err := parseRunOptions([]string{"--help"}, out)
		require.NoError(t, err)
		assert.True(t, opts.ShowHelp)
		assert.Contains(t, out.String(), "Usage: mcpgen [flags]")
		assert.Contains(t, out.String(), "--transport")
	})

	t.Run("positional args are rejected", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		_, err := parseRunOptions([]string{"extra"}, out)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "unexpected positional arguments")
	})
}

func TestRunWithOptions(t *testing.T) {
	t.Parallel()

	base := runOptions{
		Name:          "weather",
		Transport:     "stdio",
		WithTools:     true,
		WithPrompts:   true,
		WithResources: true,
	}

	t.Run("inspector disabled without tty", func(t *testing.T) {
		_, shouldTest, err := runWithOptions(base, false)
		require.NoError(t, err)
		assert.False(t, shouldTest)
	})

	t.Run("inspector enabled with tty by default", func(t *testing.T) {
		_, shouldTest, err := runWithOptions(base, true)
		require.NoError(t, err)
		assert.True(t, shouldTest)
	})

	t.Run("no-inspector wins even with tty", func(t *testing.T) {
		opts := base
		opts.NoInspector = true

		_, shouldTest, err := runWithOptions(opts, true)
		require.NoError(t, err)
		assert.False(t, shouldTest)
	})

	t.Run("elicitation without tools is rejected", func(t *testing.T) {
		opts := base
		opts.WithTools = false
		opts.WithElicitation = true

		_, _, err := runWithOptions(opts, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--with-elicitation requires --with-tools=true")
	})
}
