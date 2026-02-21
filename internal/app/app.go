package app

import (
	"fmt"
	"os"

	"github.com/alesr/mcpgen/internal/checks"
	"github.com/alesr/mcpgen/internal/generator"
	"github.com/alesr/mcpgen/internal/inspector"
	"github.com/alesr/mcpgen/internal/scaffold"
	"github.com/alesr/mcpgen/internal/ui"
	"github.com/charmbracelet/x/term"
)

func Run() error {
	fmt.Printf("+---------------------------------------+\n| [ MCPGEN ] Go MCP Server Cookiecutter |\n+---------------------------------------+\n\n")

	opts, err := parseRunOptions(os.Args[1:], os.Stdout)
	if err != nil {
		return err
	}

	if opts.ShowHelp {
		return nil
	}

	var (
		cfg        = (*ConfigRun)(nil)
		shouldTest bool
		dryRun     bool
	)

	if opts.HasCLIInput {
		canRunInspector := term.IsTerminal(os.Stdin.Fd())

		runCfg, runShouldTest, runDryRun, err := runWithOptions(opts, canRunInspector)
		if err != nil {
			return err
		}

		cfg = runCfg
		shouldTest = runShouldTest
		dryRun = runDryRun
	} else {
		runCfg, runOut, runShouldTest, err := ui.RunInteractive()
		if err != nil {
			return err
		}

		cfg = &ConfigRun{Config: runCfg, OutDir: runOut}
		shouldTest = runShouldTest
	}

	if dryRun {
		fmt.Println("Dry run enabled. Skipping generation, checks, and inspector.")
		return nil
	}

	gen := &generator.Generator{Config: cfg.Config, OutDir: cfg.OutDir}
	if err := gen.Run(); err != nil {
		return err
	}

	if err := checks.Run(cfg.OutDir); err != nil {
		return err
	}

	if shouldTest {
		if err := inspector.RunTest(cfg.OutDir, cfg.Config); err != nil {
			return err
		}
	}
	scaffold.PrintInspectorHint(cfg.OutDir, cfg.Config)
	return nil
}
