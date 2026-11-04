package generator

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoRawString(t *testing.T) {
	t.Parallel()
	// Plain JSON schema: readable backtick raw literal.
	assert.Equal(t, "`{\"type\":\"object\"}`", goRawString(`{"type":"object"}`))
	// A schema containing a backtick must fall back to a quoted literal.
	withTick := "{\"description\":\"use `code` here\"}"
	assert.Equal(t, strconv.Quote(withTick), goRawString(withTick))
}

func TestRenderTemplate_GoRawInToolStatement(t *testing.T) {
	t.Parallel()
	td := ToolData{
		GeneratorVersion: "test",
		ID:               "greet",
		GoName:           "Greet",
		Title:            "Greet",
		Description:      "Greets.",
		InputSchema:      `{"type":"object"}`,
		OutputSchema:     `{"type":"object"}`,
	}
	stmt, err := RenderStatement("stmt_tool.go.gotmpl", td)
	assert.NoError(t, err)
	assert.Contains(t, stmt, "json.RawMessage(`{\"type\":\"object\"}`)")
	assert.NotContains(t, stmt, `json.RawMessage("{\"type\":\"object\"}")`)
}
