// User-owned file: mcpgen never overwrites this file (use --force to replace).

package resources

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Readme implements the "readme" resource.
//
// TODO: replace the stub with real content. The SDK fills in the resource URI
// and MIME type on the returned contents, so only the payload needs to be set.
func Readme(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{Text: "Welcome to your MCP server."},
		},
	}, nil
}
