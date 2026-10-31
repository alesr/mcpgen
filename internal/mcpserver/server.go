// Package mcpserver runs mcpgen itself as an MCP server over stdio, exposing
// the same operations the CLI exposes. It uses the go-sdk version mcpgen
// depends on, so the dogfood path keeps the pinned SDK healthy.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alesr/mcpgen/internal/adder"
	"github.com/alesr/mcpgen/internal/checks"
	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/ops"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the mcpgen MCP server on stdio until ctx is cancelled.
func Serve(ctx context.Context, version string) error {
	return NewServer(version).Run(ctx, &mcp.StdioTransport{})
}

// NewServer builds the mcpgen MCP server exposing mcpgen_new, mcpgen_add,
// mcpgen_check, and mcpgen_schema. Serve runs it on stdio; tests can attach
// any transport.
func NewServer(version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcpgen",
		Version: version,
		Title:   "mcpgen — deterministic Go codegen for MCP servers",
	}, &mcp.ServerOptions{
		Instructions: "mcpgen generates, extends, and verifies Go MCP servers. " +
			"Typical loop: mcpgen_new → implement handlers → mcpgen_check (loop until ok=true) → mcpgen_add. " +
			"All results are JSON envelopes with ok, result, error, next_actions.",
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mcpgen_new",
		Title:       "Generate a new MCP server project",
		Description: "Generate a new Go MCP server project (mcpgen new). Writes files unless dryRun. Runs the validation pipeline unless noCheck.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in NewInput) (*mcp.CallToolResult, NewOutput, error) {
		o, err := in.toOptions(version)
		if err != nil {
			return nil, NewOutput{OK: false, Error: strPtr(err.Error())}, nil
		}
		res, err := ops.New(o)
		return nil, newOutput(res, err), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mcpgen_add",
		Title:       "Add a capability to an existing project",
		Description: "Add a tool, prompt, or resource to an existing mcpgen-generated project (mcpgen add). spec is the parsed JSON spec object.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in AddInput) (*mcp.CallToolResult, AddOutput, error) {
		kind, err := adder.ParseKind(in.Kind)
		if err != nil {
			return nil, AddOutput{OK: false, Error: strPtr(err.Error())}, nil
		}
		specRaw, err := json.Marshal(in.Spec)
		if err != nil {
			return nil, AddOutput{OK: false, Error: strPtr(err.Error())}, nil
		}
		specPath, cleanup, err := writeTempSpec(specRaw)
		if err != nil {
			return nil, AddOutput{OK: false, Error: strPtr(err.Error())}, nil
		}
		defer cleanup()
		res, err := ops.Add(kind, specPath, adder.Options{Dir: in.Dir, Force: in.Force, DryRun: in.DryRun, Version: version})
		out := AddOutput{OK: err == nil}
		if res != nil {
			out.Result = res
		}
		if err != nil {
			out.Error = strPtr(err.Error())
			if ee := cli.AsExitError(err); ee != nil {
				out.ExitCode = ee.ExitCode
				out.Code = string(ee.Code)
				out.Fix = ee.Fix
			}
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mcpgen_check",
		Title:       "Verify a project",
		Description: "Run the mcpgen verification pipeline (mcpgen check). ok=true means green; stages lists every stage status and error.fix carries the first failure file:line.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in CheckInput) (*mcp.CallToolResult, CheckOutput, error) {
		var conf *bool
		switch {
		case in.Conformance != nil:
			conf = in.Conformance
		}
		res := ops.Check(in.Dir, checks.Options{
			Conformance: conf,
			Inspector:   in.Inspector,
			Timeout:     in.TimeoutDuration(),
		})
		out := CheckOutput{OK: res.OK, Result: res}
		if !res.OK {
			if err := res.Err(); err != nil {
				out.Error = strPtr(err.Error())
				if ee := cli.AsExitError(err); ee != nil {
					out.ExitCode = ee.ExitCode
					out.Code = string(ee.Code)
					out.Fix = ee.Fix
				}
			}
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mcpgen_schema",
		Title:       "Machine-readable spec of mcpgen",
		Description: "Emit the mcpgen schema document: commands, flags, exit codes, the JSON envelope, and the mcpgen.json JSON Schema (mcpgen schema).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in SchemaInput) (*mcp.CallToolResult, any, error) {
		doc, err := ops.SchemaDoc(version)
		if err != nil {
			return nil, nil, err
		}
		// Out is `any`, so the SDK omits the output schema and does not
		// validate the payload shape.
		return nil, doc, nil
	})

	return server
}

