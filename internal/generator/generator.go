package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
)

// Actions a plan entry can take.
const (
	ActionCreate    = "create"
	ActionOverwrite = "overwrite"
	ActionSkip      = "skip"
)

// FilePlan describes what would happen (or happened) to one output file.
type FilePlan struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Owner  string `json:"owner"`
	Reason string `json:"reason,omitempty"`
}

// Plan is the deterministic build plan, also the --dry-run output.
type Plan struct {
	Dir   string     `json:"dir"`
	Files []FilePlan `json:"files"`
}

// Result is the outcome of a generation run.
type Result struct {
	Dir          string   `json:"dir"`
	FilesWritten []string `json:"files_written"`
	FilesSkipped []string `json:"files_skipped"`
	Plan         *Plan    `json:"plan,omitempty"`
	ConfigPath   string   `json:"config"`
	ManifestPath string   `json:"manifest"`
}

type targetFile struct {
	dest     string // relative to OutDir, slash-separated
	owner    string
	template string // embedded template name, empty for computed content
	data     any    // template data (TemplateData or a single item)
	gofmt    bool
	content  []byte // computed content when template == ""
}

// Generator renders a project from a config.
type Generator struct {
	Config  *config.Config
	OutDir  string
	Force   bool
	DryRun  bool
	Version string
}

// Run executes (or, with DryRun, only plans) the generation.
func (g *Generator) Run() (*Result, error) {
	if g.Config == nil {
		return nil, cli.NewExitError(cli.ExitUsage, "generate", cli.CodeConfigInvalid, "config is nil")
	}
	if err := g.Config.Validate(); err != nil {
		return nil, cli.NewExitError(cli.ExitUsage, "config", cli.CodeConfigInvalid, "invalid config: %v", err)
	}
	if g.OutDir == "" {
		return nil, cli.NewExitError(cli.ExitUsage, "generate", cli.CodeUsage, "output dir is required")
	}
	if unsafe, err := isUnsafeOutDir(g.OutDir); err != nil || unsafe {
		return nil, cli.NewExitError(cli.ExitUsage, "generate", cli.CodeUsage, "output dir must not be the current directory or filesystem root")
	}
	if g.Version == "" {
		g.Version = "dev"
	}

	if err := g.checkTargetDir(); err != nil {
		return nil, err
	}

	data, err := BuildTemplateData(g.Config, g.Version)
	if err != nil {
		return nil, cli.NewExitError(cli.ExitUsage, "generate", cli.CodeConfigInvalid, "%v", err)
	}

	targets, err := g.buildTargets(data)
	if err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(g.OutDir)
	if err != nil {
		return nil, err
	}

	plan, actions, err := g.plan(targets, manifest)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Dir:          g.OutDir,
		Plan:         plan,
		ConfigPath:   filepath.ToSlash(filepath.Join(g.OutDir, config.FileName)),
		ManifestPath: filepath.ToSlash(config.ManifestPath(g.OutDir)),
	}

	if g.DryRun {
		return result, nil
	}

	for _, t := range targets {
		action, ok := actions[t.dest]
		if !ok || action == ActionSkip {
			if ok {
				result.FilesSkipped = append(result.FilesSkipped, t.dest)
			}
			continue
		}
		full := filepath.Join(g.OutDir, filepath.FromSlash(t.dest))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, cli.NewExitError(cli.ExitPermission, "generate", cli.CodeIO, "create dir for %s: %v", t.dest, err)
		}
		if err := os.WriteFile(full, t.content, 0o644); err != nil {
			if os.IsPermission(err) {
				return nil, cli.NewExitError(cli.ExitPermission, "generate", cli.CodePermission, "write %s: %v", t.dest, err)
			}
			return nil, cli.NewExitError(cli.ExitFailure, "generate", cli.CodeIO, "write %s: %v", t.dest, err)
		}
		result.FilesWritten = append(result.FilesWritten, t.dest)
	}

	// Persist mcpgen.json (single source of truth) and the manifest last.
	if err := config.Save(g.OutDir, g.Config); err != nil {
		return nil, cli.NewExitError(cli.ExitFailure, "generate", cli.CodeIO, "write %s: %v", config.FileName, err)
	}
	result.FilesWritten = append(result.FilesWritten, config.FileName)

	// Prune orphaned entries: a manifest entry whose file is gone from disk at
	// this point (the write loop above has already recreated every targeted
	// file) is stale — e.g. the handler of a capability removed from
	// mcpgen.json whose file the user deleted before regenerating.
	for dest := range manifest.Files {
		if dest == config.FileName {
			continue
		}
		if _, err := os.Stat(filepath.Join(g.OutDir, filepath.FromSlash(dest))); os.IsNotExist(err) {
			delete(manifest.Files, dest)
		}
	}

	cfgHash, _ := HashFile(config.Path(g.OutDir))
	manifest.Generator = "mcpgen"
	manifest.Version = g.Version
	if manifest.Files == nil {
		manifest.Files = map[string]ManifestEntry{}
	}
	manifest.Files[config.FileName] = ManifestEntry{Owner: OwnerMCPGen, SHA256: cfgHash}
	for _, t := range targets {
		if actions[t.dest] == ActionSkip {
			// Skipped files keep their previous entry: rehashing a modified
			// file here would reset the baseline and let the next regen
			// overwrite the user's edit.
			continue
		}
		full := filepath.Join(g.OutDir, filepath.FromSlash(t.dest))
		hash, err := HashFile(full)
		if err != nil {
			continue
		}
		manifest.Files[t.dest] = ManifestEntry{Owner: t.owner, SHA256: hash}
	}
	if err := SaveManifest(g.OutDir, manifest); err != nil {
		return nil, cli.NewExitError(cli.ExitFailure, "generate", cli.CodeIO, "write manifest: %v", err)
	}

	sort.Strings(result.FilesWritten)
	sort.Strings(result.FilesSkipped)
	return result, nil
}

