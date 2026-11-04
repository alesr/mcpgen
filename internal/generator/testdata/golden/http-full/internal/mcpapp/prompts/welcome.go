// User-owned file: mcpgen never overwrites this file (use --force to replace).

package prompts

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Welcome implements the "welcome" prompt.
//
// TODO: replace the stub with real prompt logic.
func Welcome(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx
	args := req.Params.Arguments
	if args == nil {
		args = map[string]string{}
	}
	var sb strings.Builder
	sb.WriteString("A friendly welcome prompt.")
	return &mcp.GetPromptResult{
		Description: "A friendly welcome prompt.",
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: sb.String()}},
		},
	}, nil
}
