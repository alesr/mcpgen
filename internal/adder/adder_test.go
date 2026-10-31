package adder

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProject generates a fresh project in a temp dir and returns its path.
func newProject(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "proj")
	cfg := config.DefaultConfig("http", 9090, true, true, true)
	cfg.Server.Name = "weather"
	cfg.Server.Module = "github.com/acme/weather"
	require.NoError(t, cfg.Validate())
	gen := &generator.Generator{Config: cfg, OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)
	return dir
}

func writeSpec(t *testing.T, dir, name string, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

const toolSpec = `{"id": "health", "title": "Health", "description": "Health check.", "readOnly": true, "idempotent": true}`

func TestAdd_Tool(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)

	res, err := Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)
	assert.Equal(t, "health", res.ID)
	assert.Contains(t, res.FilesWritten, "internal/mcpapp/tools/handlers/health.go")
	assert.Contains(t, res.FilesWritten, "internal/mcpapp/tools/handlers/health_test.go")
	assert.Equal(t, "internal/mcpapp/tools.go", res.RegisteredIn)

	// config updated
	cfg, err := config.Load(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg.FindTool("health"))

	// registration inserted inside markers, file still parses + gofmt-clean
	toolsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools.go"))
	require.NoError(t, err)
	begin := strings.Index(string(toolsGo), "// mcpgen:begin tools")
	end := strings.Index(string(toolsGo), "// mcpgen:end tools")
	require.Positive(t, begin)
	require.Positive(t, end)
	region := string(toolsGo)[begin:end]
	assert.Contains(t, region, `Name:         "health"`)
	assert.Contains(t, region, "handlers.Health")
	assert.Contains(t, region, "IdempotentHint: true")

	// handler file is user-owned and parses
	handler, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools/handlers/health.go"))
	require.NoError(t, err)
	assert.Contains(t, string(handler), "User-owned")
	assert.Contains(t, string(handler), "func Health(")

	// manifest updated
	m, err := generator.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, generator.OwnerUser, m.Files["internal/mcpapp/tools/handlers/health.go"].Owner)
}

// TestAdd_Tool_EmptyRegion_AddsJSONImport: on a project generated with no
// tools, tools.go has no encoding/json import; the inserted statement uses
// json.RawMessage, so add must ensure the import or the project won't build.
func TestAdd_Tool_EmptyRegion_AddsJSONImport(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "proj")
	cfg := config.DefaultConfig("http", 9090, false, false, false)
	cfg.Server.Name = "bare"
	cfg.Server.Module = "example.com/bare"
	require.NoError(t, cfg.Validate())
	gen := &generator.Generator{Config: cfg, OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)

	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)
	res, err := Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)
	assert.Equal(t, "health", res.ID)

	toolsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools.go"))
	require.NoError(t, err)
	s := string(toolsGo)
	assert.Contains(t, s, `"encoding/json"`)
	assert.Contains(t, s, `Name:         "health"`)
	// file still parses
	_, err = parser.ParseFile(token.NewFileSet(), "tools.go", toolsGo, parser.ParseComments)
	assert.NoError(t, err)
}

func TestAdd_DuplicateConflict_Exit5(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)
	_, err := Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)

	_, err = Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.Error(t, err)
	assert.Equal(t, cli.ExitConflict, cli.ExitCodeOf(err))
	assert.Contains(t, err.Error(), "already registered")
}

func TestAdd_NotAProject_Exit3(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)
	_, err := Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.Error(t, err)
	assert.Equal(t, cli.ExitExists, cli.ExitCodeOf(err))
}

func TestAdd_PromptAndResource(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	promptSpec := writeSpec(t, t.TempDir(), "p.json", `{"id": "report", "description": "r", "arguments": [{"name": "topic", "required": true}]}`)
	resSpec := writeSpec(t, t.TempDir(), "r.json", `{"id": "cfg", "uriTemplate": "file:///c/{name}", "mimeType": "application/json", "text": "{}"}`)

	resP, err := Add(KindPrompt, promptSpec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)
	assert.Equal(t, "report", resP.ID)
	promptsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/prompts.go"))
	require.NoError(t, err)
	assert.Contains(t, string(promptsGo), `Name:        "report"`)
	assert.Contains(t, string(promptsGo), "prompts.Report")
	assert.FileExists(t, filepath.Join(dir, "internal/mcpapp/prompts/report.go"))

	resR, err := Add(KindResource, resSpec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)
	assert.Equal(t, "cfg", resR.ID)
	resourcesGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/resources.go"))
	require.NoError(t, err)
	assert.Contains(t, string(resourcesGo), "AddResourceTemplate")
	assert.Contains(t, string(resourcesGo), `URITemplate: "file:///c/{name}"`)
}

