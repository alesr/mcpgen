// Package adder implements `mcpgen add tool|prompt|resource`: append a
// capability to mcpgen.json, create the user-owned implementation files, and
// insert registration into the mcpgen marker region of the registration file.
package adder

import (
	"context"
	"encoding/json"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/alesr/mcpgen/internal/checks"
	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/generator"
)

// Kind is the capability kind: tool, prompt, or resource.
type Kind string

const (
	KindTool     Kind = "tool"
	KindPrompt   Kind = "prompt"
	KindResource Kind = "resource"
)

// ParseKind validates a kind string.
func ParseKind(s string) (Kind, error) {
	k := Kind(strings.ToLower(strings.TrimSpace(s)))
	switch k {
	case KindTool, KindPrompt, KindResource:
		return k, nil
	default:
		return "", cli.NewExitError(cli.ExitUsage, "add", cli.CodeUsage, "unknown capability kind %q; expected tool, prompt, or resource", s)
	}
}

// Result is the outcome of an add.
type Result struct {
	Dir          string         `json:"dir"`
	Kind         Kind           `json:"kind"`
	ID           string         `json:"id"`
	FilesWritten []string       `json:"files_written"`
	FilesSkipped []string       `json:"files_skipped"`
	ConfigPath   string         `json:"config"`
	RegisteredIn string         `json:"registered_in"`
	Check        *checks.Result `json:"check,omitempty"`
}

// Options controls add behavior.
type Options struct {
	Dir    string
	Force  bool
	DryRun bool
	// NoCheck skips the post-add validation pipeline.
	NoCheck bool
	// Conformance controls the conformance stage of the post-add pipeline
	// (nil = auto: on when node/npx are present and transport is http).
	Conformance *bool
	// Version is the mcpgen version for file headers.
	Version string
}

// registrationFile maps a kind to its mcpgen-owned registration file.
func registrationFile(kind Kind) string {
	switch kind {
	case KindTool:
		return "internal/mcpapp/tools.go"
	case KindPrompt:
		return "internal/mcpapp/prompts.go"
	case KindResource:
		return "internal/mcpapp/resources.go"
	}
	return ""
}

func region(kind Kind) string {
	switch kind {
	case KindTool:
		return RegionTools
	case KindPrompt:
		return RegionPrompts
	case KindResource:
		return RegionResources
	}
	return ""
}

// Add applies the spec file to the project in opts.Dir as the given kind.
func Add(kind Kind, specPath string, opts Options) (*Result, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}

	dir := opts.Dir
	if !config.IsProject(dir) {
		return nil, cli.NewExitError(cli.ExitExists, "add", cli.CodeNotMCPGenProject,
			"%s is not an mcpgen-generated project (missing %s or %s)",
			dir, config.Path(dir), config.ManifestPath(dir)).WithFix(
			"run `mcpgen new` to create a project, or run this command from the project root")
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return nil, cli.NewExitError(cli.ExitUsage, "add", cli.CodeConfigInvalid, "%v", err)
	}

	res := &Result{
		Dir:        dir,
		Kind:       kind,
		ConfigPath: filepath.ToSlash(filepath.Join(dir, config.FileName)),
	}

	switch kind {
	case KindTool:
		var tool config.Tool
		if err := loadSpec(specPath, &tool); err != nil {
			return nil, err
		}
		if cfg.FindTool(tool.ID) != nil {
			return nil, conflictErr(dir, "tool", tool.ID)
		}
		cfg.Tools = append(cfg.Tools, tool)
		if err := cfg.Validate(); err != nil {
			return nil, cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "invalid tool spec: %v", err)
		}
		res.ID = tool.ID
		if err := addTool(dir, cfg, tool, opts, res); err != nil {
			return nil, err
		}
	case KindPrompt:
		var prompt config.Prompt
		if err := loadSpec(specPath, &prompt); err != nil {
			return nil, err
		}
		if cfg.FindPrompt(prompt.ID) != nil {
			return nil, conflictErr(dir, "prompt", prompt.ID)
		}
		cfg.Prompts = append(cfg.Prompts, prompt)
		if err := cfg.Validate(); err != nil {
			return nil, cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "invalid prompt spec: %v", err)
		}
		res.ID = prompt.ID
		if err := addPrompt(dir, cfg, prompt, opts, res); err != nil {
			return nil, err
		}
	case KindResource:
		var resource config.Resource
		if err := loadSpec(specPath, &resource); err != nil {
			return nil, err
		}
		if cfg.FindResource(resource.ID) != nil {
			return nil, conflictErr(dir, "resource", resource.ID)
		}
		cfg.Resources = append(cfg.Resources, resource)
		if err := cfg.Validate(); err != nil {
			return nil, cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "invalid resource spec: %v", err)
		}
		res.ID = resource.ID
		if err := addResource(dir, cfg, resource, opts, res); err != nil {
			return nil, err
		}
	}

	if opts.DryRun {
		return res, nil
	}

	if err := config.Save(dir, cfg); err != nil {
		return nil, cli.NewExitError(cli.ExitFailure, "add", cli.CodeIO, "write %s: %v", config.FileName, err)
	}
	if err := refreshManifest(dir); err != nil {
		return nil, err
	}

	// Run the validation pipeline (mcpgen check) and report via the result,
	// unless disabled.
	if !opts.NoCheck {
		checkRes := checks.Run(context.Background(), dir, checks.Options{Conformance: opts.Conformance})
		res.Check = &checkRes
		if !checkRes.OK {
			return res, checkRes.Err()
		}
	}
	return res, nil
}

