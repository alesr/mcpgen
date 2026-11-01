package generator

import (
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formatStruct runs a generated struct definition through gofmt, mirroring the
// generator pipeline (which formats every generated Go file), so assertions
// target the output that actually lands on disk.
func formatStruct(t *testing.T, s string) string {
	t.Helper()
	src, err := format.Source([]byte("package p\n\n" + s + "\n"))
	require.NoError(t, err)
	return strings.TrimSpace(strings.TrimPrefix(string(src), "package p\n\n"))
}

func TestStructDef(t *testing.T) {
	t.Parallel()

	t.Run("empty object", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "type Foo struct {}", structDef("Foo", `{"type":"object"}`))
	})

	t.Run("invalid JSON yields empty struct", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "type Foo struct {}", structDef("Foo", `not json`))
	})

	t.Run("flat properties with required and description", func(t *testing.T) {
		t.Parallel()
		schema := `{
			"type": "object",
			"properties": {
				"city":   {"type": "string",  "description": "City name"},
				"days":   {"type": "integer"},
				"temp":   {"type": "number"},
				"active": {"type": "boolean"}
			},
			"required": ["city"]
		}`
		want := strings.Join([]string{
			"type Foo struct {",
			"\tActive bool `json:\"active,omitempty\"`",
			"\t// City name",
			"\tCity string `json:\"city\"`",
			"\tDays int64 `json:\"days,omitempty\"`",
			"\tTemp float64 `json:\"temp,omitempty\"`",
			"}",
		}, "\n")
		assert.Equal(t, want, structDef("Foo", schema))
	})

	t.Run("nested object is an inline struct", func(t *testing.T) {
		t.Parallel()
		schema := `{
			"type": "object",
			"properties": {
				"location": {
					"type": "object",
					"properties": {"lat": {"type": "number"}, "lng": {"type": "number"}}
				}
			}
		}`
		got := formatStruct(t, structDef("Foo", schema))
		assert.Contains(t, got, "Location struct {\n\t\tLat float64 `json:\"lat,omitempty\"`\n\t\tLng float64 `json:\"lng,omitempty\"`\n\t} `json:\"location,omitempty\"`")
	})

	t.Run("array of strings and array of objects", func(t *testing.T) {
		t.Parallel()
		schema := `{
			"type": "object",
			"properties": {
				"tags": {"type": "array", "items": {"type": "string"}},
				"pts":  {"type": "array", "items": {"type": "object", "properties": {"x": {"type": "integer"}}}}
			}
		}`
		got := formatStruct(t, structDef("Foo", schema))
		assert.Contains(t, got, "Pts []struct {\n\t\tX int64 `json:\"x,omitempty\"`\n\t} `json:\"pts,omitempty\"`")
		assert.Contains(t, got, "Tags []string `json:\"tags,omitempty\"`")
	})

	t.Run("union and untyped fall back to any", func(t *testing.T) {
		t.Parallel()
		schema := `{
			"type": "object",
			"properties": {
				"one_of":  {"type": ["string", "number"]},
				"untyped": {}
			}
		}`
		got := structDef("Foo", schema)
		assert.Contains(t, got, "OneOf any `json:\"one_of,omitempty\"`")
		assert.Contains(t, got, "Untyped any `json:\"untyped,omitempty\"`")
	})

	t.Run("colliding field names are deduplicated", func(t *testing.T) {
		t.Parallel()
		schema := `{
			"type": "object",
			"properties": {
				"foo_bar": {"type": "string"},
				"fooBar":  {"type": "string"}
			}
		}`
		got := structDef("Foo", schema)
		assert.Contains(t, got, "FooBar string `json:\"fooBar,omitempty\"`")
		assert.Contains(t, got, "FooBar2 string `json:\"foo_bar,omitempty\"`")
	})
}

func TestURITemplateVars(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"name"}, uriTemplateVars("file:///config/{name}"))
	assert.Equal(t, []string{"name", "path"}, uriTemplateVars("file:///{name}/{path}"))
	assert.Empty(t, uriTemplateVars("file:///static"))
}
