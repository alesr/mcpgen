package app

import (
	"fmt"

	"github.com/alesr/mcpgen/internal/checks"
	"github.com/alesr/mcpgen/internal/generator"
	"github.com/alesr/mcpgen/internal/inspector"
	"github.com/alesr/mcpgen/internal/scaffold"
	"github.com/alesr/mcpgen/internal/ui"
)

func Run() error {
	fmt.Printf("+---------------------------------------+\n| [ MCPGEN ] Go MCP Server Cookiecutter |\n+---------------------------------------+\n\n")

	cfg, out, shouldTest, err := ui.RunInteractive()
	if err != nil {
		return err
	}

	gen := &generator.Generator{Config: cfg, OutDir: out}
	if err := gen.Run(); err != nil {
		return err
	}

	if err := checks.Run(out); err != nil {
		return err
	}

	if shouldTest {
		if err := inspector.RunTest(out, cfg); err != nil {
			return err
		}
	}
	scaffold.PrintInspectorHint(out, cfg)
	return nil
}
