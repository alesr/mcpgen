// User-owned file: mcpgen never overwrites this file (use --force to replace).

package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGreet(t *testing.T) {
	t.Parallel()
	res, out, err := Greet(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "greet"},
	}, GreetInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected a non-nil CallToolResult")
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("output does not marshal to JSON: %v", err)
	}
	if string(raw) == "null" {
		t.Fatal("expected a non-null JSON output")
	}
}
