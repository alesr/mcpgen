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
		assert.False(t, opts.NoInspector)
		assert.False(t, opts.DryRun)
	})

	t.Run("custom flags", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		opts, err := parseRunOptions([]string{
			"--name", "weather",
			"--transport", "http",
			"--with-tools=false",
			"--with-prompts=false",
			"--with-resources=true",
			"--no-inspector",
			"--dry-run",
		}, out)
		require.NoError(t, err)

		assert.True(t, opts.HasCLIInput)
		assert.Equal(t, "weather", opts.Name)
		assert.Equal(t, "http", opts.Transport)
		assert.False(t, opts.WithTools)
		assert.False(t, opts.WithPrompts)
		assert.True(t, opts.WithResources)
		assert.True(t, opts.NoInspector)
		assert.True(t, opts.DryRun)
	})

	t.Run("invalid transport", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		_, err := parseRunOptions([]string{"--transport", "tcp"}, out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --transport")
	})

	t.Run("help", func(t *testing.T) {
		t.Parallel()

		out := bytes.NewBuffer(nil)

		opts, err := parseRunOptions([]string{"--help"}, out)
		require.NoError(t, err)
		assert.True(t, opts.ShowHelp)
		assert.Contains(t, out.String(), "Usage: mcpgen [flags]")
		assert.Contains(t, out.String(), "--transport")
		assert.Contains(t, out.String(), "--dry-run")
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
		_, shouldTest, dryRun, err := runWithOptions(base, false)
		require.NoError(t, err)
		assert.False(t, shouldTest)
		assert.False(t, dryRun)
	})

	t.Run("inspector enabled with tty by default", func(t *testing.T) {
		_, shouldTest, dryRun, err := runWithOptions(base, true)
		require.NoError(t, err)
		assert.True(t, shouldTest)
		assert.False(t, dryRun)
	})

	t.Run("no-inspector wins even with tty", func(t *testing.T) {
		opts := base
		opts.NoInspector = true

		_, shouldTest, _, err := runWithOptions(opts, true)
		require.NoError(t, err)
		assert.False(t, shouldTest)
	})
}
