package app

import (
	"flag"
	"io"
	"strings"

	"github.com/alesr/mcpgen/internal/adder"
	"github.com/alesr/mcpgen/internal/cli"
)

func runAdd(args []string, stdout, stderr io.Writer) int {
	var (
		dir           string
		jsonOut       bool
		dryRun        bool
		force         bool
		noCheck       bool
		conformance   bool
		noConformance bool
	)
	fs := flag.NewFlagSet("mcpgen add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&dir, "dir", ".", "Project root (must contain mcpgen.json)")
	fs.BoolVar(&jsonOut, "json", false, "Emit the JSON envelope on stdout")
	fs.BoolVar(&dryRun, "dry-run", false, "Print the plan as JSON; write nothing")
	fs.BoolVar(&force, "force", false, "Overwrite an existing user-owned handler file")
	fs.BoolVar(&noCheck, "no-check", false, "Skip the post-add validation pipeline")
	fs.BoolVar(&conformance, "conformance", false, "Force the conformance stage of the post-add pipeline on")
	fs.BoolVar(&noConformance, "no-conformance", false, "Disable the conformance stage of the post-add pipeline")

	fs.Usage = func() {
		_, _ = io.WriteString(stderr, `Usage: mcpgen add <tool|prompt|resource> <spec.json> [flags]

Adds a capability to an existing mcpgen-generated project: appends to
mcpgen.json, creates the user-owned implementation files, and inserts
registration into the mcpgen marker region (AST-aware).

Flags:
`)
		fs.PrintDefaults()
		_, _ = io.WriteString(stderr, `
Example:
  mcpgen add tool tools/health.json --dir ./weather --json
`)
	}

	leading, flagArgs := splitLeadingPositionals(args)

	if err := fs.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			return cli.ExitOK
		}
		return emit(stdout, stderr, jsonOut, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "add", cli.CodeUsage, "%v", err), nil))
	}

	pos := append(leading, fs.Args()...)
	if len(pos) != 2 {
		return emit(stdout, stderr, jsonOut, envelopeErr(cli.NewExitError(cli.ExitUsage, "add", cli.CodeUsage,
			"expected exactly 2 arguments: <tool|prompt|resource> <spec.json>, got: %s", strings.Join(pos, " ")), nil))
	}

	kind, err := adder.ParseKind(pos[0])
	if err != nil {
		return emit(stdout, stderr, jsonOut, envelopeErr(err, nil))
	}

	var conf *bool
	switch {
	case conformance && noConformance:
		return emit(stdout, stderr, jsonOut, envelopeErr(cli.NewExitError(cli.ExitUsage, "add", cli.CodeUsage,
			"--conformance and --no-conformance are mutually exclusive"), nil))
	case conformance:
		t := true
		conf = &t
	case noConformance:
		f := false
		conf = &f
	}

	res, err := adder.Add(kind, pos[1], adder.Options{
		Dir:         dir,
		Force:       force,
		DryRun:      dryRun,
		NoCheck:     noCheck,
		Conformance: conf,
		Version:     Version,
	})
	if err != nil {
		env := envelopeErr(err, []string{"mcpgen check " + dir + " --json"})
		if res != nil && jsonOut {
			env.Result = res
		}
		return emit(stdout, stderr, jsonOut, env)
	}

	next := []string{
		"implement the stub in " + handlerHint(kind, res),
		"mcpgen check " + res.Dir + " --json",
	}
	if !jsonOut {
		cli.Log(stderr, "added %s %q in %s (files written: %d, skipped: %d)",
			res.Kind, res.ID, res.Dir, len(res.FilesWritten), len(res.FilesSkipped))
	}
	return emit(stdout, stderr, jsonOut, cli.EnvelopeOK(res, next))
}

func handlerHint(kind adder.Kind, res *adder.Result) string {
	switch kind {
	case adder.KindTool:
		return "internal/mcpapp/tools/handlers/"
	case adder.KindPrompt:
		return "internal/mcpapp/prompts/"
	default:
		return "internal/mcpapp/resources/"
	}
}
