package scaffold

import (
	"fmt"
	"strings"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
)

func DefaultConfig(outDir, transport string, port int, addTool, addResource, addPrompt bool) (*config.Config, string) {
	cfg := config.Config{
		Server: config.ServerConfig{
			Name:    config.DefaultServerName,
			Version: config.DefaultServerVersion,
			Module:  config.DefaultServerModule,
		},
		Transport: config.TransportConfig{
			Type:     strings.ToLower(strings.TrimSpace(transport)),
			HTTPPort: port,
		},
	}

	if addTool {
		cfg.Tools = []config.ToolConfig{{ID: config.DefaultToolID}}
	}

	if addPrompt {
		cfg.Prompts = []config.PromptConfig{{
			ID:       config.DefaultPromptID,
			Template: config.DefaultPromptTemplate,
		}}
	}

	if addResource {
		res := config.ResourceConfig{ID: config.DefaultResourceID, Text: config.DefaultResourceText}
		res.URI = "file:///" + config.DefaultResourceID
		cfg.Resources = []config.ResourceConfig{res}
	}
	return &cfg, outDir
}

func PrintSummary(cfg *config.Config, outDir string) {
	var features []string
	if len(cfg.Tools) > 0 {
		features = append(features, cfg.Tools[0].ID)
	}
	if len(cfg.Resources) > 0 {
		features = append(features, cfg.Resources[0].ID)
	}
	if len(cfg.Prompts) > 0 {
		features = append(features, cfg.Prompts[0].ID)
	}

	featureList := "none"
	if len(features) > 0 {
		featureList = strings.Join(features, ", ")
	}

	var resourceDetail string
	if len(cfg.Resources) > 0 {
		res := cfg.Resources[0]
		if res.URITemplate != "" {
			resourceDetail = fmt.Sprintf("  Resource template: %s\n", res.URITemplate)
		} else {
			resourceDetail = fmt.Sprintf("  Resource uri: %s\n", res.URI)
		}
	}

	fmt.Printf(`
Summary
  Server:   %s (%s)
  Module:   %s
  Features: %s
%s  Output:   %s

`, cfg.Server.Name, cfg.Server.Version, cfg.Server.Module, featureList, resourceDetail, outDir)
}

func PrintInspectorHint(outDir string, cfg *config.Config) {
	serverName := utils.DefaultServerName(cfg.Server.Name)

	if cfg.Transport.Type == "http" {
		fmt.Printf(`
Open in Inspector:
  cd %s
  go run ./cmd/%s &
  npx @modelcontextprotocol/inspector
  (In the UI, set transport to HTTP and use http://localhost:%d/mcp)

  Single command:
  	cd generated && npx @modelcontextprotocol/inspector
`, outDir, serverName, cfg.Transport.HTTPPort)
		return
	}

	fmt.Printf(`
Open in Inspector:
  cd %s
  npx @modelcontextprotocol/inspector
  (In the UI, choose stdio and run: go run ./cmd/%s)
`, outDir, serverName)
}