func TestAdd_DryRun_WritesNothing(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)

	res, err := Add(KindTool, spec, Options{Dir: dir, DryRun: true, Version: "test", NoCheck: true})
	require.NoError(t, err)
	assert.Contains(t, res.FilesWritten, "internal/mcpapp/tools/handlers/health.go")

	assert.NoFileExists(t, filepath.Join(dir, "internal/mcpapp/tools/handlers/health.go"))
	cfg, err := config.Load(dir)
	require.NoError(t, err)
	assert.Nil(t, cfg.FindTool("health"))
}

func TestAdd_MissingMarkers_Exit5(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	// destroy the markers
	toolsPath := filepath.Join(dir, "internal/mcpapp/tools.go")
	raw, err := os.ReadFile(toolsPath)
	require.NoError(t, err)
	broken := strings.ReplaceAll(string(raw), "// mcpgen:end tools", "// oops")
	require.NoError(t, os.WriteFile(toolsPath, []byte(broken), 0o644))

	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)
	_, err = Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.Error(t, err)
	assert.Equal(t, cli.ExitConflict, cli.ExitCodeOf(err))
	assert.Contains(t, err.Error(), "restore the mcpgen:begin/end markers")
}

func TestAdd_ExistingUserFileSkipped(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	spec := writeSpec(t, t.TempDir(), "health.json", toolSpec)
	_, err := Add(KindTool, spec, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)

	// hand-edit the handler, add a different tool: handler must be untouched
	handler := filepath.Join(dir, "internal/mcpapp/tools/handlers/health.go")
	require.NoError(t, os.WriteFile(handler, []byte("package handlers\n\n// mine\n"), 0o644))

	spec2 := writeSpec(t, t.TempDir(), "other.json", `{"id": "other", "description": "x"}`)
	_, err = Add(KindTool, spec2, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)
	got, err := os.ReadFile(handler)
	require.NoError(t, err)
	assert.Contains(t, string(got), "mine")
}

func TestParseKind(t *testing.T) {
	t.Parallel()
	k, err := ParseKind("tool")
	require.NoError(t, err)
	assert.Equal(t, KindTool, k)
	_, err = ParseKind("bogus")
	require.Error(t, err)
	assert.Equal(t, cli.ExitUsage, cli.ExitCodeOf(err))
}

func TestInsertIntoMarkerRegion_Formats(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	toolsPath := filepath.Join(dir, "internal/mcpapp/tools.go")
	stmt := "server.AddTool(&mcp.Tool{\nName: \"x\",\n}, handlers.X)" // intentionally badly indented
	require.NoError(t, InsertIntoMarkerRegion(toolsPath, RegionTools, stmt))
	raw, err := os.ReadFile(toolsPath)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "// mcpgen:begin tools")
	assert.Contains(t, s, "// mcpgen:end tools")
	assert.Contains(t, s, `Name: "x"`)
	// gofmt aligned the inserted statement
	assert.Contains(t, s, "\t\tName: \"x\",")
}

// TestAdd_TwiceWithHandEdits is the M3 acceptance criterion: `mcpgen add`
// works twice in a row on a project with hand-edited handlers without
// touching user files.
func TestAdd_TwiceWithHandEdits(t *testing.T) {
	t.Parallel()
	dir := newProject(t)
	base := t.TempDir()
	specA := writeSpec(t, base, "a.json", `{"id": "aaa", "description": "a"}`)
	specB := writeSpec(t, base, "b.json", `{"id": "bbb", "description": "b"}`)

	_, err := Add(KindTool, specA, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)

	// hand-edit tool A's handler (user-owned)
	handlerA := filepath.Join(dir, "internal/mcpapp/tools/handlers/aaa.go")
	require.NoError(t, os.WriteFile(handlerA, []byte("package handlers\n\n// hand-written logic\n"), 0o644))

	// add tool B
	res, err := Add(KindTool, specB, Options{Dir: dir, Version: "test", NoCheck: true})
	require.NoError(t, err)
	assert.Equal(t, "bbb", res.ID)

	// A's hand edit survived
	got, err := os.ReadFile(handlerA)
	require.NoError(t, err)
	assert.Contains(t, string(got), "hand-written logic")

	// both registered in the marker region, config has both
	toolsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools.go"))
	require.NoError(t, err)
	assert.Contains(t, string(toolsGo), `"aaa"`)
	assert.Contains(t, string(toolsGo), `"bbb"`)
	cfg, err := config.Load(dir)
	require.NoError(t, err)
	assert.NotNil(t, cfg.FindTool("aaa"))
	assert.NotNil(t, cfg.FindTool("bbb"))
}
