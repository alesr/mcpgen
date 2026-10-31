package app

import (
	"context"
	"encoding/json"
	"flag"
	"io"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/mcpserver"
)

func runSchema(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcpgen schema", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return cli.ExitOK
		}
		// schema has no --json flag and its success path is the raw doc, so
		// the error path is the only envelope schema ever emits: always JSON.
		return emit(stdout, stderr, true, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "schema", cli.CodeUsage, "%v", err), nil))
	}
	doc, err := cli.BuildSchemaDoc(Version)
	if err != nil {
		return emit(stdout, stderr, true, envelopeErr(err, nil))
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return emit(stdout, stderr, true, envelopeErr(err, nil))
	}
	_, _ = stdout.Write(append(raw, '\n'))
	return cli.ExitOK
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	var jsonOut bool
	fs := flag.NewFlagSet("mcpgen version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&jsonOut, "json", false, "Emit the JSON envelope on stdout")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return cli.ExitOK
		}
		return emit(stdout, stderr, jsonOut, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "version", cli.CodeUsage, "%v", err), nil))
	}
	if jsonOut {
		return emit(stdout, stderr, true, cli.EnvelopeOK(map[string]string{
			"version": Version,
			"tool":    "mcpgen",
		}, nil))
	}
	_, _ = io.WriteString(stdout, "mcpgen v"+Version+"\n")
	return cli.ExitOK
}

func runMCP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcpgen mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return cli.ExitOK
		}
		return emit(stdout, stderr, false, envelopeErr(
			cli.NewExitError(cli.ExitUsage, "mcp", cli.CodeUsage, "%v", err), nil))
	}
	cli.Log(stderr, "mcpgen v%s serving MCP over stdio", Version)
	if err := mcpserver.Serve(context.Background(), Version); err != nil {
		return emit(stdout, stderr, false, envelopeErr(err, nil))
	}
	return cli.ExitOK
}
