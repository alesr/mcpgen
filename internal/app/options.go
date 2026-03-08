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
	Name            string
	Transport       string
	WithTools       bool
	WithPrompts     bool
	WithResources   bool
	WithElicitation bool
	NoInspector     bool
	ShowHelp        bool
	HasCLIInput     bool
}

type ConfigRun struct {
	Config *config.Config
	OutDir string
}

func parseRunOptions(args []string, out io.Writer) (runOptions, error) {
	var opts runOptions

	fs := flag.NewFlagSet("mcpgen", flag.ContinueOnError)

	fs.SetOutput(out)
	fs.StringVar(&opts.Name, "name", config.DefaultServerName, "Server name")
	fs.StringVar(&opts.Transport, "transport", config.DefaultTransport, "Transport: stdio|http")
	fs.BoolVar(&opts.WithTools, "with-tools", true, "Generate tool stub")
	fs.BoolVar(&opts.WithPrompts, "with-prompts", true, "Generate prompt stub")
	fs.BoolVar(&opts.WithResources, "with-resources", true, "Generate resource stub")
	fs.BoolVar(&opts.WithElicitation, "with-elicitation", false, "Generate elicitation example in tool handlers")
	fs.BoolVar(&opts.NoInspector, "no-inspector", false, "Skip inspector checks")

	fs.Usage = func() {
		_, _ = io.WriteString(out, `Usage: mcpgen [flags]

Generate a new Go MCP server interactively or from flags.

Flags:
`)
		fs.PrintDefaults()
		_, _ = io.WriteString(out, `
Examples:
  mcpgen --name weather --transport stdio
  mcpgen --name weather --transport http --no-inspector

Notes:
  - With no flags on a TTY, mcpgen starts interactive mode.
  - --with-tools, --with-prompts, and --with-resources default to true.
  - --with-elicitation requires --with-tools=true.
  - Inspector checks run only when stdin is a TTY (or in interactive mode).
`)
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

	if opts.WithElicitation && !opts.WithTools {
		return opts, errors.New("--with-elicitation requires --with-tools=true")
	}
	return opts, nil
}

func runWithOptions(opts runOptions, canRunInspector bool) (*ConfigRun, bool, error) {
	if opts.WithElicitation && !opts.WithTools {
		return nil, false, errors.New("--with-elicitation requires --with-tools=true")
	}

	cfg, outDir := scaffold.DefaultConfig(
		config.DefaultOutputDir,
		opts.Transport,
		config.DefaultHTTPPort,
		opts.WithTools,
		opts.WithResources,
		opts.WithPrompts,
		opts.WithElicitation,
	)

	cfg.Server.Name = opts.Name

	if err := cfg.Validate(); err != nil {
		return nil, false, fmt.Errorf("could not validate config: %w", err)
	}

	scaffold.PrintSummary(cfg, outDir)
	shouldTest := canRunInspector && !opts.NoInspector
	return &ConfigRun{Config: cfg, OutDir: outDir}, shouldTest, nil
}
