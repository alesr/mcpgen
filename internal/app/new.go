package app

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/ops"
	"github.com/alesr/mcpgen/internal/ui"
	"github.com/charmbracelet/x/term"
)

type newOptions struct {
	dir       string
	name      string
	module    string
	version   string
	transport string
	port      int
	protocol  string
	tools     multiFlag
	prompts   multiFlag
	resources multiFlag
	features  string
	noStubs   bool
	jsonOut   bool
	dryRun    bool
	force     bool
	noCheck   bool
}

// multiFlag collects repeated string flags.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

type newOptionsParsed struct {
	newOptions
	showHelp bool
	setFlags map[string]bool
}

func runNew(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	// Interactive mode: `mcpgen new` with no flags and a TTY on stdin.
	if len(args) == 0 && stdin != nil && term.IsTerminal(stdin.Fd()) {
		return runInteractiveNew(stdin, stdout, stderr)
	}

	opts, err := parseNewOptions(args, stderr)
	if err != nil {
		if opts.showHelp {
			return cli.ExitOK
		}
		return emit(stdout, stderr, false, envelopeErr(err, nil))
	}

	tools, err := ops.LoadSpecs[config.Tool](opts.tools)
	if err != nil {
		return emit(stdout, stderr, opts.jsonOut, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "new", cli.CodeSpecInvalid, "tool: %v", err), nil))
	}
	prompts, err := ops.LoadSpecs[config.Prompt](opts.prompts)
	if err != nil {
		return emit(stdout, stderr, opts.jsonOut, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "new", cli.CodeSpecInvalid, "prompt: %v", err), nil))
	}
	resources, err := ops.LoadSpecs[config.Resource](opts.resources)
	if err != nil {
		return emit(stdout, stderr, opts.jsonOut, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "new", cli.CodeSpecInvalid, "resource: %v", err), nil))
	}

	feats := config.Features{}
	// parse features via a throwaway config so errors carry the right code
	tmp := &config.Config{Features: feats}
	if err := ops.ApplyFeatures(tmp, opts.features); err != nil {
		return emit(stdout, stderr, opts.jsonOut, envelopeErr(err, nil))
	}
	feats = tmp.Features

	res, err := ops.New(ops.NewOptions{
		Dir:              opts.dir,
		Name:             opts.name,
		Module:           opts.module,
		Version:          opts.version,
		Transport:        opts.transport,
		Port:             opts.port,
		Protocol:         opts.protocol,
		Tools:            tools,
		Prompts:          prompts,
		Resources:        resources,
		Features:         feats,
		NoStubs:          opts.noStubs,
		DryRun:           opts.dryRun,
		Force:            opts.force,
		NoCheck:          opts.noCheck,
		SetFlags:         opts.setFlags,
		GeneratorVersion: Version,
	})
	if err != nil {
		next := []string{}
		if res != nil && res.Dir != "" {
			next = append(next, "mcpgen check "+res.Dir+" --json")
		}
		env := envelopeErr(err, next)
		if res != nil && opts.jsonOut {
			env.Result = res
		}
		return emit(stdout, stderr, opts.jsonOut, env)
	}

	next := nextActionsForNew(opts, res)
	if !opts.jsonOut {
		printNewHuman(stdout, res)
	}
	return emit(stdout, stderr, opts.jsonOut, cli.EnvelopeOK(res, next))
}

