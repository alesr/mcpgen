package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/alesr/mcpgen/internal/cli"
	"github.com/alesr/mcpgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T, transport string) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig(transport, 9090, true, true, true)
	cfg.Server.Name = "weather"
	cfg.Server.Module = "github.com/acme/weather"
	require.NoError(t, cfg.Validate())
	return cfg
}

func TestGenerator_Run_WritesExpectedTree(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "weather")
	gen := &Generator{Config: testConfig(t, "http"), OutDir: dir, Version: "test"}
	res, err := gen.Run()
	require.NoError(t, err)

	want := []string{
		"go.mod",
		"mcpgen.json",
		".mcpgen/manifest.json",
		"cmd/weather/main.go",
		"internal/mcpapp/mcpapp.go",
		"internal/mcpapp/tools.go",
		"internal/mcpapp/prompts.go",
		"internal/mcpapp/resources.go",
		"internal/mcpapp/tools/handlers/greet.go",
		"internal/mcpapp/tools/handlers/greet_test.go",
		"internal/mcpapp/prompts/welcome.go",
		"internal/mcpapp/resources/readme.go",
		"Makefile",
		".github/workflows/ci.yml",
		"server.json",
		"AGENTS.md",
		"README.md",
		"Dockerfile",
	}
	for _, w := range want {
		assert.FileExists(t, filepath.Join(dir, filepath.FromSlash(w)), w)
	}

	// go.mod pins the SDK
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(gomod), "github.com/modelcontextprotocol/go-sdk v1.8.0")
	assert.Contains(t, string(gomod), "go 1.26")

	// markers present
	toolsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools.go"))
	require.NoError(t, err)
	assert.Contains(t, string(toolsGo), "// mcpgen:begin tools")
	assert.Contains(t, string(toolsGo), "// mcpgen:end tools")
	assert.Contains(t, string(toolsGo), "handlers.Greet")

	// ownership headers
	mcpappGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/mcpapp.go"))
	require.NoError(t, err)
	assert.Contains(t, string(mcpappGo), "DO NOT EDIT region: mcpgen-owned")
	handlerGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools/handlers/greet.go"))
	require.NoError(t, err)
	assert.Contains(t, string(handlerGo), "User-owned")

	// server.json has registry name + OCI package
	serverJSON, err := os.ReadFile(filepath.Join(dir, "server.json"))
	require.NoError(t, err)
	var sj map[string]any
	require.NoError(t, json.Unmarshal(serverJSON, &sj))
	assert.Equal(t, "io.github.acme/weather", sj["name"])
	pkgs := sj["packages"].([]any)
	pkg := pkgs[0].(map[string]any)
	assert.Equal(t, "oci", pkg["registryType"])
	assert.Equal(t, "ghcr.io/acme/weather", pkg["identifier"])

	// manifest tracks files with owners
	m, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, OwnerMCPGen, m.Files["go.mod"].Owner)
	assert.Equal(t, OwnerUser, m.Files["internal/mcpapp/tools/handlers/greet.go"].Owner)
	assert.NotEmpty(t, m.Files["go.mod"].SHA256)

	// result lists written files
	assert.Contains(t, res.FilesWritten, "mcpgen.json")
	assert.NotNil(t, res.Plan)
}

func TestGenerator_Deterministic(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	dirA, dirB := filepath.Join(base, "a"), filepath.Join(base, "b")
	for _, d := range []string{dirA, dirB} {
		gen := &Generator{Config: testConfig(t, "stdio"), OutDir: d, Version: "test"}
		_, err := gen.Run()
		require.NoError(t, err)
	}
	assertTreesEqual(t, dirA, dirB)
}

func TestGenerator_DryRun_WritesNothing(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "dry")
	gen := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, DryRun: true, Version: "test"}
	res, err := gen.Run()
	require.NoError(t, err)
	assert.NotNil(t, res.Plan)
	assert.NotEmpty(t, res.Plan.Files)
	for _, f := range res.Plan.Files {
		assert.Equal(t, ActionCreate, f.Action)
	}
	// The plan is the full output set: mcpgen.json and the manifest are
	// written outside the targets loop but must still appear in the plan.
	paths := make([]string, 0, len(res.Plan.Files))
	for _, f := range res.Plan.Files {
		paths = append(paths, f.Path)
	}
	assert.Contains(t, paths, config.FileName)
	assert.Contains(t, paths, config.ManifestDir+"/"+config.ManifestFile)
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err), "dry-run must not create the output dir")
}

