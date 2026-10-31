package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_Defaults(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Version: ConfigVersion,
		Server:  Server{Name: "weather"},
	}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, "weather", cfg.Server.Title)
	assert.Equal(t, "0.1.0", cfg.Server.Version)
	assert.Equal(t, "example.com/weather", cfg.Server.Module)
	assert.Equal(t, "stdio", cfg.Transport.Type)
	assert.Equal(t, DefaultHTTPPort, cfg.Transport.HTTPPort)
	assert.Equal(t, ProtocolCompat, cfg.Protocol)
}

func TestConfig_Validate_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"version unsupported", func(c *Config) { c.Version = 99 }, "unsupported mcpgen.json version"},
		{"missing name", func(c *Config) { c.Server.Name = " " }, "server.name is required"},
		{"bad module", func(c *Config) { c.Server.Module = "not a module!!" }, "server.module"},
		{"bad transport", func(c *Config) { c.Transport.Type = "grpc" }, "transport.type"},
		{"bad port", func(c *Config) { c.Transport.HTTPPort = 70000 }, "httpPort"},
		{"bad protocol", func(c *Config) { c.Protocol = "bleh" }, "protocol"},
		{"tool missing id", func(c *Config) { c.Tools = []Tool{{}} }, "tools[0].id is required"},
		{"duplicate tool id", func(c *Config) { c.Tools = []Tool{{ID: "a"}, {ID: "a"}} }, "duplicate id"},
		{"bad input schema", func(c *Config) { c.Tools = []Tool{{ID: "a", InputSchema: json.RawMessage(`{invalid`)}} }, "inputSchema"},
		{"schema not object", func(c *Config) { c.Tools = []Tool{{ID: "a", InputSchema: json.RawMessage(`{"type":"array"}`)}} }, "type=object"},
		{"schema type array without object", func(c *Config) {
			c.Tools = []Tool{{ID: "a", InputSchema: json.RawMessage(`{"type":["string","null"]}`)}}
		}, "type=object"},
		{"schema type not string or array", func(c *Config) { c.Tools = []Tool{{ID: "a", InputSchema: json.RawMessage(`{"type":123}`)}} }, "type=object"},
		{"resource needs uri", func(c *Config) { c.Resources = []Resource{{ID: "r"}} }, "either uri or uriTemplate"},
		{"resource both uri forms", func(c *Config) { c.Resources = []Resource{{ID: "r", URI: "file:///a", URITemplate: "file:///{x}"}} }, "only one"},
		{"resource bad uri", func(c *Config) { c.Resources = []Resource{{ID: "r", URI: "no-scheme"}} }, "scheme"},
		{"prompt arg missing name", func(c *Config) { c.Prompts = []Prompt{{ID: "p", Arguments: []PromptArgument{{}}}} }, "arguments[0].name"},
		{"duplicate prompt arg", func(c *Config) {
			c.Prompts = []Prompt{{ID: "p", Arguments: []PromptArgument{{Name: "x"}, {Name: "x"}}}}
		}, "duplicate name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{Version: ConfigVersion, Server: Server{Name: "s", Module: "example.com/s"}}
			tt.mutate(cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConfig_Validate_SchemaTypeArray(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		`{"type":"object"}`,
		`{"type":["object","null"]}`,
		`{"type":["null","object"],"properties":{"a":{"type":"string"}}}`,
	} {
		cfg := &Config{
			Version: ConfigVersion,
			Server:  Server{Name: "s", Module: "example.com/s"},
			Tools:   []Tool{{ID: "a", InputSchema: json.RawMessage(raw)}},
		}
		require.NoError(t, cfg.Validate(), raw)
	}
}

func TestConfig_LoadSave_Roundtrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := DefaultConfig("http", 9090, true, true, true)
	cfg.Server.Name = "weather"
	cfg.Server.Module = "github.com/acme/weather"
	require.NoError(t, cfg.Validate())

	require.NoError(t, Save(dir, cfg))

	loaded, err := Load(dir)
	require.NoError(t, err)
	// compare semantically (RawMessage bytes may be re-indented by Save)
	var a, b any
	require.NoError(t, json.Unmarshal(mustJSON(t, cfg), &a))
	require.NoError(t, json.Unmarshal(mustJSON(t, loaded), &b))
	assert.Equal(t, a, b)

	// deterministic bytes
	raw1, err := Marshal(cfg)
	require.NoError(t, err)
	raw2, err := os.ReadFile(filepath.Join(dir, FileName))
	require.NoError(t, err)
	assert.Equal(t, raw1, raw2)
	assert.True(t, strings.HasSuffix(string(raw2), "\n"))
}