func conflictErr(dir, kind, id string) error {
	return cli.NewExitError(cli.ExitConflict, "add", cli.CodeConflict,
		"%s %q is already registered in %s; fix: choose a different id", kind, id, filepath.Join(dir, config.FileName))
}

func loadSpec(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeIO, "read spec %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "parse spec %s: %v", path, err)
	}
	return nil
}

func addTool(dir string, cfg *config.Config, tool config.Tool, opts Options, res *Result) error {
	data, err := generator.BuildTemplateData(cfg, opts.Version)
	if err != nil {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "%v", err)
	}
	td, ok := findTool(data, tool.ID)
	if !ok {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "tool %q missing after config update", tool.ID)
	}

	handlerPath := "internal/mcpapp/tools/handlers/" + td.FileName
	testPath := strings.TrimSuffix(handlerPath, ".go") + "_test.go"

	if err := writeUserFile(dir, handlerPath, "tool_handler.go.gotmpl", td, opts, res); err != nil {
		return err
	}
	if err := writeUserFile(dir, testPath, "tool_handler_test.go.gotmpl", td, opts, res); err != nil {
		return err
	}

	stmt, err := generator.RenderStatement("stmt_tool.go.gotmpl", td)
	if err != nil {
		return cli.NewExitError(cli.ExitFailure, "add", cli.CodeGenerateFailed, "render registration: %v", err)
	}
	return register(dir, cfg, KindTool, tool.ID, stmt, opts, res)
}

func addPrompt(dir string, cfg *config.Config, prompt config.Prompt, opts Options, res *Result) error {
	data, err := generator.BuildTemplateData(cfg, opts.Version)
	if err != nil {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "%v", err)
	}
	pd, ok := findPrompt(data, prompt.ID)
	if !ok {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "prompt %q missing after config update", prompt.ID)
	}

	base := "internal/mcpapp/prompts/" + pd.FileName
	if err := writeUserFile(dir, base, "prompt.go.gotmpl", pd, opts, res); err != nil {
		return err
	}
	if err := writeUserFile(dir, strings.TrimSuffix(base, ".go")+"_test.go", "prompt_test.go.gotmpl", pd, opts, res); err != nil {
		return err
	}

	stmt, err := generator.RenderStatement("stmt_prompt.go.gotmpl", pd)
	if err != nil {
		return cli.NewExitError(cli.ExitFailure, "add", cli.CodeGenerateFailed, "render registration: %v", err)
	}
	return register(dir, cfg, KindPrompt, prompt.ID, stmt, opts, res)
}

func addResource(dir string, cfg *config.Config, resource config.Resource, opts Options, res *Result) error {
	data, err := generator.BuildTemplateData(cfg, opts.Version)
	if err != nil {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "%v", err)
	}
	rd, ok := findResource(data, resource.ID)
	if !ok {
		return cli.NewExitError(cli.ExitUsage, "add", cli.CodeSpecInvalid, "resource %q missing after config update", resource.ID)
	}

	base := "internal/mcpapp/resources/" + rd.FileName
	if err := writeUserFile(dir, base, "resource.go.gotmpl", rd, opts, res); err != nil {
		return err
	}
	if err := writeUserFile(dir, strings.TrimSuffix(base, ".go")+"_test.go", "resource_test.go.gotmpl", rd, opts, res); err != nil {
		return err
	}

	stmt, err := generator.RenderStatement("stmt_resource.go.gotmpl", rd)
	if err != nil {
		return cli.NewExitError(cli.ExitFailure, "add", cli.CodeGenerateFailed, "render registration: %v", err)
	}
	return register(dir, cfg, KindResource, resource.ID, stmt, opts, res)
}

func findTool(data generator.TemplateData, id string) (generator.ToolData, bool) {
	for _, t := range data.Tools {
		if t.ID == id {
			return t, true
		}
	}
	return generator.ToolData{}, false
}

