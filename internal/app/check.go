package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alesr/mcpgen/internal/checks"
	"github.com/alesr/mcpgen/internal/cli"
)

func runCheck(args []string, stdout, stderr io.Writer) int {
	var (
		dir           = "."
		jsonOut       bool
		conformance   bool
		noConformance bool
		inspector     bool
		timeout       time.Duration
	)
	fs := flag.NewFlagSet("mcpgen check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&jsonOut, "json", false, "Emit the JSON envelope on stdout")
	fs.BoolVar(&conformance, "conformance", false, "Force the conformance stage on")
	fs.BoolVar(&noConformance, "no-conformance", false, "Disable the conformance stage")
	fs.BoolVar(&inspector, "inspector", false, "Run the Inspector v2 CLI smoke check (debugging aid)")
	fs.DurationVar(&timeout, "timeout", checks.DefaultTimeout, "Overall pipeline timeout")

	fs.Usage = func() {
		_, _ = io.WriteString(stderr, `Usage: mcpgen check [dir] [flags]

Runs the verification pipeline: config → tidy → fmt → vet → build → test →
conformance (auto when node/npx present and transport is http) → inspector
(--inspector). Exit 0 means green.

Flags:
`)
		fs.PrintDefaults()
		_, _ = io.WriteString(stderr, `
Example:
  mcpgen check ./weather --json
`)
	}

	// The project dir may appear before the flags (`mcpgen check ./dir --json`).
	flagArgs := args
	if len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "-") {
		dir = flagArgs[0]
		flagArgs = flagArgs[1:]
	}

	if err := fs.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			return cli.ExitOK
		}
		return emit(stdout, stderr, jsonOut, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "check", cli.CodeUsage, "%v", err), nil))
	}

	if fs.NArg() > 1 || (dir != "." && fs.NArg() > 0) {
		return emit(stdout, stderr, jsonOut, envelopeErr(cli.NewExitError(cli.ExitUsage, "check", cli.CodeUsage,
			"expected at most one positional argument (the project dir), got: %s", strings.Join(fs.Args(), " ")), nil))
	}
	if fs.NArg() == 1 {
		dir = fs.Arg(0)
	}
	if conformance && noConformance {
		return emit(stdout, stderr, jsonOut, envelopeErr(cli.NewExitError(cli.ExitUsage, "check", cli.CodeUsage,
			"--conformance and --no-conformance are mutually exclusive"), nil))
	}

	var conf *bool
	switch {
	case conformance:
		t := true
		conf = &t
	case noConformance:
		f := false
		conf = &f
	}

	res := checks.Run(context.Background(), dir, checks.Options{
		Conformance: conf,
		Inspector:   inspector,
		Timeout:     timeout,
	})

	if !jsonOut {
		for _, s := range res.Stages {
			_, _ = fmt.Fprintf(stdout, "[%s] %s", s.Status, s.Name)
			if s.Message != "" {
				_, _ = fmt.Fprintf(stdout, " — %s", s.Message)
			}
			_, _ = fmt.Fprintln(stdout)
			if s.Status == checks.StatusFailed && s.Fix != "" {
				_, _ = fmt.Fprintf(stdout, "    fix: %s\n", s.Fix)
			}
		}
	}

	next := []string{"fix the failing stage, then re-run: mcpgen check " + dir + " --json"}
	if res.OK {
		next = []string{"green — implement handler logic and extend with: mcpgen add tool <spec.json> --dir " + dir}
	}

	if res.OK {
		return emit(stdout, stderr, jsonOut, cli.EnvelopeOK(res, next))
	}
	return emit(stdout, stderr, jsonOut, envelopeErr(res.Err(), next))
}
