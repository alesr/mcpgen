package ops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alesr/mcpgen/internal/cli"
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

func TestBuildConfig_FreshProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg, err := BuildConfig(NewOptions{
		Dir:       dir,
		Name:      "weather",
		Module:    "example.com/weather",
		Transport: "http",
		Port:      9090,
		Protocol:  "latest-only",
		SetFlags:  map[string]bool{},
	})
	require.NoError(t, err)
	assert.Equal(t, "weather", cfg.Server.Name)
	assert.Equal(t, "example.com/weather", cfg.Server.Module)
	assert.Equal(t, "http", cfg.Transport.Type)
	assert.Equal(t, 9090, cfg.Transport.HTTPPort)
	assert.Equal(t, "latest-only", cfg.Protocol)
	// No specs and stubs enabled: default tool/prompt/resource are added.
	require.Len(t, cfg.Tools, 1)
	assert.Equal(t, config.DefaultToolID, cfg.Tools[0].ID)
	require.Len(t, cfg.Prompts, 1)
	require.Len(t, cfg.Resources, 1)
}

func TestBuildConfig_FreshProjectNoStubs(t *testing.T) {
	t.Parallel()
	cfg, err := BuildConfig(NewOptions{
		Dir:      t.TempDir(),
		Name:     "bare",
		NoStubs:  true,
		SetFlags: map[string]bool{},
	})
	require.NoError(t, err)
	assert.Empty(t, cfg.Tools)
	assert.Empty(t, cfg.Prompts)
	assert.Empty(t, cfg.Resources)
}

func TestBuildConfig_BadToolIDRejected(t *testing.T) {
	t.Parallel()
	_, err := BuildConfig(NewOptions{
		Dir:      t.TempDir(),
		Name:     "x",
		Tools:    []config.Tool{{ID: "../evil"}},
		NoStubs:  true,
		SetFlags: map[string]bool{},
	})
	require.Error(t, err)
	ee := cli.AsExitError(err)
	require.NotNil(t, ee)
	assert.Equal(t, cli.CodeConfigInvalid, ee.Code)
	assert.Contains(t, err.Error(), "id must match")
}

func TestBuildConfig_ExistingProjectUnloadable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// IsProject requires mcpgen.json + manifest; write a broken config.
	require.NoError(t, os.WriteFile(filepath.Join(dir, config.FileName), []byte(`{invalid`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, config.ManifestDir), 0o755))
	require.NoError(t, os.WriteFile(config.ManifestPath(dir), []byte(`{"files":{}}`), 0o644))

	_, err := BuildConfig(NewOptions{Dir: dir, Name: "x", Force: true, SetFlags: map[string]bool{"force": true}})
	require.Error(t, err)
	assert.Equal(t, cli.CodeConfigInvalid, cli.AsExitError(err).Code)
}

func TestLoadSpecs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	plain := filepath.Join(dir, "a.json")
	require.NoError(t, os.WriteFile(plain, []byte(`{"id":"orig","title":"T"}`), 0o644))
	named := filepath.Join(dir, "b.json")
	require.NoError(t, os.WriteFile(named, []byte(`{"id":"orig"}`), 0o644))

	specs, err := LoadSpecs[config.Tool]([]string{plain, "renamed=" + named})
	require.NoError(t, err)
	require.Len(t, specs, 2)
	assert.Equal(t, "orig", specs[0].ID)
	assert.Equal(t, "renamed", specs[1].ID, "name=file form overrides the spec id")
}

func TestLoadSpecs_Errors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte(`{nope`), 0o644))

	_, err := LoadSpecs[config.Tool]([]string{filepath.Join(dir, "missing.json")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")

	_, err = LoadSpecs[config.Tool]([]string{bad})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestApplyFeatures(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	require.NoError(t, ApplyFeatures(cfg, ""))
	assert.Equal(t, config.Features{}, cfg.Features)

	require.NoError(t, ApplyFeatures(cfg, "tasks, Skills ,OAUTH"))
	assert.Equal(t, config.Features{Tasks: true, Skills: true, OAuth: true}, cfg.Features)

	err := ApplyFeatures(cfg, "bogus")
	require.Error(t, err)
	assert.Equal(t, cli.CodeUsage, cli.AsExitError(err).Code)
	assert.Contains(t, err.Error(), "unknown feature")
}
