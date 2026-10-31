package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelope_Roundtrip(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	env := EnvelopeOK(map[string]string{"dir": "./x"}, []string{"cd x"})
	require.NoError(t, WriteEnvelope(&buf, env))

	var got Envelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.True(t, got.OK)
	assert.Nil(t, got.Error)
	assert.Equal(t, []string{"cd x"}, got.NextActions)
}

func TestEnvelopeFromError_ExitError(t *testing.T) {
	t.Parallel()
	ee := NewExitError(ExitConflict, "add", CodeConflict, "tool %q already registered", "greet").WithFix("pick another id")
	env := EnvelopeFromError(ee, []string{"next"})
	require.NotNil(t, env.Error)
	assert.Equal(t, 5, env.Error.ExitCode)
	assert.Equal(t, "add", env.Error.Stage)
	assert.Equal(t, CodeConflict, env.Error.Code)
	assert.Equal(t, `tool "greet" already registered`, env.Error.Message)
	assert.Equal(t, "pick another id", env.Error.Fix)
	assert.Equal(t, []string{"next"}, env.NextActions)
	assert.False(t, env.OK)
}

func TestEnvelopeFromError_PlainError(t *testing.T) {
	t.Parallel()
	env := EnvelopeFromError(errors.New("boom"), nil)
	require.NotNil(t, env.Error)
	assert.Equal(t, ExitFailure, env.Error.ExitCode)
	assert.Equal(t, CodeInternal, env.Error.Code)
	assert.Equal(t, "boom", env.Error.Message)
}

func TestExitError_WithFixReturnsCopy(t *testing.T) {
	t.Parallel()
	ee := NewExitError(ExitFailure, "s", CodeInternal, "m")
	c := ee.WithFix("f")
	assert.Empty(t, ee.Fix, "WithFix must not mutate the receiver")
	assert.Equal(t, "f", c.Fix)
}

func TestExitCodeOf(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ExitOK, ExitCodeOf(nil))
	assert.Equal(t, ExitUsage, ExitCodeOf(NewExitError(ExitUsage, "x", CodeUsage, "bad")))
	assert.Equal(t, ExitFailure, ExitCodeOf(errors.New("plain")))
}

func TestBuildSchemaDoc(t *testing.T) {
	t.Parallel()
	doc, err := BuildSchemaDoc("1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "mcpgen", doc.Tool)
	assert.Equal(t, "1.0.0", doc.Version)

	names := make([]string, 0, len(doc.Commands))
	for _, c := range doc.Commands {
		names = append(names, c.Name)
	}
	assert.Contains(t, names, "new")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "check")
	assert.Contains(t, names, "schema")
	assert.Contains(t, names, "version")
	assert.Contains(t, names, "mcp")

	codes := make([]int, 0, len(doc.ExitCodes))
	for _, e := range doc.ExitCodes {
		codes = append(codes, e.Code)
	}
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6}, codes)

	assert.Equal(t, "object", doc.ConfigJSONSchema["type"])

	// must marshal to JSON cleanly
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "mcpgen.json")
}
