package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCLI(t *testing.T, args ...string) (int, cli.Envelope, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, nil, &stdout, &stderr)
	var env cli.Envelope
	if stdout.Len() > 0 {
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &env), "stdout must be a single JSON envelope, got: %s", stdout.String())
	}
	return code, env, stderr.String()
}

func TestRun_Version(t *testing.T) {
	t.Parallel()
	code, env, _ := runCLI(t, "version", "--json")
	assert.Equal(t, 0, code)
	assert.True(t, env.OK)
	res, ok := env.Result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, Version, res["version"])
}

func TestRun_Schema(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"schema"}, nil, &stdout, &stderr)
	assert.Equal(t, 0, code)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))
	assert.Equal(t, "mcpgen", doc["tool"])
	assert.Contains(t, doc, "commands")
	assert.Contains(t, doc, "exit_codes")
	assert.Contains(t, doc, "config_json_schema")
}

func TestRun_VersionDashAliases(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"--version", "-v"} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{arg}, nil, &stdout, &stderr)
		assert.Equal(t, 0, code, arg)
		assert.Contains(t, stdout.String(), "mcpgen v"+Version, arg)
	}
}

func TestRun_HelpDashAliases(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"--help", "-h"} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{arg}, nil, &stdout, &stderr)
		assert.Equal(t, 0, code, arg)
		assert.Contains(t, stdout.String(), "Usage:", arg)
	}
}

func TestRun_Schema_BadFlag_StructuredEnvelope(t *testing.T) {
	t.Parallel()
	code, env, _ := runCLI(t, "schema", "--bogus")
	assert.Equal(t, cli.ExitUsage, code)
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, "schema", env.Error.Stage)
	assert.Equal(t, string(cli.CodeUsage), string(env.Error.Code))
}

func TestRun_MCP_BadFlag_StructuredEnvelope(t *testing.T) {
	t.Parallel()
	code, env, _ := runCLI(t, "mcp", "--bogus")
	assert.Equal(t, cli.ExitUsage, code)
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, "mcp", env.Error.Stage)
	assert.Equal(t, string(cli.CodeUsage), string(env.Error.Code))
}

func TestRun_UnknownCommand_Exit2(t *testing.T) {
	t.Parallel()
	code, _, _ := runCLI(t, "bogus")
	assert.Equal(t, cli.ExitUsage, code)
}

func TestRun_NoArgs_NotTTY_Exit2(t *testing.T) {
	t.Parallel()
	code, _, _ := runCLI(t)
	assert.Equal(t, cli.ExitUsage, code)
}

func TestRun_New_DryRun_JSON(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "weather")
	code, env, _ := runCLI(t, "new", dir, "--name", "weather", "--module", "github.com/acme/weather", "--transport", "http", "--dry-run", "--json")
	assert.Equal(t, 0, code)
	assert.True(t, env.OK)
	assert.NotEmpty(t, env.NextActions)
	res := env.Result.(map[string]any)
	plan := res["plan"].(map[string]any)
	files := plan["files"].([]any)
	assert.NotEmpty(t, files)
	// dry-run wrote nothing
	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestRun_New_ExistingDir_Exit3(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	code, env, _ := runCLI(t, "new", dir, "--json")
	assert.Equal(t, 0, code)
	assert.True(t, env.OK)

	// second run without --force: dir exists and non-empty
	code, env, _ = runCLI(t, "new", dir, "--json")
	assert.Equal(t, cli.ExitExists, code)
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, string(cli.CodeDirExists), string(env.Error.Code))
	assert.Equal(t, cli.ExitExists, env.Error.ExitCode)
	assert.NotEmpty(t, env.Error.Fix)
}