// checkTargetDir enforces exit code 3 when the target exists, is non-empty,
// and --force was not given.
func (g *Generator) checkTargetDir() error {
	info, err := os.Stat(g.OutDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return cli.NewExitError(cli.ExitFailure, "generate", cli.CodeIO, "stat %s: %v", g.OutDir, err)
	}
	if !info.IsDir() {
		return cli.NewExitError(cli.ExitExists, "generate", cli.CodeDirExists,
			"target %s exists and is not a directory; fix: choose another directory or remove it", g.OutDir)
	}
	entries, err := os.ReadDir(g.OutDir)
	if err != nil {
		return cli.NewExitError(cli.ExitPermission, "generate", cli.CodePermission, "read %s: %v", g.OutDir, err)
	}
	if len(entries) == 0 {
		return nil
	}
	if !g.Force {
		return cli.NewExitError(cli.ExitExists, "generate", cli.CodeDirExists,
			"target directory %s exists and is not empty", g.OutDir).
			WithFix("pass --force to regenerate an existing mcpgen project (user-modified files are never overwritten), or choose a new directory")
	}
	return nil
}

// buildTargets renders every output file into memory.
func (g *Generator) buildTargets(data TemplateData) ([]targetFile, error) {
	serverName := data.ServerName

	targets := []targetFile{
		{dest: "go.mod", owner: OwnerMCPGen, template: "go.mod.gotmpl", data: data},
		{dest: ".gitignore", owner: OwnerMCPGen, template: "gitignore.gotmpl", data: data},
		{dest: "README.md", owner: OwnerMCPGen, template: "README.md.gotmpl", data: data},
		{dest: "AGENTS.md", owner: OwnerMCPGen, template: "AGENTS.md.gotmpl", data: data},
		{dest: "Makefile", owner: OwnerMCPGen, template: "Makefile.gotmpl", data: data},
		{dest: "Dockerfile", owner: OwnerMCPGen, template: "Dockerfile.gotmpl", data: data},
		{dest: ".github/workflows/ci.yml", owner: OwnerMCPGen, template: "ci.yml.gotmpl", data: data},
		{dest: "server.json", owner: OwnerMCPGen, template: "server.json.gotmpl", data: data},
		{dest: "cmd/" + serverName + "/main.go", owner: OwnerMCPGen, template: "cmd_main.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/mcpapp.go", owner: OwnerMCPGen, template: "mcpapp.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/instructions.go", owner: OwnerMCPGen, template: "instructions.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/tools.go", owner: OwnerMCPGen, template: "tools.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/prompts.go", owner: OwnerMCPGen, template: "prompts.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/resources.go", owner: OwnerMCPGen, template: "resources.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/tools/handlers/doc.go", owner: OwnerMCPGen, template: "doc_handlers.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/prompts/doc.go", owner: OwnerMCPGen, template: "doc_prompts.go.gotmpl", data: data, gofmt: true},
		{dest: "internal/mcpapp/resources/doc.go", owner: OwnerMCPGen, template: "doc_resources.go.gotmpl", data: data, gofmt: true},
	}

	for _, t := range data.Tools {
		base := "internal/mcpapp/tools/handlers/" + t.FileName
		targets = append(targets,
			targetFile{dest: base, owner: OwnerUser, template: "tool_handler.go.gotmpl", data: t, gofmt: true},
			targetFile{dest: strings.TrimSuffix(base, ".go") + "_test.go", owner: OwnerUser, template: "tool_handler_test.go.gotmpl", data: t, gofmt: true},
		)
	}
	for _, p := range data.Prompts {
		base := "internal/mcpapp/prompts/" + p.FileName
		targets = append(targets,
			targetFile{dest: base, owner: OwnerUser, template: "prompt.go.gotmpl", data: p, gofmt: true},
			targetFile{dest: strings.TrimSuffix(base, ".go") + "_test.go", owner: OwnerUser, template: "prompt_test.go.gotmpl", data: p, gofmt: true},
		)
	}
	for _, r := range data.Resources {
		base := "internal/mcpapp/resources/" + r.FileName
		targets = append(targets,
			targetFile{dest: base, owner: OwnerUser, template: "resource.go.gotmpl", data: r, gofmt: true},
			targetFile{dest: strings.TrimSuffix(base, ".go") + "_test.go", owner: OwnerUser, template: "resource_test.go.gotmpl", data: r, gofmt: true},
		)
	}

	if data.Features.Tasks {
		targets = append(targets, targetFile{dest: "internal/mcpapp/tasks.go", owner: OwnerMCPGen, template: "tasks.go.gotmpl", data: data, gofmt: true})
	}
	if data.Features.Skills {
		targets = append(targets, targetFile{dest: "internal/mcpapp/skills.go", owner: OwnerMCPGen, template: "skills.go.gotmpl", data: data, gofmt: true})
	}
	if data.Features.OAuth {
		targets = append(targets, targetFile{dest: "internal/mcpapp/oauth.go", owner: OwnerMCPGen, template: "oauth.go.gotmpl", data: data, gofmt: true})
	}

	for i := range targets {
		content, err := renderTarget(&targets[i], data)
		if err != nil {
			return nil, cli.NewExitError(cli.ExitFailure, "generate", cli.CodeGenerateFailed, "%v", err)
		}
		targets[i].content = content
	}
	return targets, nil
}

