// User-owned file: mcpgen never overwrites this file (use --force to replace).

package prompts

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestReport(t *testing.T) {
	t.Parallel()
	args := map[string]string{}
	args["topic"] = "test"
	res, err := Report(context.Background(), &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Name: "report", Arguments: args}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res.Messages) == 0 {
		t.Fatal("expected messages")
	}
}