func TestRun_New_Then_Add_Then_Conflict(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	dir := filepath.Join(base, "proj")
	code, env, _ := runCLI(t, "new", dir, "--name", "p", "--module", "example.com/p", "--no-check", "--json")
	require.Equal(t, 0, code, "env: %+v", env)

	spec := filepath.Join(base, "health.json")
	require.NoError(t, os.WriteFile(spec, []byte(`{"id":"health","description":"h","readOnly":true}`), 0o644))

	code, env, _ = runCLI(t, "add", "tool", spec, "--dir", dir, "--json", "--no-check")
	require.Equal(t, 0, code, "env: %+v", env)
	assert.True(t, env.OK)

	// duplicate -> exit 5
	code, env, _ = runCLI(t, "add", "tool", spec, "--dir", dir, "--json", "--no-check")
	assert.Equal(t, cli.ExitConflict, code)
	assert.False(t, env.OK)
	assert.Equal(t, string(cli.CodeConflict), string(env.Error.Code))
	assert.NotEmpty(t, env.NextActions)
}

func TestRun_Add_NotAProject_Exit3(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	spec := filepath.Join(base, "t.json")
	require.NoError(t, os.WriteFile(spec, []byte(`{"id":"x"}`), 0o644))
	code, env, _ := runCLI(t, "add", "tool", spec, "--dir", base, "--json")
	assert.Equal(t, cli.ExitExists, code)
	assert.Equal(t, string(cli.CodeNotMCPGenProject), string(env.Error.Code))
}

func TestRun_Check_NotAProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	code, env, _ := runCLI(t, "check", dir, "--json", "--no-conformance")
	assert.Equal(t, cli.ExitFailure, code)
	assert.False(t, env.OK)
	assert.Equal(t, "config", env.Error.Stage)
}

func TestRun_New_BadTransport_Exit2(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "x")
	code, env, _ := runCLI(t, "new", dir, "--transport", "grpc", "--json")
	assert.Equal(t, cli.ExitUsage, code)
	assert.False(t, env.OK)
}

func TestRun_NewForce_PreservesAddedCapabilities(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	dir := filepath.Join(base, "proj")

	code, env, _ := runCLI(t, "new", dir, "--name", "p", "--module", "example.com/p", "--no-check", "--json")
	require.Equal(t, 0, code, "env: %+v", env)

	spec := filepath.Join(base, "health.json")
	require.NoError(t, os.WriteFile(spec, []byte(`{"id":"health","description":"h"}`), 0o644))
	code, env, _ = runCLI(t, "add", "tool", spec, "--dir", dir, "--json", "--no-check")
	require.Equal(t, 0, code, "env: %+v", env)

	// hand-edit the user-owned handler
	handler := filepath.Join(dir, "internal/mcpapp/tools/handlers/health.go")
	require.NoError(t, os.WriteFile(handler, []byte("package handlers\n\n// hand edit\n"), 0o644))

	// regen with --force: keeps the added tool, skips the edited user file
	code, env, _ = runCLI(t, "new", dir, "--force", "--no-check", "--json")
	require.Equal(t, 0, code, "env: %+v", env)
	res := env.Result.(map[string]any)
	skipped := res["files_skipped"].([]any)
	assert.Contains(t, skipped, "internal/mcpapp/tools/handlers/health.go")

	toolsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools.go"))
	require.NoError(t, err)
	assert.Contains(t, string(toolsGo), `"health"`)

	got, err := os.ReadFile(handler)
	require.NoError(t, err)
	assert.Contains(t, string(got), "hand edit")
}

func TestRun_NewForce_BrokenExistingConfig_Exit2(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "proj")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".mcpgen"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mcpgen.json"),
		[]byte(`{"version":1,"server":{"name":"x","module":"example.com/x"},"transport":{"type":"grpc"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".mcpgen", "manifest.json"),
		[]byte(`{"generator":"mcpgen","version":"1.0.0","files":{}}`), 0o644))

	// IsProject true but config unloadable: must fail, not silently regenerate
	code, env, _ := runCLI(t, "new", dir, "--force", "--json")
	assert.Equal(t, cli.ExitUsage, code)
	assert.False(t, env.OK)
	require.NotNil(t, env.Error)
	assert.Equal(t, string(cli.CodeConfigInvalid), string(env.Error.Code))
}

func TestRun_LegacyFlags_RouteToNew(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "legacy")
	code, env, _ := runCLI(t, "--name", "legacy", "--no-check", "--json", dir)
	// legacy form: flags first, dir as trailing positional
	assert.Equal(t, 0, code, "env: %+v", env)
	assert.True(t, env.OK)
	assert.DirExists(t, dir)
}
