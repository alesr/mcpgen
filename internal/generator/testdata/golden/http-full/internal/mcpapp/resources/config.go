// User-owned file: mcpgen never overwrites this file (use --force to replace).

package resources

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config implements the "config" resource template.
//
// TODO: extract the URI template variables (name)
// from req.Params.URI (matched against "file:///config/{name}") and return real
// content. The SDK fills in the resource URI and MIME type on the returned
// contents, so only the payload needs to be set here.
func Config(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{Text: "{}"},
		},
	}, nil
}
