package mcpserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInput_ToOptions_RejectsBadInput(t *testing.T) {
	t.Parallel()
	in := NewInput{Tools: []json.RawMessage{[]byte(`{"id": 123}`)}}
	_, err := in.toOptions("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tools[0]")

	in2 := NewInput{Features: []string{"bogus"}}
	_, err = in2.toOptions("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown feature")
}

// TestMCPServer_EndToEnd drives mcpgen_new → mcpgen_add → mcpgen_check →
// mcpgen_schema over an in-memory MCP transport, exercising the same ops code
// path as the CLI.
func TestMCPServer_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end MCP test needs the go toolchain and module downloads")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := NewServer("test")
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Run(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	// tools/list: all four tools exposed
	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	names := make([]string, 0, len(tools.Tools))
	for _, tl := range tools.Tools {
		names = append(names, tl.Name)
	}
	assert.ElementsMatch(t, []string{"mcpgen_new", "mcpgen_add", "mcpgen_check", "mcpgen_schema"}, names)

	call := func(t *testing.T, name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		require.NoError(t, err)
		require.False(t, res.IsError, "tool %s returned IsError; content: %v", name, res.Content)
		require.NotEmpty(t, res.Content)
		text, ok := res.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		var out map[string]any
		require.NoError(t, json.Unmarshal([]byte(text.Text), &out))
		return out
	}

	dir := filepath.Join(t.TempDir(), "proj")

	// mcpgen_new (real, no check for speed of the loop start)
	out := call(t, "mcpgen_new", map[string]any{
		"dir": dir, "name": "viamcp", "module": "example.com/viamcp", "transport": "stdio", "noCheck": true,
	})
	assert.Equal(t, true, out["ok"])

	// mcpgen_add
	out = call(t, "mcpgen_add", map[string]any{
		"kind": "tool", "dir": dir,
		"spec": map[string]any{"id": "ping", "description": "probe", "readOnly": true},
	})
	assert.Equal(t, true, out["ok"])
	assert.Equal(t, "ping", out["result"].(map[string]any)["id"])

	// mcpgen_check (conformance off for stdio anyway; still runs tidy..test)
	out = call(t, "mcpgen_check", map[string]any{"dir": dir, "conformance": false, "timeoutSec": 300})
	assert.Equal(t, true, out["ok"], "check output: %v", out)
	res := out["result"].(map[string]any)
	assert.Equal(t, true, res["ok"])

	// mcpgen_schema
	out = call(t, "mcpgen_schema", map[string]any{})
	assert.Equal(t, "mcpgen", out["tool"])

	cancel()
	<-serverErr
}