func TestConfig_Load_Missing(t *testing.T) {
	t.Parallel()
	_, err := Load(t.TempDir())
	require.ErrorIs(t, err, ErrConfigNotFound)
}

func TestConfig_Load_BadVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte(`{"version": 42, "server": {"name": "x", "module": "example.com/x"}, "transport": {"type": "stdio"}}`), 0o644))
	_, err := Load(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported mcpgen.json version")
	assert.Contains(t, err.Error(), "fix:")
}

func TestIsProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	assert.False(t, IsProject(dir))
	cfg := DefaultConfig("stdio", 8080, true, false, false)
	require.NoError(t, cfg.Validate())
	require.NoError(t, Save(dir, cfg))
	assert.False(t, IsProject(dir)) // manifest still missing
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ManifestDir), 0o755))
	require.NoError(t, os.WriteFile(ManifestPath(dir), []byte(`{"generator":"mcpgen","version":"1.0.0","files":{}}`), 0o644))
	assert.True(t, IsProject(dir))
}

// TestLoad_ServerModuleOptional: schema must not require server.module —
// Validate derives it from the server name when absent.
func TestLoad_ServerModuleOptional(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"version":1,"server":{"name":"x"},"transport":{"type":"stdio"}}`), 0o644))
	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "example.com/x", cfg.Server.Module)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}

func TestNormalizeJSONSchema(t *testing.T) {
	t.Parallel()
	assert.Equal(t, `{"type":"object"}`, NormalizeJSONSchema(nil))
	assert.Equal(t, `{"type":"object"}`, NormalizeJSONSchema(json.RawMessage("  ")))
	assert.Equal(t, `{"properties":{"a":{"type":"string"}},"type":"object"}`, NormalizeJSONSchema(json.RawMessage("{ \"properties\": {\"a\": {\"type\": \"string\"}}, \"type\": \"object\" }")))
	assert.Equal(t, `{invalid`, NormalizeJSONSchema(json.RawMessage(`{invalid`)))
}

func TestSchemaJSON_Valid(t *testing.T) {
	t.Parallel()
	raw, err := SchemaJSON()
	require.NoError(t, err)
	var v map[string]any
	require.NoError(t, json.Unmarshal(raw, &v))
	assert.Equal(t, "object", v["type"])
	assert.Contains(t, v, "definitions")
}

func TestLoad_SchemaValidation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// valid config passes schema + semantic validation
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"version":1,"server":{"name":"x","module":"example.com/x"},"transport":{"type":"stdio"},"tools":[{"id":"t"}]}`), 0o644))
	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "t", cfg.Tools[0].ID)

	// schema violation: unknown top-level field (additionalProperties: false)
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"version":1,"server":{"name":"x","module":"example.com/x"},"transport":{"type":"stdio"},"bogus":true}`), 0o644))
	_, err = Load(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON Schema")

	// schema violation: bad enum
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"version":1,"server":{"name":"x","module":"example.com/x"},"transport":{"type":"grpc"}}`), 0o644))
	_, err = Load(dir)
	require.Error(t, err)

	// unknown version keeps its sentinel (mapped to exit 2 by callers)
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName),
		[]byte(`{"version":42,"server":{"name":"x","module":"example.com/x"},"transport":{"type":"stdio"}}`), 0o644))
	_, err = Load(dir)
	require.ErrorIs(t, err, ErrConfigVersionUnsupported)
}