func renderTarget(t *targetFile, data TemplateData) ([]byte, error) {
	raw, err := RenderTemplate(t.template, t.data)
	if err != nil {
		return nil, err
	}
	if t.gofmt {
		formatted, err := format.Source(raw)
		if err != nil {
			return nil, fmt.Errorf("format %s: %w", t.dest, err)
		}
		return formatted, nil
	}
	return raw, nil
}

// plan decides create/overwrite/skip per target using the manifest, and
// renders the Plan for --dry-run. actions maps dest -> action for execution.
func (g *Generator) plan(targets []targetFile, manifest *Manifest) (*Plan, map[string]string, error) {
	plan := &Plan{Dir: g.OutDir}
	actions := map[string]string{}

	for _, t := range targets {
		full := filepath.Join(g.OutDir, filepath.FromSlash(t.dest))
		action, reason := g.decide(t, full, manifest)
		plan.Files = append(plan.Files, FilePlan{Path: t.dest, Action: action, Owner: t.owner, Reason: reason})
		actions[t.dest] = action
	}

	// mcpgen.json and the manifest are written outside the targets loop (see
	// Run), but they are part of the output: include them so --dry-run shows
	// the full plan. Both are always rewritten, so the action is create when
	// absent and overwrite when present.
	for _, dest := range []string{config.FileName, config.ManifestDir + "/" + config.ManifestFile} {
		action := ActionCreate
		if _, err := os.Stat(filepath.Join(g.OutDir, filepath.FromSlash(dest))); err == nil {
			action = ActionOverwrite
		}
		plan.Files = append(plan.Files, FilePlan{Path: dest, Action: action, Owner: OwnerMCPGen})
		actions[dest] = action
	}
	return plan, actions, nil
}

func (g *Generator) decide(t targetFile, full string, manifest *Manifest) (action, reason string) {
	current, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return ActionCreate, ""
		}
		return ActionSkip, "unreadable: " + err.Error()
	}

	entry, tracked := manifest.Files[t.dest]
	switch {
	case t.owner == OwnerUser:
		return ActionSkip, "user-owned file already exists"
	case tracked && entry.Owner == OwnerMCPGen:
		hash, err := hashBytes(current)
		if err == nil && hash == entry.SHA256 {
			return ActionOverwrite, ""
		}
		return ActionSkip, "modified since generation (hash mismatch)"
	default:
		// Untracked existing file: never clobber foreign content.
		if string(current) == string(t.content) {
			return ActionSkip, "already up to date"
		}
		return ActionSkip, "exists and is not managed by mcpgen"
	}
}

// isUnsafeOutDir blocks locations where writing would be dangerous.
func isUnsafeOutDir(outDir string) (bool, error) {
	cleaned := filepath.Clean(outDir)
	if cleaned == "." || cleaned == ".." || cleaned == string(filepath.Separator)+".." {
		return true, nil
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return true, err
	}
	if abs == filepath.Dir(abs) {
		return true, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return true, err
	}
	return abs == cwd, nil
}

func hashBytes(b []byte) (string, error) {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
