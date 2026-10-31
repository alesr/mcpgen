// Package ops holds mcpgen's core operations (new/add/check/schema) shared by
// the CLI (internal/app) and the MCP server mode (internal/mcpserver), so the
// two surfaces can never diverge.
package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alesr/mcpgen/internal/adder"
	"github.com/alesr/mcpgen/internal/checks"
	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/generator"
)

// NewOptions describes a `mcpgen new` run.
type NewOptions struct {
	Dir       string
	Name      string
	Module    string
	Version   string
	Transport string
	Port      int
	Protocol  string
	Tools     []config.Tool
	Prompts   []config.Prompt
	Resources []config.Resource
	Features  config.Features
	NoStubs   bool
	DryRun    bool
	Force     bool
	NoCheck   bool
	// SetFlags records which CLI flags were explicitly set, so that re-running
	// `new --force` on an existing project applies flags as overrides on top of
	// the existing mcpgen.json instead of resetting it.
	SetFlags map[string]bool
	// GeneratorVersion is stamped into headers/manifest (defaults to "dev").
	GeneratorVersion string
}

// NewResult is the outcome of a new run.
type NewResult struct {
	Dir          string          `json:"dir"`
	FilesWritten []string        `json:"files_written"`
	FilesSkipped []string        `json:"files_skipped"`
	Plan         *generator.Plan `json:"plan,omitempty"`
	Config       string          `json:"config"`
	Manifest     string          `json:"manifest"`
	Check        *checks.Result  `json:"check,omitempty"`
}

