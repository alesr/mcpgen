package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	"github.com/alesr/mcpgen/internal/config"
)

type Generator struct {
	Config *config.Config
	OutDir string
}

func (g *Generator) Run() error {
	if err := g.validate(); err != nil {
		return fmt.Errorf("could not validate config: %w", err)
	}

	if err := g.cleanupGenerated(); err != nil {
		return fmt.Errorf("could not cleanup generated files: %w", err)
	}

	serverName := g.serverName()
	if err := g.ensureOutputDirs(serverName); err != nil {
		return fmt.Errorf("could not ensure output dirs: %w", err)
	}

	data := buildTemplateData(g.Config, serverName)
	if err := g.writeCoreTemplates(serverName, data); err != nil {
		return fmt.Errorf("could not write core templates: %w", err)
	}
	return g.writeOptionalTemplates(data)
}

func (g *Generator) validate() error {
	if g.Config == nil {
		return fmt.Errorf("config is nil")
	}

	if g.OutDir == "" {
		return fmt.Errorf("out dir is required")
	}
	return nil
}

func (g *Generator) serverName() string {
	serverName := goFileName(g.Config.Server.Name)
	if serverName == "" {
		return "mcp"
	}
	return serverName
}

func (g *Generator) ensureOutputDirs(serverName string) error {
	paths := []string{
		filepath.Join(g.OutDir, "cmd", serverName),
		filepath.Join(g.OutDir, "internal", "mcpapp"),
		filepath.Join(g.OutDir, "internal", "mcpapp", "tools"),
		filepath.Join(g.OutDir, "internal", "mcpapp", "tools", "handlers"),
		filepath.Join(g.OutDir, "internal", "mcpapp", "prompts"),
		filepath.Join(g.OutDir, "internal", "mcpapp", "resources"),
		filepath.Join(g.OutDir, "internal", "mcpapp", "stubs"),
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("could not create directory %s: %w", p, err)
		}
	}
	return nil
}

func (g *Generator) writeCoreTemplates(serverName string, data TemplateData) error {
	type v struct {
		src       string
		dest      string
		overwrite bool
	}

	jobs := []v{
		{"go.mod.gotmpl", "go.mod", false},
		{"README.md.gotmpl", "README.md", false},
		{"cmd_main.go.gotmpl", filepath.Join("cmd", serverName, "main.go"), true},
		{"instructions.go.gotmpl", filepath.Join("internal", "mcpapp", "instructions.go"), true},
		{"mcpapp.go.gotmpl", filepath.Join("internal", "mcpapp", "mcpapp.go"), true},
		{"tools.go.gotmpl", filepath.Join("internal", "mcpapp", "tools", "tools.go"), true},
		{"handlers.go.gotmpl", filepath.Join("internal", "mcpapp", "tools", "handlers", "handlers.go"), true},
		{"prompts.go.gotmpl", filepath.Join("internal", "mcpapp", "prompts", "prompts.go"), true},
		{"resources.go.gotmpl", filepath.Join("internal", "mcpapp", "resources", "resources.go"), true},
		{"stubs.go.gotmpl", filepath.Join("internal", "mcpapp", "stubs", "stubs.go"), true},
	}

	for _, j := range jobs {
		fullPath := filepath.Join(g.OutDir, j.dest)
		if err := g.writeTemplate(j.src, fullPath, data, j.overwrite); err != nil {
			return fmt.Errorf("could not write template %s: %w", j.src, err)
		}
	}
	return nil
}

func (g *Generator) writeOptionalTemplates(data TemplateData) error {
	type optionalJob struct {
		src         string
		dest        string
		shouldWrite bool
	}

	jobs := []optionalJob{
		{"handlers_test.go.gotmpl", "internal/mcpapp/tools/handlers/handlers_test.go", len(data.Tools) > 0},
		{"prompts_test.go.gotmpl", "internal/mcpapp/prompts/prompts_test.go", len(data.Prompts) > 0},
		{"resources_test.go.gotmpl", "internal/mcpapp/resources/resources_test.go", len(data.Resources) > 0},
	}

	for _, j := range jobs {
		if !j.shouldWrite {
			continue
		}

		fullPath := filepath.Join(g.OutDir, j.dest)
		if err := g.writeTemplate(j.src, fullPath, data, true); err != nil {
			return fmt.Errorf("could not write template %s: %w", j.src, err)
		}
	}
	return nil
}

func (g *Generator) cleanupGenerated() error {
	paths := []string{
		filepath.Join(g.OutDir, "cmd"),
		filepath.Join(g.OutDir, "internal", "mcpapp"),
	}

	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("cleanup %s: %w", p, err)
		}
	}
	return nil
}

func (g *Generator) writeTemplate(name string, path string, data TemplateData, gofmt bool) error {
	content, err := RenderTemplate(name, data)
	if err != nil {
		return err
	}

	if gofmt {
		formatted, err := format.Source(content)
		if err != nil {
			return fmt.Errorf("format %s: %w", path, err)
		}
		content = formatted
	}
	return os.WriteFile(path, content, 0o644)
}
