package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/scaffold"
)

type runOptions struct {
	Name          string
	Transport     string
	WithTools     bool
	WithPrompts   bool
	WithResources bool
	NoInspector   bool
	DryRun        bool
	ShowHelp      bool
	HasCLIInput   bool
}

type ConfigRun struct {
	Config *config.Config
	OutDir string
}

func parseRunOptions(args []string, out io.Writer) (runOptions, error) {
	opts := runOptions{}

	fs := flag.NewFlagSet("mcpgen", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&opts.Name, "name", config.DefaultServerName, "Server name")
	fs.StringVar(&opts.Transport, "transport", config.DefaultTransport, "Transport: stdio|http")
	fs.BoolVar(&opts.WithTools, "with-tools", true, "Generate tool stubs")
	fs.BoolVar(&opts.WithPrompts, "with-prompts", true, "Generate prompt stubs")
	fs.BoolVar(&opts.WithResources, "with-resources", true, "Generate resource stubs")
	fs.BoolVar(&opts.NoInspector, "no-inspector", false, "Skip inspector checks")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Print plan without generating files")

	fs.Usage = func() {
		fmt.Fprintln(out, "Usage: mcpgen [flags]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Generate a new Go MCP server interactively or from flags.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		fmt.Fprintln(out, "  mcpgen --name weather --transport stdio")
		fmt.Fprintln(out, "  mcpgen --name weather --transport http --no-inspector")
		fmt.Fprintln(out, "  mcpgen --dry-run --name weather")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Notes:")
		fmt.Fprintln(out, "  - With no flags on a TTY, mcpgen starts interactive mode.")
		fmt.Fprintln(out, "  - --with-tools, --with-prompts, and --with-resources default to true.")
		fmt.Fprintln(out, "  - Inspector checks run only when stdin is a TTY (or in interactive mode).")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			opts.ShowHelp = true
			return opts, nil
		}
		return opts, err
	}

	if fs.NArg() > 0 {
		return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	opts.HasCLIInput = len(args) > 0
	opts.Name = strings.TrimSpace(opts.Name)
	opts.Transport = strings.ToLower(strings.TrimSpace(opts.Transport))

	if opts.Name == "" {
		return opts, errors.New("--name cannot be empty")
	}

	switch opts.Transport {
	case "stdio", "http":
		// valid
	default:
		return opts, fmt.Errorf("invalid --transport %q (expected stdio or http)", opts.Transport)
	}

	return opts, nil
}

func runWithOptions(opts runOptions, canRunInspector bool) (*ConfigRun, bool, bool, error) {
	cfg, outDir := scaffold.DefaultConfig(
		config.DefaultOutputDir,
		opts.Transport,
		config.DefaultHTTPPort,
		opts.WithTools,
		opts.WithResources,
		opts.WithPrompts,
	)

	cfg.Server.Name = opts.Name

	if err := cfg.Validate(); err != nil {
		return nil, false, false, fmt.Errorf("could not validate config: %w", err)
	}

	scaffold.PrintSummary(cfg, outDir)
	shouldTest := canRunInspector && !opts.NoInspector
	return &ConfigRun{Config: cfg, OutDir: outDir}, shouldTest, opts.DryRun, nil
}
