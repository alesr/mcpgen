package checks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun_InvalidConfig_TidySkipped: with a broken mcpgen.json the tidy stage
// must report skipped (not failed with an empty message).
func TestRun_InvalidConfig_TidySkipped(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mcpgen.json"),
		[]byte(`{"version":1,"server":{"name":"x","module":"example.com/x"},"transport":{"type":"grpc"}}`), 0o644))
	res := Run(context.Background(), dir, Options{Timeout: time.Minute})
	require.False(t, res.OK)
	stages := map[string]Stage{}
	for _, s := range res.Stages {
		stages[s.Name] = s
	}
	assert.Equal(t, StatusFailed, stages["config"].Status)
	assert.Equal(t, StatusSkipped, stages["tidy"].Status)
	assert.Equal(t, StatusSkipped, stages["serverjson"].Status)
}

func TestFirstTestFailure(t *testing.T) {
	t.Parallel()
	out := `=== RUN   TestGreet
--- FAIL: TestGreet (0.00s)
    greet_test.go:30: boom
FAIL
FAIL	github.com/acme/weather/internal/mcpapp/tools/handlers	0.01s`
	loc, test := firstTestFailure(out)
	assert.Equal(t, "TestGreet", test)
	assert.Equal(t, "greet_test.go:30", loc)
}

func TestFirstTestFailure_NoLocation(t *testing.T) {
	t.Parallel()
	loc, test := firstTestFailure("--- FAIL: TestX\nFAIL\n")
	assert.Equal(t, "TestX", test)
	assert.Empty(t, loc)
}

func TestFirstGoLocation(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "main.go:12", firstGoLocation("# pkg\nmain.go:12: undefined: X\n"))
	assert.Empty(t, firstGoLocation("no locations here"))
}

func TestParseConformance(t *testing.T) {
	t.Parallel()
	out := `Running active suite (30 scenarios) against http://127.0.0.1:1234/mcp

=== Running scenario: server-initialize ===
Running client scenario 'server-initialize' against server: http://127.0.0.1:1234/mcp

=== Running scenario: tools-call-simple-text ===
Running client scenario 'tools-call-simple-text' against server: http://127.0.0.1:1234/mcp

=== Running scenario: resources-subscribe ===
Running client scenario 'resources-subscribe' against server: http://127.0.0.1:1234/mcp

=== Running scenario: tools-list ===
Running client scenario 'tools-list' against server: http://127.0.0.1:1234/mcp

✓ server-initialize: 1 passed, 0 failed
✗ tools-call-simple-text: 0 passed, 1 failed
✗ resources-subscribe: 0 passed, 1 failed
✓ tools-list: 1 passed, 0 failed

Total: 2 passed, 2 failed
`
	s := parseConformance(out)
	assert.Equal(t, 2, s.passed)
	assert.Equal(t, 4, s.total)
	assert.Empty(t, s.gatingFailures, "fixture scenarios must not gate")
	assert.Contains(t, s.fixtureSkipped, "tools-call-simple-text")
	assert.Contains(t, s.fixtureSkipped, "resources-subscribe")
}

func TestParseConformance_GatingFailure(t *testing.T) {
	t.Parallel()
	out := `=== Running scenario: server-initialize ===
=== Failed Checks ===
  - ServerInitialize: handshake
    Error: Failed: handshake mismatch

=== Running scenario: ping ===
run
✗ server-initialize: 0 passed, 1 failed
✓ ping: 1 passed, 0 failed
Total: 1 passed, 1 failed
`
	s := parseConformance(out)
	require.Len(t, s.gatingFailures, 1)
	assert.Equal(t, "server-initialize", s.gatingFailures[0].scenario)
	assert.Contains(t, s.gatingFailures[0].errMsg, "handshake mismatch")
}