// BuildConfig assembles and validates a project config from NewOptions. When
// the target dir is already an mcpgen project, the existing mcpgen.json is
// the base and only explicitly-set flags override it (idempotent regen).
func BuildConfig(o NewOptions) (*config.Config, error) {
	var base *config.Config
	if config.IsProject(o.Dir) {
		existing, err := config.Load(o.Dir)
		if err != nil {
			// The dir claims to be an mcpgen project but its config does not
			// load; do not silently overwrite it with a fresh flag-derived
			// config under --force.
			return nil, cli.NewExitError(cli.ExitUsage, "new", cli.CodeConfigInvalid, "%v", err)
		}
		base = existing
	}

	var cfg *config.Config
	if base != nil {
		cfg = base
		if o.SetFlags["name"] {
			cfg.Server.Name = strings.TrimSpace(o.Name)
		}
		if o.SetFlags["module"] {
			cfg.Server.Module = strings.TrimSpace(o.Module)
		}
		if o.SetFlags["version"] {
			cfg.Server.Version = o.Version
		}
		if o.SetFlags["transport"] {
			cfg.Transport.Type = strings.ToLower(strings.TrimSpace(o.Transport))
		}
		if o.SetFlags["port"] {
			cfg.Transport.HTTPPort = o.Port
		}
		if o.SetFlags["protocol"] {
			cfg.Protocol = strings.ToLower(strings.TrimSpace(o.Protocol))
		}
		if o.SetFlags["features"] {
			// Features are opt-in booleans that can only be enabled via
			// flags (ApplyFeatures never sets false), so a regen merges them
			// into the existing set instead of replacing it. Removal is done
			// by editing mcpgen.json directly.
			cfg.Features.Tasks = cfg.Features.Tasks || o.Features.Tasks
			cfg.Features.Skills = cfg.Features.Skills || o.Features.Skills
			cfg.Features.OAuth = cfg.Features.OAuth || o.Features.OAuth
		}
		if len(o.Tools) > 0 {
			cfg.Tools = o.Tools
		}
		if len(o.Prompts) > 0 {
			cfg.Prompts = o.Prompts
		}
		if len(o.Resources) > 0 {
			cfg.Resources = o.Resources
		}
	} else {
		cfg = &config.Config{
			Version: config.ConfigVersion,
			Server: config.Server{
				Name:    strings.TrimSpace(o.Name),
				Version: o.Version,
				Module:  strings.TrimSpace(o.Module),
			},
			Transport: config.Transport{
				Type:     strings.ToLower(strings.TrimSpace(o.Transport)),
				HTTPPort: o.Port,
			},
			Protocol: strings.ToLower(strings.TrimSpace(o.Protocol)),
			Features: o.Features,
		}

		hasAnySpec := len(o.Tools) > 0 || len(o.Prompts) > 0 || len(o.Resources) > 0
		cfg.Tools = o.Tools
		cfg.Prompts = o.Prompts
		cfg.Resources = o.Resources
		if !hasAnySpec && !o.NoStubs {
			cfg.Tools = []config.Tool{{ID: config.DefaultToolID, ReadOnly: true}}
			cfg.Prompts = []config.Prompt{{ID: config.DefaultPromptID}}
			cfg.Resources = []config.Resource{{ID: config.DefaultResourceID, URI: "file:///" + config.DefaultResourceID, Text: config.DefaultResourceText}}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, cli.NewExitError(cli.ExitUsage, "new", cli.CodeConfigInvalid, "%v", err)
	}
	return cfg, nil
}

// New runs generation (and, unless disabled, the validation pipeline).
func New(o NewOptions) (*NewResult, error) {
	cfg, err := BuildConfig(o)
	if err != nil {
		return nil, err
	}
	return NewFromConfig(cfg, o)
}

// NewFromConfig runs generation from an already-built config.
func NewFromConfig(cfg *config.Config, o NewOptions) (*NewResult, error) {
	version := o.GeneratorVersion
	if version == "" {
		version = "dev"
	}
	gen := &generator.Generator{
		Config:  cfg,
		OutDir:  o.Dir,
		Force:   o.Force,
		DryRun:  o.DryRun,
		Version: version,
	}
	res, err := gen.Run()
	if err != nil {
		return nil, err
	}
	nr := &NewResult{
		Dir:          res.Dir,
		FilesWritten: res.FilesWritten,
		FilesSkipped: res.FilesSkipped,
		Plan:         res.Plan,
		Config:       res.ConfigPath,
		Manifest:     res.ManifestPath,
	}
	if !o.DryRun && !o.NoCheck {
		checkRes := checks.Run(context.Background(), res.Dir, checks.Options{})
		nr.Check = &checkRes
		if !checkRes.OK {
			return nr, checkRes.Err()
		}
	}
	return nr, nil
}

// Add wraps adder.Add.
func Add(kind adder.Kind, specPath string, o adder.Options) (*adder.Result, error) {
	return adder.Add(kind, specPath, o)
}

// Check wraps checks.Run.
func Check(dir string, o checks.Options) checks.Result {
	return checks.Run(context.Background(), dir, o)
}

// SchemaDoc wraps cli.BuildSchemaDoc.
func SchemaDoc(version string) (*cli.SchemaDoc, error) {
	return cli.BuildSchemaDoc(version)
}

// LoadSpecs loads repeated spec flags. Each entry is a path to a JSON file,
// optionally in name=file.json form (name overrides/sets the id).
func LoadSpecs[T any](flags []string) ([]T, error) {
	out := make([]T, 0, len(flags))
	for _, f := range flags {
		name, path := "", f
		if i := strings.Index(f, "="); i >= 0 {
			name, path = f[:i], f[i+1:]
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if name != "" {
			m["id"] = name
		}
		fixed, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal(fixed, &v); err != nil {
			return nil, fmt.Errorf("parse %s into spec: %w", path, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// ApplyFeatures parses a comma-separated feature list into cfg.Features.
func ApplyFeatures(cfg *config.Config, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	for _, f := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "tasks":
			cfg.Features.Tasks = true
		case "skills":
			cfg.Features.Skills = true
		case "oauth":
			cfg.Features.OAuth = true
		case "":
		default:
			return cli.NewExitError(cli.ExitUsage, "new", cli.CodeUsage, "unknown feature %q (expected tasks, skills, oauth)", f)
		}
	}
	return nil
}
