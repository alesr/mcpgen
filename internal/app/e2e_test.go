package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentLoopE2E is the scripted fake-agent loop from the PRD: generate →
// inject a deliberate test failure → check (assert the envelope names the
// stage and fix carries file:line) → fix → check exit 0.
func TestAgentLoopE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("E2E needs the go toolchain and module downloads")
	}

	dir := filepath.Join(t.TempDir(), "weather")

	// 1. mcpgen new --json (runs the validation pipeline by default; stdio
	// transport means conformance auto-skips)
	code, env, _ := runCLI(t, "new", dir, "--name", "weather", "--module", "example.com/weather", "--json")
	require.Equal(t, 0, code, "new failed: %+v", env)
	require.True(t, env.OK)
	res := env.Result.(map[string]any)
	require.NotNil(t, res["check"], "new should include check stages")

	// 2. inject a deliberate test failure
	handlerTest := filepath.Join(dir, "internal/mcpapp/tools/handlers/greet_test.go")
	require.NoError(t, os.WriteFile(handlerTest, append(mustRead(t, handlerTest),
		[]byte("\nfunc TestDeliberateFailure(t *testing.T) {\n\tt.Fatal(\"boom\")\n}\n")...), 0o644))

	// 3. mcpgen check --json → exit 1, stage=test, fix has file:line
	code, env, _ = runCLI(t, "check", dir, "--json", "--no-conformance")
	assert.Equal(t, cli.ExitFailure, code)
	require.NotNil(t, env.Error)
	assert.Equal(t, "test", env.Error.Stage)
	assert.Equal(t, string(cli.CodeTestsFailed), string(env.Error.Code))
	assert.Contains(t, env.Error.Fix, "TestDeliberateFailure")
	assert.Contains(t, env.Error.Fix, "greet_test.go:")
	assert.NotEmpty(t, env.NextActions)

	// 4. fix: remove the failing test
	require.NoError(t, os.WriteFile(handlerTest, mustRead(t, handlerTest)[:strings.Index(string(mustRead(t, handlerTest)), "\nfunc TestDeliberateFailure")], 0o644))

	// 5. check again → exit 0
	code, env, _ = runCLI(t, "check", dir, "--json", "--no-conformance")
	assert.Equal(t, 0, code, "final check: %+v", env)
	assert.True(t, env.OK)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return raw
}

// TestCheckUnknownConfigVersion_Exit2 asserts the PRD contract: unknown
// mcpgen.json versions fail with exit code 2 and a fix hint.
func TestCheckUnknownConfigVersion_Exit2(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := `{"version": 99, "server": {"name": "x", "module": "example.com/x"}, "transport": {"type": "stdio"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mcpgen.json"), []byte(cfg), 0o644))

	code, env, _ := runCLI(t, "check", dir, "--json", "--no-conformance")
	assert.Equal(t, cli.ExitUsage, code)
	require.NotNil(t, env.Error)
	assert.Equal(t, "config", env.Error.Stage)
	assert.Equal(t, string(cli.CodeConfigVersion), string(env.Error.Code))
	assert.Contains(t, env.Error.Fix, `"version": 1`)
}

// TestCheckServerJSONStage validates the serverjson stage of the pipeline.
func TestCheckServerJSONStage(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	dir := filepath.Join(base, "proj")
	code, env, _ := runCLI(t, "new", dir, "--name", "sj", "--module", "example.com/sj", "--no-check", "--json")
	require.Equal(t, 0, code, "env: %+v", env)

	// break server.json
	require.NoError(t, os.WriteFile(filepath.Join(dir, "server.json"), []byte(`{"name": "", "packages": []}`), 0o644))
	code, env, _ = runCLI(t, "check", dir, "--json", "--no-conformance")
	assert.Equal(t, cli.ExitFailure, code)
	require.NotNil(t, env.Error)
	assert.Equal(t, "serverjson", env.Error.Stage)
	assert.Contains(t, env.Error.Message, "server.json")
}