func TestParseConformance_UnknownScenarioNonFixtureErrorGates(t *testing.T) {
	t.Parallel()
	out := `=== Running scenario: brand-new-protocol-thing ===
run
✗ brand-new-protocol-thing: 0 passed, 1 failed
    Error: Failed: connection reset by peer
Total: 0 passed, 1 failed
`
	s := parseConformance(out)
	require.Len(t, s.gatingFailures, 1, "unknown scenario with a real error must gate")
}

func TestIsFixtureError(t *testing.T) {
	t.Parallel()
	assert.True(t, isFixtureError(`MCP error -32602: unknown tool "test_simple_text"`))
	assert.True(t, isFixtureError("Resource not found"))
	assert.True(t, isFixtureError(`MCP error -32601: method not found: "resources/subscribe"`))
	assert.False(t, isFixtureError("connection reset by peer"))
}

func TestResult_Err(t *testing.T) {
	t.Parallel()
	ok := Result{OK: true}
	assert.NoError(t, ok.Err())

	bad := Result{OK: false, Stages: []Stage{
		{Name: "config", Status: StatusPassed},
		{Name: "test", Status: StatusFailed, Message: "3 of 12 tests failed", Fix: "first failure: TestX (x_test.go:1)"},
	}}
	err := bad.Err()
	require.Error(t, err)
	assert.Equal(t, cli.ExitFailure, cli.ExitCodeOf(err))
	ee := cli.AsExitError(err)
	require.NotNil(t, ee)
	assert.Equal(t, "test", ee.Stage)
	assert.Equal(t, cli.CodeTestsFailed, ee.Code)
	assert.Contains(t, ee.Fix, "x_test.go:1")
}

func TestCheckSDKPin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644))
	}
	write("module x\n\ngo 1.26\n\nrequire github.com/modelcontextprotocol/go-sdk v1.8.0\n")
	assert.NoError(t, checkSDKPin(dir))

	write("module x\n\ngo 1.26\n\nrequire github.com/modelcontextprotocol/go-sdk v1.3.0\n")
	assert.Error(t, checkSDKPin(dir))
}

func TestValidateServerJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "server.json"), []byte(content), 0o644))
	}

	write(`{
		"name": "io.github.acme/weather",
		"description": "Weather server",
		"version": "0.1.0",
		"packages": [{"registryType": "oci", "identifier": "ghcr.io/acme/weather", "version": "0.1.0", "transport": {"type": "stdio"}}]
	}`)
	msg, _, err := validateServerJSON(dir)
	assert.NoError(t, err, msg)

	write(`{"name": "", "version": "not-semver", "packages": [{"registryType": "npm", "identifier": ""}]}`)
	msg, fix, err := validateServerJSON(dir)
	require.Error(t, err)
	assert.Contains(t, msg, "name")
	assert.Contains(t, msg, "semver")
	assert.Contains(t, msg, "oci")
	assert.NotEmpty(t, fix)

	require.NoError(t, os.Remove(filepath.Join(dir, "server.json")))
	_, _, err = validateServerJSON(dir)
	require.Error(t, err)
}

func TestResult_Err_PreservesStageExitCode(t *testing.T) {
	t.Parallel()
	// usage error from the config stage (unknown version) → exit 2
	res := Result{OK: false, Stages: []Stage{
		{Name: "config", Status: StatusFailed, Message: "unsupported mcpgen.json version: got 99",
			Code: cli.CodeConfigVersion, ExitCode: cli.ExitUsage},
	}}
	err := res.Err()
	require.Error(t, err)
	assert.Equal(t, cli.ExitUsage, cli.ExitCodeOf(err))

	// prerequisite failure → exit 6
	res2 := Result{OK: false, Stages: []Stage{
		{Name: "prerequisite", Status: StatusFailed, Message: "go toolchain not found",
			Code: cli.CodePrerequisite, ExitCode: cli.ExitPrerequisite},
	}}
	assert.Equal(t, cli.ExitPrerequisite, cli.ExitCodeOf(res2.Err()))
}