func parseNewOptions(args []string, out io.Writer) (newOptionsParsed, error) {
	var opts newOptionsParsed
	fs := flag.NewFlagSet("mcpgen new", flag.ContinueOnError)
	fs.SetOutput(out)

	fs.StringVar(&opts.name, "name", config.DefaultServerName, "Server name")
	fs.StringVar(&opts.module, "module", "", "Go module path (default: derived from name)")
	fs.StringVar(&opts.version, "version", config.DefaultServerVersion, "Server version (semver)")
	fs.StringVar(&opts.transport, "transport", config.DefaultTransport, "Transport: stdio|http")
	fs.IntVar(&opts.port, "port", config.DefaultHTTPPort, "HTTP port when transport=http")
	fs.StringVar(&opts.protocol, "protocol", config.ProtocolCompat, "Protocol versions: compat|latest-only")
	fs.Var(&opts.tools, "tool", "Tool spec JSON file (repeatable), optionally name=file.json")
	fs.Var(&opts.prompts, "prompt", "Prompt spec JSON file (repeatable), optionally name=file.json")
	fs.Var(&opts.resources, "resource", "Resource spec JSON file (repeatable), optionally name=file.json")
	fs.StringVar(&opts.features, "features", "", "Comma-separated opt-in features: tasks,skills,oauth (additive on regen; edit mcpgen.json to remove)")
	fs.BoolVar(&opts.noStubs, "no-stubs", false, "Do not add the default tool/prompt/resource stubs")
	fs.BoolVar(&opts.jsonOut, "json", false, "Emit the JSON envelope on stdout")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "Print the plan as JSON; write nothing")
	fs.BoolVar(&opts.force, "force", false, "Allow generating into an existing non-empty directory")
	fs.BoolVar(&opts.noCheck, "no-check", false, "Skip the post-generation validation pipeline")

	fs.Usage = func() {
		_, _ = io.WriteString(out, `Usage: mcpgen new [dir] [flags]

Generate a new MCP server project.

Flags:
`)
		fs.PrintDefaults()
		_, _ = io.WriteString(out, `
Examples:
  mcpgen new ./weather --name weather --module github.com/acme/weather --transport http --json
  mcpgen new ./weather --tool tools/forecast.json --dry-run --json
  mcpgen new            # interactive (TTY)
`)
	}

	// The output dir may appear before the flags (`mcpgen new ./dir --json`);
	// pull a leading positional out so flag.Parse sees only flags.
	flagArgs := args
	dir := ""
	if len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "-") {
		dir = flagArgs[0]
		flagArgs = flagArgs[1:]
	}

	if err := fs.Parse(flagArgs); err != nil {
		if err == flag.ErrHelp {
			opts.showHelp = true
			return opts, nil
		}
		return opts, cli.NewExitError(cli.ExitUsage, "new", cli.CodeUsage, "%v", err)
	}

	if fs.NArg() > 1 || (dir != "" && fs.NArg() > 0) {
		return opts, cli.NewExitError(cli.ExitUsage, "new", cli.CodeUsage, "expected at most one positional argument (the output dir), got: %s", strings.Join(args, " "))
	}
	opts.dir = config.DefaultOutputDir
	if dir != "" {
		opts.dir = dir
	} else if fs.NArg() == 1 {
		opts.dir = fs.Arg(0)
	}
	opts.setFlags = map[string]bool{}
	fs.Visit(func(f *flag.Flag) { opts.setFlags[f.Name] = true })
	return opts, nil
}

func nextActionsForNew(opts newOptionsParsed, res *ops.NewResult) []string {
	if opts.dryRun {
		n := 0
		if res.Plan != nil {
			n = len(res.Plan.Files)
		}
		return []string{fmt.Sprintf("re-run without --dry-run to write the %d planned files into %s", n, res.Dir)}
	}
	na := []string{
		fmt.Sprintf("cd %s", res.Dir),
		fmt.Sprintf("mcpgen check %s --json", res.Dir),
	}
	if res.Dir != "." {
		na = append(na, fmt.Sprintf("make -C %s check", res.Dir))
	}
	return na
}

func printNewHuman(w io.Writer, res *ops.NewResult) {
	_, _ = fmt.Fprintf(w, "Generated MCP server in %s\n", res.Dir)
	_, _ = fmt.Fprintf(w, "  files written: %d", len(res.FilesWritten))
	if len(res.FilesSkipped) > 0 {
		_, _ = fmt.Fprintf(w, ", skipped: %d", len(res.FilesSkipped))
	}
	_, _ = fmt.Fprintln(w)
	if res.Check != nil {
		for _, s := range res.Check.Stages {
			_, _ = fmt.Fprintf(w, "  [%s] %s %s\n", s.Status, s.Name, s.Message)
		}
	}
	_, _ = fmt.Fprintf(w, "Next: cd %s && mcpgen check . --json\n", res.Dir)
}

// runInteractiveNew drives the TUI flow and then the same ops code path.
func runInteractiveNew(stdin *os.File, stdout, stderr io.Writer) int {
	cfg, outDir, runChecks, err := ui.RunInteractive()
	if err != nil {
		return emit(stdout, stderr, false, envelopeErr(err, nil))
	}
	res, err := ops.NewFromConfig(cfg, ops.NewOptions{
		Dir:              outDir,
		NoCheck:          !runChecks,
		GeneratorVersion: Version,
	})
	if err != nil {
		var env cli.Envelope
		if res != nil {
			env = envelopeErr(err, []string{"mcpgen check " + res.Dir + " --json"})
			env.Result = res
		} else {
			env = envelopeErr(err, nil)
		}
		return emit(stdout, stderr, false, env)
	}
	printNewHuman(stdout, res)
	return emit(stdout, stderr, false, cli.EnvelopeOK(res, []string{"cd " + res.Dir, "mcpgen check " + res.Dir + " --json"}))
}
