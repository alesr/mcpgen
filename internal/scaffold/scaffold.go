// Package scaffold holds small helpers shared by the interactive UI and the
// CLI for building a default project config and printing summaries.
package scaffold

import (
	"fmt"
	"strings"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
)

// DefaultConfig returns a config with the default tool/prompt/resource stubs.
func DefaultConfig(transport string, port int, addTool, addResource, addPrompt bool) (*config.Config, string) {
	cfg := config.DefaultConfig(transport, port, addTool, addResource, addPrompt)
	return cfg, config.DefaultOutputDir
}

// PrintSummary prints a human summary of the config being generated.
func PrintSummary(cfg *config.Config, outDir string) {
	var features []string
	for _, t := range cfg.Tools {
		features = append(features, "tool:"+t.ID)
	}
	for _, p := range cfg.Prompts {
		features = append(features, "prompt:"+p.ID)
	}
	for _, r := range cfg.Resources {
		features = append(features, "resource:"+r.ID)
	}
	featureList := "none"
	if len(features) > 0 {
		featureList = strings.Join(features, ", ")
	}

	fmt.Printf(`
Summary
  Server:   %s (%s)
  Module:   %s
  Transport: %s
  Features: %s
  Output:   %s

`, cfg.Server.Name, cfg.Server.Version, cfg.Server.Module, cfg.Transport.Type, featureList, outDir)
}

// PrintInspectorHint prints how to open the generated server in the Inspector.
func PrintInspectorHint(outDir string, cfg *config.Config) {
	serverName := utils.DefaultServerName(cfg.Server.Name)

	if cfg.Transport.Type == "http" {
		fmt.Printf(`
Open in Inspector:
  cd %s
  go run ./cmd/%s &
  npx @modelcontextprotocol/inspector
  (set transport to streamable HTTP and use http://localhost:%d/mcp)
`, outDir, serverName, cfg.Transport.HTTPPort)
		return
	}

	fmt.Printf(`
Open in Inspector:
  cd %s
  npx @modelcontextprotocol/inspector
  (choose stdio and run: go run ./cmd/%s)
`, outDir, serverName)
}
