package checks

import (
	"context"
	"net"
	"net/http"
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
	// Every stage that needs a valid config must skip, not run and fail.
	for _, name := range []string{"fmt", "vet", "build", "test"} {
		assert.Equal(t, StatusSkipped, stages[name].Status, "stage %s must skip on invalid config", name)
	}
	// The config failure carries CONFIG_INVALID with exit 2, matching add/new.
	assert.Equal(t, cli.CodeConfigInvalid, stages["config"].Code)
	assert.Equal(t, cli.ExitUsage, stages["config"].ExitCode)
	err := res.Err()
	require.Error(t, err)
	assert.Equal(t, cli.ExitUsage, cli.ExitCodeOf(err))
	assert.Equal(t, cli.CodeConfigInvalid, cli.AsExitError(err).Code)
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
    Error: MCP error -32602: unknown tool "test_simple_text"

=== Running scenario: resources-subscribe ===
    Error: Resource not found

=== Running scenario: prompts-get-simple ===
=== Failed Checks ===
  - PromptsGetSimple: run

=== Running scenario: tools-list ===
Running client scenario 'tools-list' against server: http://127.0.0.1:1234/mcp

✓ server-initialize: 1 passed, 0 failed
✗ tools-call-simple-text: 0 passed, 1 failed
✗ resources-subscribe: 0 passed, 1 failed
✗ prompts-get-simple: 0 passed, 1 failed
✓ tools-list: 1 passed, 0 failed

Total: 2 passed, 3 failed
`
	s := parseConformance(out)
	assert.Equal(t, 2, s.passed)
	assert.Equal(t, 5, s.total)
	assert.Empty(t, s.gatingFailures, "fixture scenarios must not gate")
	assert.Contains(t, s.fixtureSkipped, "tools-call-simple-text")
	assert.Contains(t, s.fixtureSkipped, "resources-subscribe")
	assert.Contains(t, s.unparsed, "prompts-get-simple", "failures without an Error: line are unclassified, not fixture n/a")
}

func TestParseConformance_UnparsedFailureDoesNotPretendToBeFixture(t *testing.T) {
	t.Parallel()
	// Real suite output format: no per-scenario "Error:" lines at all.
	out := `=== Running scenario: tools-call-image ===
run
✗ tools-call-image: 0 passed, 1 failed
✓ ping: 1 passed, 0 failed
Total: 1 passed, 1 failed
`
	s := parseConformance(out)
	assert.Empty(t, s.gatingFailures)
	assert.Empty(t, s.fixtureSkipped)
	assert.Equal(t, []string{"tools-call-image"}, s.unparsed)
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

func TestFreePort(t *testing.T) {
	t.Parallel()
	p, err := freePort()
	require.NoError(t, err)
	assert.Greater(t, p, 0)
	assert.LessOrEqual(t, p, 65535)
}

func TestAtoiTail(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 42, atoi("42"))
	assert.Equal(t, 0, atoi("bogus"))
	assert.Equal(t, "bc", tail("abc", 2))
	assert.Equal(t, "abc", tail("abc", 10))
}

func TestIsFixtureError_BroadToolPatternRemoved(t *testing.T) {
	t.Parallel()
	// "Tool '" alone is too broad to classify as fixture; require a
	// not-found style message.
	assert.False(t, isFixtureError("Tool 'x' exploded with panic"))
	assert.True(t, isFixtureError("unknown tool \"test_simple_text\""))
	assert.True(t, isFixtureError("Tool 'test_simple_text' not found. Available tools: greet"))
}

func TestStageErrorCode(t *testing.T) {
	t.Parallel()
	want := map[string]cli.ErrorCode{
		"config": cli.CodeConfigInvalid, "tidy": cli.CodeTidyFailed, "fmt": cli.CodeFmtFailed,
		"vet": cli.CodeVetFailed, "build": cli.CodeBuildFailed, "test": cli.CodeTestsFailed,
		"conformance": cli.CodeConformanceFailed, "inspector": cli.CodeInspectorFailed,
		"other": cli.CodeStageFailed,
	}
	for stage, code := range want {
		assert.Equal(t, code, stageErrorCode(stage), stage)
	}
}

func TestSkippedError(t *testing.T) {
	t.Parallel()
	err := errSkipped("config invalid")
	assert.Equal(t, "skipped: config invalid", err.Error())
	var sk skippedError
	assert.ErrorAs(t, err, &sk)
	assert.Equal(t, "config invalid", sk.reason)
}

func TestResult_Err_NoFailedStageFallsBack(t *testing.T) {
	t.Parallel()
	res := Result{OK: false, Stages: []Stage{{Name: "config", Status: StatusPassed}}}
	err := res.Err()
	require.Error(t, err)
	assert.Equal(t, cli.ExitFailure, cli.ExitCodeOf(err))
	assert.Equal(t, cli.CodeStageFailed, cli.AsExitError(err).Code)
}

func TestNpxAvailable(t *testing.T) {
	t.Parallel()
	// Environment-dependent; just exercise the LookPath logic.
	_ = npxAvailable()
}

func TestRunCmd(t *testing.T) {
	t.Parallel()
	out, err := runCmd(context.Background(), t.TempDir(), "go", "version")
	require.NoError(t, err)
	assert.Contains(t, out, "go version")
}

func TestParseConformance_SummaryWithoutScenarioHeaderIgnored(t *testing.T) {
	t.Parallel()
	// A failure summary with no matching "=== Running scenario:" header is
	// counted in totals but never classified.
	out := `=== Running scenario: ping ===
run
✗ ghost-scenario: 0 passed, 1 failed
✓ ping: 1 passed, 0 failed
`
	s := parseConformance(out)
	assert.Equal(t, 1, s.passed)
	assert.Equal(t, 2, s.total)
	assert.Empty(t, s.gatingFailures)
	assert.Empty(t, s.fixtureSkipped)
	assert.Empty(t, s.unparsed)
}

func TestWaitHealthz(t *testing.T) {
	t.Parallel()
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	ln, err := net.Listen("tcp", srv.Addr)
	require.NoError(t, err)
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()
	require.NoError(t, waitHealthz(context.Background(), ln.Addr().String()))

	// Unhealthy server: times out fast via context.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	dead := "127.0.0.1:1"
	assert.Error(t, waitHealthz(ctx, dead))
}