// ---- input/output types ----

type NewInput struct {
	Dir       string            `json:"dir,omitempty" jsonschema:"Output directory, e.g. ./weather"`
	Name      string            `json:"name,omitempty" jsonschema:"Server name"`
	Module    string            `json:"module,omitempty" jsonschema:"Go module path, e.g. github.com/acme/weather"`
	Version   string            `json:"version,omitempty" jsonschema:"Server version (semver)"`
	Transport string            `json:"transport,omitempty" jsonschema:"stdio or http"`
	Port      int               `json:"port,omitempty" jsonschema:"HTTP port when transport=http"`
	Protocol  string            `json:"protocol,omitempty" jsonschema:"compat or latest-only"`
	Tools     []json.RawMessage `json:"tools,omitempty" jsonschema:"Tool spec objects"`
	Prompts   []json.RawMessage `json:"prompts,omitempty" jsonschema:"Prompt spec objects"`
	Resources []json.RawMessage `json:"resources,omitempty" jsonschema:"Resource spec objects"`
	Features  []string          `json:"features,omitempty" jsonschema:"Opt-in features: tasks, skills, oauth"`
	DryRun    bool              `json:"dryRun,omitempty" jsonschema:"Plan only; write nothing"`
	Force     bool              `json:"force,omitempty" jsonschema:"Allow generating into an existing non-empty directory"`
	NoCheck   bool              `json:"noCheck,omitempty" jsonschema:"Skip the validation pipeline"`
}

func (in NewInput) toOptions(version string) (ops.NewOptions, error) {
	o := ops.NewOptions{
		Dir:              in.Dir,
		Name:             in.Name,
		Module:           in.Module,
		Version:          in.Version,
		Transport:        in.Transport,
		Port:             in.Port,
		Protocol:         in.Protocol,
		DryRun:           in.DryRun,
		Force:            in.Force,
		NoCheck:          in.NoCheck,
		GeneratorVersion: version,
	}
	if o.Dir == "" {
		o.Dir = config.DefaultOutputDir
	}
	if o.Name == "" {
		o.Name = config.DefaultServerName
	}
	if o.Version == "" {
		o.Version = config.DefaultServerVersion
	}
	if o.Transport == "" {
		o.Transport = config.DefaultTransport
	}
	if o.Port == 0 {
		o.Port = config.DefaultHTTPPort
	}
	if o.Protocol == "" {
		o.Protocol = config.ProtocolCompat
	}
	for i, raw := range in.Tools {
		var t config.Tool
		if err := json.Unmarshal(raw, &t); err != nil {
			return o, fmt.Errorf("tools[%d] is not a valid tool spec: %w", i, err)
		}
		o.Tools = append(o.Tools, t)
	}
	for i, raw := range in.Prompts {
		var p config.Prompt
		if err := json.Unmarshal(raw, &p); err != nil {
			return o, fmt.Errorf("prompts[%d] is not a valid prompt spec: %w", i, err)
		}
		o.Prompts = append(o.Prompts, p)
	}
	for i, raw := range in.Resources {
		var r config.Resource
		if err := json.Unmarshal(raw, &r); err != nil {
			return o, fmt.Errorf("resources[%d] is not a valid resource spec: %w", i, err)
		}
		o.Resources = append(o.Resources, r)
	}
	for _, f := range in.Features {
		switch f {
		case "tasks":
			o.Features.Tasks = true
		case "skills":
			o.Features.Skills = true
		case "oauth":
			o.Features.OAuth = true
		default:
			return o, fmt.Errorf("unknown feature %q (expected tasks, skills, oauth)", f)
		}
	}
	return o, nil
}

