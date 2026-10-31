// Package app implements the mcpgen CLI: subcommand dispatch, the JSON
// envelope contract, and semantic exit codes.
package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/charmbracelet/x/term"
)

// Version is the mcpgen release version, stamped into generated file headers,
// the manifest, and `mcpgen version`. It is a var so goreleaser can inject the
// tagged version via -ldflags "-X .../internal/app.Version={{.Version}}".
var Version = "1.0.0"

// Run executes the CLI and returns the process exit code.
func Run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if stdin != nil && term.IsTerminal(stdin.Fd()) {
			return runInteractiveNew(stdin, stdout, stderr)
		}
		printUsage(stderr)
		return cli.ExitUsage
	}

	cmd, rest := args[0], args[1:]

	// Version/help aliases must be checked before the legacy flag-only
	// branch below, which routes every dash-prefixed arg to `new`.
	switch cmd {
	case "version", "--version", "-v":
		return runVersion(rest, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return cli.ExitOK
	}

	// Backwards compatibility with v0.2 flag-only invocation:
	// `mcpgen --name weather ...` behaves like `mcpgen new --name weather ...`.
	if strings.HasPrefix(cmd, "-") {
		return runNew(args, stdin, stdout, stderr)
	}

	switch cmd {
	case "new":
		return runNew(rest, stdin, stdout, stderr)
	case "add":
		return runAdd(rest, stdout, stderr)
	case "check":
		return runCheck(rest, stdout, stderr)
	case "schema":
		return runSchema(rest, stdout, stderr)
	case "mcp":
		return runMCP(rest, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "mcpgen: unknown command %q\n\n", cmd)
		printUsage(stderr)
		return cli.ExitUsage
	}
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, `mcpgen — deterministic Go codegen for spec-compliant MCP servers (v`+Version+`)

Usage:
  mcpgen new [dir] [flags]              Generate a new MCP server project
  mcpgen add <tool|prompt|resource> <spec.json> [flags]
                                        Add a capability to an existing project
  mcpgen check [dir] [flags]            Verify a project (config→…→conformance)
  mcpgen schema                         Machine-readable spec of commands/flags/exit codes
  mcpgen version [--json]               Print version
  mcpgen mcp                            Run mcpgen as an MCP server (stdio)
  mcpgen                                Interactive mode (TTY only)

Run any command with --json for the JSON envelope contract, and see
`+"`mcpgen schema`"+` for the full machine-readable spec.

Exit codes: 0 ok · 1 failure · 2 usage · 3 exists/not-project · 4 permission · 5 conflict · 6 prerequisite
`)
}

// emit writes the envelope: JSON on stdout when jsonOut; on error always the
// structured envelope (plus a human line on stderr). Returns the exit code.
func emit(stdout, stderr io.Writer, jsonOut bool, env cli.Envelope) int {
	if jsonOut || !env.OK {
		if err := cli.WriteEnvelope(stdout, env); err != nil {
			_, _ = fmt.Fprintf(stderr, "mcpgen: could not write envelope: %v\n", err)
		}
	}
	if !env.OK && env.Error != nil {
		_, _ = fmt.Fprintf(stderr, "mcpgen: error: %s\n", env.Error.Message)
		if env.Error.Fix != "" {
			_, _ = fmt.Fprintf(stderr, "mcpgen: fix: %s\n", env.Error.Fix)
		}
		return env.Error.ExitCode
	}
	return cli.ExitOK
}

// envelopeErr builds a failure envelope from err.
func envelopeErr(err error, nextActions []string) cli.Envelope {
	return cli.EnvelopeFromError(err, nextActions)
}

// splitLeadingPositionals pulls leading non-flag args out of args so
// flag.FlagSet.Parse (which stops at the first positional) still sees the
// flags that follow them: `mcpgen new ./dir --json` and `mcpgen add tool
// spec.json --json` both work.
func splitLeadingPositionals(args []string) (pos []string, rest []string) {
	i := 0
	for i < len(args) && !strings.HasPrefix(args[i], "-") {
		pos = append(pos, args[i])
		i++
	}
	return pos, args[i:]
}