func TestGenerator_DryRun_PlanCoversConfigAndManifestOnRegen(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "proj")
	gen := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)

	res, err := (&Generator{Config: testConfig(t, "stdio"), OutDir: dir, Force: true, DryRun: true, Version: "test"}).Run()
	require.NoError(t, err)
	byPath := map[string]FilePlan{}
	for _, f := range res.Plan.Files {
		byPath[f.Path] = f
	}
	require.Contains(t, byPath, config.FileName)
	assert.Equal(t, ActionOverwrite, byPath[config.FileName].Action)
	assert.Equal(t, OwnerMCPGen, byPath[config.FileName].Owner)
	require.Contains(t, byPath, config.ManifestDir+"/"+config.ManifestFile)
	assert.Equal(t, ActionOverwrite, byPath[config.ManifestDir+"/"+config.ManifestFile].Action)
}

func TestGenerator_ExistingDirRequiresForce(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "exists")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644))

	gen := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.Error(t, err)
	assert.Equal(t, cli.ExitExists, cli.ExitCodeOf(err))

	genForce := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, Force: true, Version: "test"}
	_, err = genForce.Run()
	require.NoError(t, err)
	// pre-existing foreign file untouched
	assert.FileExists(t, filepath.Join(dir, "file.txt"))
}

func TestGenerator_UnsafeOutDir(t *testing.T) {
	t.Parallel()
	for _, d := range []string{".", ".."} {
		gen := &Generator{Config: testConfig(t, "stdio"), OutDir: d, Version: "test"}
		_, err := gen.Run()
		require.Error(t, err)
		assert.Equal(t, cli.ExitUsage, cli.ExitCodeOf(err))
	}
}

func TestGenerator_RegenSkipsUserAndModifiedFiles(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "proj")
	gen := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)

	// user edits a handler (user-owned)
	handler := filepath.Join(dir, "internal/mcpapp/tools/handlers/greet.go")
	require.NoError(t, os.WriteFile(handler, []byte("package handlers\n\n// my edit\n"), 0o644))
	// user edits an mcpgen-owned file
	mcpapp := filepath.Join(dir, "internal/mcpapp/mcpapp.go")
	raw, err := os.ReadFile(mcpapp)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(mcpapp, append(raw, []byte("\n// hand edit\n")...), 0o644))

	gen2 := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, Force: true, Version: "test"}
	res, err := gen2.Run()
	require.NoError(t, err)
	assert.Contains(t, res.FilesSkipped, "internal/mcpapp/tools/handlers/greet.go")
	assert.Contains(t, res.FilesSkipped, "internal/mcpapp/mcpapp.go")

	// user edit survived
	got, err := os.ReadFile(handler)
	require.NoError(t, err)
	assert.Contains(t, string(got), "my edit")
}

// TestGenerator_RegenPrunesOrphanedManifestEntries covers the removal
// workflow: drop a tool from the config, delete its handler files, regen —
// the stale manifest entries must be pruned.
func TestGenerator_RegenPrunesOrphanedManifestEntries(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "proj")
	cfg := testConfig(t, "stdio")
	cfg.Tools = append(cfg.Tools, config.Tool{ID: "extra", Description: "extra tool"})
	require.NoError(t, cfg.Validate())
	_, err := (&Generator{Config: cfg, OutDir: dir, Version: "test"}).Run()
	require.NoError(t, err)

	m, err := LoadManifest(dir)
	require.NoError(t, err)
	require.Contains(t, m.Files, "internal/mcpapp/tools/handlers/extra.go")

	// remove the tool from the config and delete its files, then regen
	cfg.Tools = cfg.Tools[:1]
	require.NoError(t, os.Remove(filepath.Join(dir, "internal/mcpapp/tools/handlers/extra.go")))
	require.NoError(t, os.Remove(filepath.Join(dir, "internal/mcpapp/tools/handlers/extra_test.go")))

	_, err = (&Generator{Config: cfg, OutDir: dir, Force: true, Version: "test"}).Run()
	require.NoError(t, err)

	m, err = LoadManifest(dir)
	require.NoError(t, err)
	assert.NotContains(t, m.Files, "internal/mcpapp/tools/handlers/extra.go")
	assert.NotContains(t, m.Files, "internal/mcpapp/tools/handlers/extra_test.go")
	// unrelated entries survive
	assert.Contains(t, m.Files, config.FileName)
	assert.Contains(t, m.Files, "internal/mcpapp/tools/handlers/greet.go")
}

// TestGenerator_RegenTwice_KeepsModifiedMCPGenFile guards the manifest
// baseline: a file skipped for hash mismatch must keep its original manifest
// entry, otherwise the second regen sees a match and overwrites the edit.
func TestGenerator_RegenTwice_KeepsModifiedMCPGenFile(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "proj")
	gen := &Generator{Config: testConfig(t, "stdio"), OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)

	mcpapp := filepath.Join(dir, "internal/mcpapp/mcpapp.go")
	raw, err := os.ReadFile(mcpapp)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(mcpapp, append(raw, []byte("\n// hand edit\n")...), 0o644))

	for i := range 2 {
		res, err := (&Generator{Config: testConfig(t, "stdio"), OutDir: dir, Force: true, Version: "test"}).Run()
		require.NoError(t, err)
		assert.Contains(t, res.FilesSkipped, "internal/mcpapp/mcpapp.go", "regen %d must skip the modified file", i+1)
		got, err := os.ReadFile(mcpapp)
		require.NoError(t, err)
		assert.Contains(t, string(got), "hand edit", "regen %d must not clobber the edit", i+1)
	}
}

