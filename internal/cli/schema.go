package cli

import (
	"encoding/json"

	"github.com/alesr/mcpgen/internal/config"
)

// SchemaDoc is the machine-readable self-discovery document emitted by
// `mcpgen schema`. It describes every command, flag, the JSON envelope, the
// exit-code table, and the mcpgen.json JSON Schema.
type SchemaDoc struct {
	Tool             string         `json:"tool"`
	Version          string         `json:"version"`
	Description      string         `json:"description"`
	Envelope         EnvelopeDoc    `json:"envelope"`
	ExitCodes        []ExitCodeDoc  `json:"exit_codes"`
	Commands         []CommandDoc   `json:"commands"`
	ConfigJSONSchema map[string]any `json:"config_json_schema"`
}

type EnvelopeDoc struct {
	Description string `json:"description"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	Fields      any    `json:"fields"`
}

type ExitCodeDoc struct {
	Code    int    `json:"code"`
	Meaning string `json:"meaning"`
}

type CommandDoc struct {
	Name        string    `json:"name"`
	Usage       string    `json:"usage"`
	Description string    `json:"description"`
	Flags       []FlagDoc `json:"flags"`
	Examples    []string  `json:"examples,omitempty"`
}

type FlagDoc struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default,omitempty"`
	Repeat   bool   `json:"repeat,omitempty"`
	Required bool   `json:"required,omitempty"`
	Help     string `json:"help"`
}

