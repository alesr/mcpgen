package checks

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
)

// runConformance builds the generated server, starts it on an ephemeral
// loopback port, runs the official MCP conformance suite against it, and
// stops it. Only loopback networking is used.
//
// The suite mixes protocol-level scenarios (initialize, ping, list methods,
// dns-rebinding — every spec-compliant server must pass these) with
// capability-fixture scenarios that require well-known test tools/resources
// (test_simple_text, test://static-text, ...). Fixture failures are reported
// but do not gate; protocol failures do.
func runConformance(ctx context.Context, dir string, cfg *config.Config) (string, string, error) {
	serverName := utils.DefaultServerName(cfg.Server.Name)

	bin := filepath.Join(dir, "dist", "mcpgen-check-server")
	if out, err := runCmd(ctx, dir, "go", "build", "-o", bin, "./cmd/"+serverName); err != nil {
		return out, fmt.Sprintf("first error at %s; run `go build ./...` in %s", firstGoLocation(out), dir), err
	}
	defer func() { _ = os.Remove(bin) }()

	port, err := freePort()
	if err != nil {
		return "", "", err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	serverCmd := exec.CommandContext(ctx, bin, "-addr", addr)
	serverCmd.Dir = dir
	serverCmd.Env = append(os.Environ(), "GOWORK=off")
	var serverLog strings.Builder
	serverCmd.Stdout = &serverLog
	serverCmd.Stderr = &serverLog
	if err := serverCmd.Start(); err != nil {
		return "", "", fmt.Errorf("start server: %w", err)
	}
	defer func() { _ = serverCmd.Process.Kill(); _ = serverCmd.Wait() }()

	if err := waitHealthz(ctx, addr); err != nil {
		return serverLog.String(), fmt.Sprintf("server did not become healthy at http://%s/healthz: %v", addr, err), err
	}

	url := fmt.Sprintf("http://%s/mcp", addr)
	out, runErr := runCmd(ctx, dir, "npx", "--yes", "@modelcontextprotocol/conformance", "server", "--url", url, "--verbose")
	summary := parseConformance(out)

	// Fail closed: zero parsed checks means the suite did not run or its
	// output format is no longer recognized — never report that as a pass.
	if summary.total == 0 {
		return out, fmt.Sprintf("conformance output not recognized (0 checks parsed); run `npx @modelcontextprotocol/conformance server --url %s --verbose` against a local server in %s; output tail: %s", url, dir, tail(out, 400)), fmt.Errorf("conformance suite did not run or produced no parseable checks (exit: %v)", runErr)
	}

	if len(summary.gatingFailures) > 0 {
		names := make([]string, 0, len(summary.gatingFailures))
		for _, f := range summary.gatingFailures {
			names = append(names, f.scenario+" ("+f.errMsg+")")
		}
		return out, fmt.Sprintf("failing protocol scenarios: %s; run `npx @modelcontextprotocol/conformance server --url %s --verbose` in %s for details", strings.Join(names, "; "), url, dir), fmt.Errorf("conformance suite failed")
	}

	msg := fmt.Sprintf("conformance passed against %s: %d/%d checks passed", url, summary.passed, summary.total)
	if len(summary.fixtureSkipped) > 0 {
		msg += fmt.Sprintf("; %d fixture scenarios n/a (require test tools/resources: %s)", len(summary.fixtureSkipped), strings.Join(summary.fixtureSkipped, ", "))
	}
	return msg, "", nil
}

// conformanceSummary is the parsed verdict of a conformance run.
type conformanceSummary struct {
	passed         int
	total          int
	fixtureSkipped []string
	gatingFailures []conformanceFailure
}

type conformanceFailure struct {
	scenario string
	errMsg   string
}

var (
	scenarioRe = regexp.MustCompile(`(?m)^=== Running scenario: (\S+) ===`)
	summaryRe  = regexp.MustCompile(`(?m)^(?:✓|✗|-) (\S+): (\d+) passed, (\d+) failed`)
	errLineRe  = regexp.MustCompile(`(?m)^\s+Error: (.*)$`)
)

// fixtureErrPatterns mark failures caused by missing conformance fixture
// capabilities (well-known test tools/resources/prompts or intentionally
// unimplemented optional/deprecated methods), not by protocol violations.
var fixtureErrPatterns = []string{
	"unknown tool",
	"not found. Available tools",
	"Tool '",
	"Resource not found",
	"Prompt not found",
	"unknown prompt",
	"method not found",
	"Method not found",
}

func isFixtureError(msg string) bool {
	low := strings.ToLower(msg)
	for _, p := range fixtureErrPatterns {
		if strings.Contains(low, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// gatingScenarios are the protocol-level scenarios every spec-compliant
// server must pass. All other scenarios require conformance fixture
// capabilities (well-known test tools/resources/prompts) or intentionally
// unimplemented optional/deprecated methods, so they are reported but do
// not gate. Bump this list deliberately as the suite evolves.
var gatingScenarios = map[string]bool{
	"server-initialize":           true,
	"ping":                        true,
	"logging-set-level":           true,
	"tools-list":                  true,
	"prompts-list":                true,
	"resources-list":              true,
	"dns-rebinding-protection":    true,
	"server-sse-polling":          true,
	"server-sse-multiple-streams": true,
}

// parseConformance parses the verbose output of `conformance server`.
func parseConformance(out string) conformanceSummary {
	var s conformanceSummary
	blocks := scenarioRe.Split(out, -1)
	names := scenarioRe.FindAllStringSubmatch(out, -1)

	failed := map[string]bool{}
	for _, m := range summaryRe.FindAllStringSubmatch(out, -1) {
		name, p, f := m[1], atoi(m[2]), atoi(m[3])
		s.passed += p
		s.total += p + f
		if f > 0 {
			failed[name] = true
		}
	}

	for i, n := range names {
		name := n[1]
		if !failed[name] {
			continue
		}
		block := ""
		if i+1 < len(blocks) {
			block = blocks[i+1]
		}
		errs := errLineRe.FindAllStringSubmatch(block, -1)
		firstErr := "failed"
		if len(errs) > 0 {
			firstErr = errs[0][1]
		}

		switch {
		case gatingScenarios[name]:
			s.gatingFailures = append(s.gatingFailures, conformanceFailure{scenario: name, errMsg: firstErr})
		case len(errs) > 0 && !isFixtureError(firstErr):
			// Unknown scenario with a non-fixture error: gate on it to be safe.
			s.gatingFailures = append(s.gatingFailures, conformanceFailure{scenario: name, errMsg: firstErr})
		default:
			s.fixtureSkipped = append(s.fixtureSkipped, name)
		}
	}
	return s
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitHealthz(ctx context.Context, addr string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://%s/healthz", addr)
	deadline := time.Now().Add(15 * time.Second)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s", url)
		}
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
}
