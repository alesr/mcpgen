package ops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeProject writes a minimal but loadable mcpgen project (config + manifest)
// so BuildConfig takes the base != nil branch.
func makeProject(t *testing.T, dir string, cfg *config.Config) {
	t.Helper()
	require.NoError(t, config.Save(dir, cfg))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, config.ManifestDir), 0o755))
	require.NoError(t, os.WriteFile(config.ManifestPath(dir),
		[]byte(`{"generator":"mcpgen","version":"test","files":{}}`), 0o644))
}

func TestBuildConfig_ForceFeaturesMergeNotReplace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	existing := &config.Config{
		Version:   config.ConfigVersion,
		Server:    config.Server{Name: "p", Module: "example.com/p", Version: "0.1.0"},
		Transport: config.Transport{Type: config.DefaultTransport},
		Features:  config.Features{Skills: true, OAuth: true},
	}
	makeProject(t, dir, existing)

	// Re-run with --features tasks only: existing skills+oauth must survive.
	cfg, err := BuildConfig(NewOptions{
		Dir:      dir,
		Features: config.Features{Tasks: true},
		SetFlags: map[string]bool{"features": true},
	})
	require.NoError(t, err)
	assert.Equal(t, config.Features{Tasks: true, Skills: true, OAuth: true}, cfg.Features)
}

func TestBuildConfig_ForceWithoutFeaturesFlagKeepsExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	existing := &config.Config{
		Version:   config.ConfigVersion,
		Server:    config.Server{Name: "p", Module: "example.com/p", Version: "0.1.0"},
		Transport: config.Transport{Type: config.DefaultTransport},
		Features:  config.Features{OAuth: true},
	}
	makeProject(t, dir, existing)

	cfg, err := BuildConfig(NewOptions{
		Dir:      dir,
		Features: config.Features{}, // flag not set
		SetFlags: map[string]bool{"force": true},
	})
	require.NoError(t, err)
	assert.Equal(t, config.Features{OAuth: true}, cfg.Features)
}