type NewOutput struct {
	OK           bool     `json:"ok"`
	Dir          string   `json:"dir,omitempty"`
	FilesWritten []string `json:"files_written,omitempty"`
	FilesSkipped []string `json:"files_skipped,omitempty"`
	Plan         any      `json:"plan,omitempty"`
	Check        any      `json:"check,omitempty"`
	Error        *string  `json:"error,omitempty"`
	ExitCode     int      `json:"exit_code,omitempty"`
	Code         string   `json:"code,omitempty"`
	Fix          string   `json:"fix,omitempty"`
	NextActions  []string `json:"next_actions,omitempty"`
}

func newOutput(res *ops.NewResult, err error) NewOutput {
	out := NewOutput{OK: err == nil}
	if res != nil {
		out.Dir = res.Dir
		out.FilesWritten = res.FilesWritten
		out.FilesSkipped = res.FilesSkipped
		if res.Plan != nil {
			out.Plan = mustUnmarshalMap(res.Plan)
		}
		if res.Check != nil {
			out.Check = mustUnmarshalMap(res.Check)
		}
	}
	if err != nil {
		out.Error = strPtr(err.Error())
		if ee := cli.AsExitError(err); ee != nil {
			out.ExitCode = ee.ExitCode
			out.Code = string(ee.Code)
			out.Fix = ee.Fix
		}
	}
	if out.OK {
		out.NextActions = []string{"implement handler stubs", "mcpgen_check until ok=true"}
	} else {
		out.NextActions = []string{"read error.fix and retry"}
	}
	return out
}

type AddInput struct {
	Kind   string         `json:"kind" jsonschema:"tool, prompt, or resource"`
	Spec   map[string]any `json:"spec" jsonschema:"The capability spec object (same shape as the spec file)"`
	Dir    string         `json:"dir,omitempty" jsonschema:"Project root; default ."`
	Force  bool           `json:"force,omitempty"`
	DryRun bool           `json:"dryRun,omitempty"`
}

type AddOutput struct {
	OK       bool          `json:"ok"`
	Result   *adder.Result `json:"result,omitempty"`
	Error    *string       `json:"error,omitempty"`
	ExitCode int           `json:"exit_code,omitempty"`
	Code     string        `json:"code,omitempty"`
	Fix      string        `json:"fix,omitempty"`
}

type CheckInput struct {
	Dir         string `json:"dir,omitempty" jsonschema:"Project root; default ."`
	Conformance *bool  `json:"conformance,omitempty" jsonschema:"Force the conformance stage on/off; default auto"`
	Inspector   bool   `json:"inspector,omitempty" jsonschema:"Run the Inspector v2 smoke check"`
	TimeoutSec  int    `json:"timeoutSec,omitempty" jsonschema:"Overall pipeline timeout in seconds; default 600"`
}

func (in CheckInput) TimeoutDuration() time.Duration {
	secs := in.TimeoutSec
	if secs <= 0 {
		secs = 600
	}
	return time.Duration(secs) * time.Second
}

type CheckOutput struct {
	OK       bool          `json:"ok"`
	Result   checks.Result `json:"result"`
	Error    *string       `json:"error,omitempty"`
	ExitCode int           `json:"exit_code,omitempty"`
	Code     string        `json:"code,omitempty"`
	Fix      string        `json:"fix,omitempty"`
}

type SchemaInput struct{}

func strPtr(s string) *string { return &s }

func mustUnmarshalMap(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func writeTempSpec(spec json.RawMessage) (string, func(), error) {
	if len(spec) == 0 {
		return "", func() {}, fmt.Errorf("spec is required")
	}
	f, err := os.CreateTemp("", "mcpgen-spec-*.json")
	if err != nil {
		return "", func() {}, err
	}
	if _, err := f.Write(spec); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", func() {}, err
	}
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}