func findPrompt(data generator.TemplateData, id string) (generator.PromptData, bool) {
	for _, p := range data.Prompts {
		if p.ID == id {
			return p, true
		}
	}
	return generator.PromptData{}, false
}

func findResource(data generator.TemplateData, id string) (generator.ResourceData, bool) {
	for _, r := range data.Resources {
		if r.ID == id {
			return r, true
		}
	}
	return generator.ResourceData{}, false
}

// writeUserFile renders a user-owned file unless it already exists (then it
// is skipped — mcpgen never overwrites user-owned files — unless opts.Force).
func writeUserFile(dir, rel, tmpl string, data any, opts Options, res *Result) error {
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if _, err := os.Stat(full); err == nil && !opts.Force {
		res.FilesSkipped = append(res.FilesSkipped, rel)
		return nil
	}
	if opts.DryRun {
		res.FilesWritten = append(res.FilesWritten, rel)
		return nil
	}
	raw, err := generator.RenderTemplate(tmpl, data)
	if err != nil {
		return cli.NewExitError(cli.ExitFailure, "add", cli.CodeGenerateFailed, "render %s: %v", tmpl, err)
	}
	formatted, err := format.Source(raw)
	if err != nil {
		return cli.NewExitError(cli.ExitFailure, "add", cli.CodeGenerateFailed, "format %s: %v", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return cli.NewExitError(cli.ExitPermission, "add", cli.CodeIO, "mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, formatted, 0o644); err != nil {
		return cli.NewExitError(cli.ExitPermission, "add", cli.CodePermission, "write %s: %v", rel, err)
	}
	res.FilesWritten = append(res.FilesWritten, rel)
	return nil
}

// subPackage maps a kind to the implementation subpackage it registers from.
func subPackage(cfg *config.Config, kind Kind) (path, name string) {
	switch kind {
	case KindTool:
		return cfg.Server.Module + "/internal/mcpapp/tools/handlers", "handlers"
	case KindPrompt:
		return cfg.Server.Module + "/internal/mcpapp/prompts", "prompts"
	default:
		return cfg.Server.Module + "/internal/mcpapp/resources", "resources"
	}
}

// register inserts the statement into the marker region (skipped on dry-run)
// after checking for an in-region conflict, and ensures the implementation
// package import exists.
func register(dir string, cfg *config.Config, kind Kind, id, stmt string, opts Options, res *Result) error {
	regFile := filepath.Join(dir, registrationFile(kind))
	contains, err := RegionContains(regFile, region(kind), id)
	if err != nil {
		return err
	}
	if contains {
		return cli.NewExitError(cli.ExitConflict, "add", cli.CodeConflict,
			"%q already present in the %s marker region of %s; fix: restore the mcpgen:begin/end markers or remove the duplicate",
			id, region(kind), regFile)
	}
	res.RegisteredIn = registrationFile(kind)
	if opts.DryRun {
		return nil
	}
	if err := InsertIntoMarkerRegion(regFile, region(kind), stmt); err != nil {
		return err
	}
	// Tool statements reference json.RawMessage, but an empty tools.go was
	// generated without the encoding/json import — ensure it too.
	if kind == KindTool {
		if err := EnsureImport(regFile, "encoding/json", "json"); err != nil {
			return err
		}
	}
	pkgPath, pkgName := subPackage(cfg, kind)
	return EnsureImport(regFile, pkgPath, pkgName)
}

// refreshManifest recomputes hashes for manifest-tracked files plus the
// registration files, so regeneration can detect user edits.
func refreshManifest(dir string) error {
	m, err := generator.LoadManifest(dir)
	if err != nil {
		return err
	}
	for _, rel := range []string{
		config.FileName,
		"internal/mcpapp/tools.go",
		"internal/mcpapp/prompts.go",
		"internal/mcpapp/resources.go",
	} {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		hash, err := generator.HashFile(full)
		if err != nil {
			continue
		}
		m.Files[rel] = generator.ManifestEntry{Owner: generator.OwnerMCPGen, SHA256: hash}
	}
	// Track user-owned handler/prompt/resource files too.
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, ok := m.Files[rel]; ok {
			return nil
		}
		if strings.HasPrefix(rel, "internal/mcpapp/tools/handlers/") ||
			strings.HasPrefix(rel, "internal/mcpapp/prompts/") ||
			strings.HasPrefix(rel, "internal/mcpapp/resources/") {
			if hash, err := generator.HashFile(path); err == nil {
				m.Files[rel] = generator.ManifestEntry{Owner: generator.OwnerUser, SHA256: hash}
			}
		}
		return nil
	})
	return generator.SaveManifest(dir, m)
}