func TestGenerator_MultiItemConfig(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig("stdio", 8080, false, false, false)
	cfg.Server.Name = "multi"
	cfg.Server.Module = "example.com/multi"
	cfg.Tools = []config.Tool{
		{ID: "a_one", ReadOnly: true},
		{ID: "b-two", Idempotent: true},
	}
	cfg.Prompts = []config.Prompt{{ID: "p1", Arguments: []config.PromptArgument{{Name: "x", Required: true}}}}
	cfg.Resources = []config.Resource{{ID: "r1", URITemplate: "file:///{id}", MIMEType: "text/plain"}}
	require.NoError(t, cfg.Validate())

	dir := filepath.Join(t.TempDir(), "multi")
	gen := &Generator{Config: cfg, OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)

	toolsGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/tools.go"))
	require.NoError(t, err)
	assert.Contains(t, string(toolsGo), `Name:         "a_one"`)
	assert.Contains(t, string(toolsGo), `Name:         "b-two"`)
	assert.Contains(t, string(toolsGo), "handlers.BTwo")
	assert.FileExists(t, filepath.Join(dir, "internal/mcpapp/tools/handlers/a_one.go"))
	assert.FileExists(t, filepath.Join(dir, "internal/mcpapp/tools/handlers/b_two.go"))

	resGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/resources.go"))
	require.NoError(t, err)
	assert.Contains(t, string(resGo), "AddResourceTemplate")
	assert.Contains(t, string(resGo), `URITemplate: "file:///{id}"`)

	// all generated go files parse
	assertAllGoParses(t, dir)
}

func TestGenerator_Features(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig("http", 8080, true, false, false)
	cfg.Server.Name = "feat"
	cfg.Server.Module = "example.com/feat"
	cfg.Features = config.Features{Tasks: true, Skills: true, OAuth: true}
	require.NoError(t, cfg.Validate())

	dir := filepath.Join(t.TempDir(), "feat")
	gen := &Generator{Config: cfg, OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "internal/mcpapp/tasks.go"))
	assert.FileExists(t, filepath.Join(dir, "internal/mcpapp/skills.go"))
	assert.FileExists(t, filepath.Join(dir, "internal/mcpapp/oauth.go"))
	mcpappGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/mcpapp.go"))
	require.NoError(t, err)
	assert.Contains(t, string(mcpappGo), "registerTasks(server)")
	assert.Contains(t, string(mcpappGo), "registerSkills(server)")
	assertAllGoParses(t, dir)
}

func TestGenerator_LatestOnlyProtocol(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig("http", 8080, false, false, false)
	cfg.Server.Name = "lo"
	cfg.Server.Module = "example.com/lo"
	cfg.Protocol = config.ProtocolLatestOnly
	require.NoError(t, cfg.Validate())
	dir := filepath.Join(t.TempDir(), "lo")
	gen := &Generator{Config: cfg, OutDir: dir, Version: "test"}
	_, err := gen.Run()
	require.NoError(t, err)
	mcpappGo, err := os.ReadFile(filepath.Join(dir, "internal/mcpapp/mcpapp.go"))
	require.NoError(t, err)
	assert.Contains(t, string(mcpappGo), `SupportedProtocolVersions: []string{"2026-07-28"}`)
}

func TestBuildTemplateData_Collisions(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig("stdio", 8080, false, false, false)
	cfg.Server.Name = "c"
	cfg.Server.Module = "example.com/c"
	cfg.Tools = []config.Tool{{ID: "get-forecast"}, {ID: "get_forecast"}}
	require.NoError(t, cfg.Validate()) // ids differ, but Go idents collide
	_, err := BuildTemplateData(cfg, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GetForecast")
}

// assertTreesEqual fails if two directory trees differ in any file content.
func assertTreesEqual(t *testing.T, a, b string) {
	t.Helper()
	filesA := listFiles(t, a)
	filesB := listFiles(t, b)
	require.Equal(t, filesA, filesB)
	for _, f := range filesA {
		ca, err := os.ReadFile(filepath.Join(a, f))
		require.NoError(t, err)
		cb, err := os.ReadFile(filepath.Join(b, f))
		require.NoError(t, err)
		assert.Equal(t, string(ca), string(cb), f)
	}
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	}))
	sort.Strings(out)
	return out
}

func assertAllGoParses(t *testing.T, root string) {
	t.Helper()
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".go" {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := parseGo(src); err != nil {
			t.Errorf("generated file does not parse: %s: %v", path, err)
		}
		return nil
	}))
}
