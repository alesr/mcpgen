// Package ui implements the interactive (TUI) flow for `mcpgen new`.
package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/scaffold"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
)

// RunInteractive runs the interactive flow and returns the config, output dir,
// and whether to run the validation pipeline afterwards.
func RunInteractive() (*config.Config, string, bool, error) {
	if !isTTY(os.Stdin) {
		return runNonInteractive()
	}

	state := struct {
		name, version, module, transport, outDir string
		addTool, addRes, addPrompt, runTest      bool
	}{
		name:      config.DefaultServerName,
		version:   config.DefaultServerVersion,
		module:    config.DefaultServerModule,
		transport: config.DefaultTransport,
		outDir:    config.DefaultOutputDir,
		addTool:   true, addRes: true, addPrompt: true, runTest: true,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Server name").Value(&state.name),
			huh.NewInput().Title("Server version").Value(&state.version),
			huh.NewInput().Title("Go module path").Value(&state.module),
			huh.NewSelect[string]().Title("Transport").
				Options(huh.NewOption("stdio", "stdio"), huh.NewOption("http", "http")).
				Value(&state.transport),
		).Title("Server"),

		huh.NewGroup(
			huh.NewConfirm().Title("Add tool?").Value(&state.addTool),
			huh.NewConfirm().Title("Add resource?").Value(&state.addRes),
			huh.NewConfirm().Title("Add prompt?").Value(&state.addPrompt),
		).Title("Features"),

		huh.NewGroup(
			huh.NewInput().Title("Output dir").Value(&state.outDir),
			huh.NewConfirm().Title("Run checks now?").Value(&state.runTest),
		).Title("Run"),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return nil, "", false, fmt.Errorf("could not run form: %w", err)
	}

	port := config.DefaultHTTPPort
	if state.transport == "http" {
		var portInput string
		if err := askPort(&portInput); err != nil {
			return nil, "", false, fmt.Errorf("could not ask for port: %w", err)
		}
		if p, ok := parsePort(portInput); ok {
			port = p
		}
	}

	cfg := config.DefaultConfig(
		state.transport,
		port,
		state.addTool,
		state.addRes,
		state.addPrompt,
	)
	out := state.outDir

	cfg.Server.Name = strings.TrimSpace(state.name)
	cfg.Server.Version = strings.TrimSpace(state.version)
	cfg.Server.Module = strings.TrimSpace(state.module)

	if err := cfg.Validate(); err != nil {
		return nil, "", false, fmt.Errorf("could not validate config: %w", err)
	}

	scaffold.PrintSummary(cfg, out)
	return cfg, out, state.runTest, nil
}

func runNonInteractive() (*config.Config, string, bool, error) {
	fmt.Println("No TTY detected. Using defaults and skipping checks.")

	cfg, out := scaffold.DefaultConfig(
		config.DefaultTransport,
		config.DefaultHTTPPort,
		true, true, true,
	)
	if err := cfg.Validate(); err != nil {
		return nil, "", false, fmt.Errorf("could not validate config: %w", err)
	}

	scaffold.PrintSummary(cfg, out)
	return cfg, out, false, nil
}

func isTTY(file *os.File) bool { return term.IsTerminal(file.Fd()) }

func askPort(value *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("HTTP port").
				Placeholder("8080").
				Value(value),
		),
	).Run()
}

func parsePort(value string) (int, bool) {
	value = strings.TrimSpace(value)
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return config.DefaultHTTPPort, false
	}
	return port, true
}