// BuildSchemaDoc assembles the schema document. version is the mcpgen version.
func BuildSchemaDoc(version string) (*SchemaDoc, error) {
	raw, err := config.SchemaJSON()
	if err != nil {
		return nil, err
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}

	return &SchemaDoc{
		Tool:        "mcpgen",
		Version:     version,
		Description: "Deterministic Go codegen for spec-compliant MCP servers: bootstrap in seconds, extend forever, verify until green.",
		Envelope: EnvelopeDoc{
			Description: "Every command emits one JSON envelope on stdout when --json is set. Errors are always structured, even without --json. next_actions is always present on success and recoverable failure; fix carries the single most actionable next step (with file:line when known).",
			Stdout:      "JSON envelope (--json) or clean human output; never mixed with logs",
			Stderr:      "progress and log lines",
			Fields: map[string]string{
				"ok":           "bool - true on success",
				"result":       "command-specific payload, null on error",
				"error":        "null, or {stage, code, message, fix, command, exit_code}",
				"next_actions": "array of strings, always present on success and recoverable failure",
			},
		},
		ExitCodes: []ExitCodeDoc{
			{ExitOK, "Success"},
			{ExitFailure, "General failure (build/test/conformance failed)"},
			{ExitUsage, "Usage error (bad flags/args, unknown config version)"},
			{ExitExists, "Target directory exists and not --force / project not mcpgen-generated"},
			{ExitPermission, "Permission denied (filesystem, port)"},
			{ExitConflict, "Conflict (name already registered in project, markers missing)"},
			{ExitPrerequisite, "Prerequisite missing (go toolchain, node/npx for conformance)"},
		},
		Commands: []CommandDoc{
			{
				Name:        "new",
				Usage:       "mcpgen new [dir] [--name] [--module] [--transport stdio|http] [--port] [--protocol compat|latest-only] [--tool spec.json ...] [--prompt spec.json ...] [--resource spec.json ...] [--features tasks,skills,oauth] [--json] [--dry-run] [--force] [--no-check]",
				Description: "Generate a new MCP server project. With no arguments on a TTY, mcpgen new runs interactively. Re-running on an existing mcpgen project requires --force and never overwrites user-modified files.",
				Flags: []FlagDoc{
					{Name: "name", Type: "string", Default: config.DefaultServerName, Help: "Server name"},
					{Name: "module", Type: "string", Default: "derived from name", Help: "Go module path"},
					{Name: "transport", Type: "string", Default: "stdio", Help: "stdio|http"},
					{Name: "port", Type: "int", Default: "8080", Help: "HTTP port when transport=http"},
					{Name: "protocol", Type: "string", Default: "compat", Help: "compat|latest-only (MCP 2026-07-28 only)"},
					{Name: "tool", Type: "string", Repeat: true, Help: "Tool spec JSON file, optionally name=file.json"},
					{Name: "prompt", Type: "string", Repeat: true, Help: "Prompt spec JSON file, optionally name=file.json"},
					{Name: "resource", Type: "string", Repeat: true, Help: "Resource spec JSON file, optionally name=file.json"},
					{Name: "features", Type: "string", Help: "Comma-separated opt-in features: tasks,skills,oauth"},
					{Name: "json", Type: "bool", Help: "Emit the JSON envelope on stdout"},
					{Name: "dry-run", Type: "bool", Help: "Print the full plan (file tree, create/modify/skip, config summary) as JSON; write nothing"},
					{Name: "force", Type: "bool", Help: "Allow generating into an existing directory"},
					{Name: "no-check", Type: "bool", Help: "Skip the post-generation validation pipeline"},
				},
				Examples: []string{
					"mcpgen new ./weather --name weather --module github.com/acme/weather --transport http --json",
					"mcpgen new ./weather --tool tools/forecast.json --json --dry-run",
				},
			},
			{
				Name:        "add",
				Usage:       "mcpgen add tool|prompt|resource <spec.json> [--dir .] [--json] [--dry-run] [--force]",
				Description: "Add a capability to an existing mcpgen-generated project. Appends to mcpgen.json, creates the user-owned handler file, and inserts registration into the mcpgen marker region (AST-aware). Duplicate IDs exit 5.",
				Flags: []FlagDoc{
					{Name: "dir", Type: "string", Default: ".", Help: "Project root (must contain mcpgen.json)"},
					{Name: "json", Type: "bool", Help: "Emit the JSON envelope on stdout"},
					{Name: "dry-run", Type: "bool", Help: "Print the plan as JSON; write nothing"},
					{Name: "force", Type: "bool", Help: "Overwrite an existing user-owned handler file"},
					{Name: "no-check", Type: "bool", Help: "Skip the post-add validation pipeline"},
					{Name: "conformance", Type: "bool", Help: "Force the conformance stage of the post-add pipeline on"},
					{Name: "no-conformance", Type: "bool", Help: "Disable the conformance stage of the post-add pipeline"},
				},
				Examples: []string{
					"mcpgen add tool tools/health.json --dir ./weather --json",
				},
			},
			{
				Name:        "check",
				Usage:       "mcpgen check [dir] [--json] [--conformance] [--no-conformance] [--inspector] [--timeout 10m]",
				Description: "Run the verification pipeline: config, tidy, fmt, vet, build, test, conformance (when node/npx present and transport is http), optional inspector smoke. Exit 0 means green; the JSON envelope lists every stage status and a fix hint with file:line of the first failure.",
				Flags: []FlagDoc{
					{Name: "json", Type: "bool", Help: "Emit the JSON envelope on stdout"},
					{Name: "conformance", Type: "bool", Default: "auto (on when npx present)", Help: "Force the conformance stage on"},
					{Name: "no-conformance", Type: "bool", Help: "Disable the conformance stage"},
					{Name: "inspector", Type: "bool", Help: "Run the Inspector v2 CLI smoke check (debugging aid)"},
					{Name: "timeout", Type: "duration", Default: "10m", Help: "Overall pipeline timeout"},
				},
				Examples: []string{
					"mcpgen check ./weather --json   # loop until exit 0",
				},
			},
			{
				Name:        "schema",
				Usage:       "mcpgen schema",
				Description: "Emit the machine-readable spec of all commands, flags, types, defaults, exit codes, the envelope, and the mcpgen.json JSON Schema. This is the agent self-discovery entry point.",
			},
			{
				Name:        "version",
				Usage:       "mcpgen version [--json]",
				Description: "Print the mcpgen version.",
				Flags:       []FlagDoc{{Name: "json", Type: "bool", Help: "Emit the JSON envelope on stdout"}},
			},
			{
				Name:        "mcp",
				Usage:       "mcpgen mcp",
				Description: "Run mcpgen itself as an MCP server over stdio, exposing mcpgen_new, mcpgen_add, mcpgen_check, and mcpgen_schema for hosts without shell access.",
			},
		},
		ConfigJSONSchema: schema,
	}, nil
}
