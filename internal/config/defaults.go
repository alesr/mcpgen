package config

import (
	"regexp"
	"strings"

	"github.com/alesr/mcpgen/internal/pkg/utils"
	"golang.org/x/mod/module"
)

const (
	DefaultServerName        = "example-mcp"
	DefaultServerTitle       = DefaultServerName
	DefaultServerVersion     = "v0.1.0"
	DefaultServerModule      = "example.com/example-mcp"
	DefaultServerDescription = "Generated MCP server."

	DefaultTransport = "stdio"
	DefaultHTTPPort  = 8080

	DefaultOutputDir = "./generated"

	DefaultToolID     = "greet"
	DefaultPromptID   = "welcome"
	DefaultResourceID = "readme"

	DefaultPromptTemplate = "Welcome!"
	DefaultResourceText   = "Welcome to your MCP server."

	defaultModuleHost = "example.com"
	defaultPromptRole = "user"

	defaultJSONSchemaObject = `{"type":"object"}`

	defaultGreetTitle         = "Greet"
	defaultGreetDescription   = "Greets a user with a short welcome."
	defaultReadmeTitle        = "Readme"
	defaultReadmeDescription  = "A readme stub resource."
	defaultWelcomeTitle       = "Welcome"
	defaultWelcomeDescription = "A friendly welcome prompt."
)

var nonModuleChar = regexp.MustCompile(`[^a-z0-9-]+`)

func defaultModulePath(name string) string {
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

func defaultToolTitle(id string) string {
	if strings.EqualFold(id, DefaultToolID) {
		return defaultGreetTitle
	}
	return utils.TitleCaseID(id)
}

func defaultToolDescription(id string) string {
	if strings.EqualFold(id, DefaultToolID) {
		return defaultGreetDescription
	}
	return "Tool stub for " + id + "."
}

func defaultResourceTitle(id string) string {
	if strings.EqualFold(id, DefaultResourceID) {
		return defaultReadmeTitle
	}
	return utils.TitleCaseID(id)
}

func defaultResourceDescription(id string) string {
	if strings.EqualFold(id, DefaultResourceID) {
		return defaultReadmeDescription
	}
	return "Resource stub for " + id + "."
}

func defaultResourceTextForID(id string) string {
	if strings.EqualFold(id, DefaultResourceID) {
		return DefaultResourceText
	}
	return "This is the " + id + " stub."
}

func defaultPromptTitle(id string) string {
	if strings.EqualFold(id, DefaultPromptID) {
		return defaultWelcomeTitle
	}
	return utils.TitleCaseID(id)
}

func defaultPromptDescription(id string) string {
	if strings.EqualFold(id, DefaultPromptID) {
		return defaultWelcomeDescription
	}
	return "Prompt stub for " + id + "."
}

func defaultPromptTemplateForID(id string) string {
	if strings.EqualFold(id, DefaultPromptID) {
		return DefaultPromptTemplate
	}
	return "Prompt " + id + " stub"
}
