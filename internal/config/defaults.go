package config

import (
	"regexp"
	"strings"

	"github.com/alesr/mcpgen/internal/pkg/utils"
	"golang.org/x/mod/module"
)

const (
	DefaultServerName        = "example-mcp"
	DefaultServerVersion     = "0.1.0"
	DefaultServerModule      = "example.com/example-mcp"
	DefaultServerDescription = "Generated MCP server."

	DefaultTransport = "stdio"
	DefaultHTTPPort  = 8080

	DefaultOutputDir = "./generated"

	DefaultToolID     = "greet"
	DefaultPromptID   = "welcome"
	DefaultResourceID = "readme"

	DefaultResourceMIMEType = "text/plain"
	DefaultResourceText     = "Welcome to your MCP server."

	defaultModuleHost = "example.com"

	defaultJSONSchemaObject = `{"type":"object"}`

	defaultGreetTitle         = "Greet"
	defaultGreetDescription   = "Greets a user with a short welcome."
	defaultReadmeTitle        = "Readme"
	defaultReadmeDescription  = "A readme stub resource."
	defaultWelcomeTitle       = "Welcome"
	defaultWelcomeDescription = "A friendly welcome prompt."
)

var nonModuleChar = regexp.MustCompile(`[^a-z0-9-]+`)

// DefaultModulePath derives a default Go module path from a server name.
func DefaultModulePath(name string) string {
	if strings.TrimSpace(name) == "" {
		return DefaultServerModule
	}

	base := strings.ToLower(strings.TrimSpace(name))
	base = nonModuleChar.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")

	if base == "" {
		return DefaultServerModule
	}

	path := defaultModuleHost + "/" + base
	if err := module.CheckPath(path); err != nil {
		return DefaultServerModule
	}
	return path
}

func DefaultToolTitle(id string) string {
	if strings.EqualFold(id, DefaultToolID) {
		return defaultGreetTitle
	}
	return utils.TitleCaseID(id)
}

func DefaultToolDescription(id string) string {
	if strings.EqualFold(id, DefaultToolID) {
		return defaultGreetDescription
	}
	return "Tool stub for " + id + "."
}

func DefaultResourceTitle(id string) string {
	if strings.EqualFold(id, DefaultResourceID) {
		return defaultReadmeTitle
	}
	return utils.TitleCaseID(id)
}

func DefaultResourceDescription(id string) string {
	if strings.EqualFold(id, DefaultResourceID) {
		return defaultReadmeDescription
	}
	return "Resource stub for " + id + "."
}

func DefaultResourceTextForID(id string) string {
	if strings.EqualFold(id, DefaultResourceID) {
		return DefaultResourceText
	}
	return "This is the " + id + " stub."
}

func DefaultPromptTitle(id string) string {
	if strings.EqualFold(id, DefaultPromptID) {
		return defaultWelcomeTitle
	}
	return utils.TitleCaseID(id)
}

func DefaultPromptDescription(id string) string {
	if strings.EqualFold(id, DefaultPromptID) {
		return defaultWelcomeDescription
	}
	return "Prompt stub for " + id + "."
}

// DefaultConfig returns a config with the default tool, prompt, and resource
// stubs enabled, mirroring the interactive flow defaults.
func DefaultConfig(transport string, port int, withTool, withResource, withPrompt bool) *Config {
	cfg := &Config{
		Version: ConfigVersion,
		Server: Server{
			Name:    DefaultServerName,
			Version: DefaultServerVersion,
			Module:  DefaultServerModule,
		},
		Transport: Transport{
			Type:     strings.ToLower(strings.TrimSpace(transport)),
			HTTPPort: port,
		},
		Protocol: ProtocolCompat,
	}

	if withTool {
		cfg.Tools = append(cfg.Tools, Tool{ID: DefaultToolID, ReadOnly: true})
	}
	if withPrompt {
		cfg.Prompts = append(cfg.Prompts, Prompt{ID: DefaultPromptID})
	}
	if withResource {
		res := Resource{ID: DefaultResourceID, URI: "file:///" + DefaultResourceID, Text: DefaultResourceText}
		cfg.Resources = append(cfg.Resources, res)
	}
	return cfg
}
