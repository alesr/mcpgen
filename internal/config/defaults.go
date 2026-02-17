package config

import (
	"regexp"
	"strings"
)

const (
	DefaultServerName    = "example-mcp"
	DefaultServerVersion = "v0.1.0"
	DefaultServerModule  = "example.com/example-mcp"

	DefaultTransport = "stdio"
	DefaultHTTPPort  = 8080

	DefaultOutputDir = "./generated"

	DefaultToolID     = "greet"
	DefaultPromptID   = "welcome"
	DefaultResourceID = "readme"

	DefaultPromptTemplate = "Welcome!"
	DefaultResourceText   = "Welcome to your MCP server."
)

var nonModuleChar = regexp.MustCompile(`[^a-z0-9-]+`)

func defaultModulePath(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	base = nonModuleChar.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")

	if base == "" {
		base = "mcp-server"
	}
	return "example.com/" + base
}

func defaultToolTitle(id string) string {
	if strings.EqualFold(id, "greet") {
		return "Greet"
	}
	return titleCaseID(id)
}

func defaultToolDescription(id string) string {
	if strings.EqualFold(id, "greet") {
		return "Greets a user with a short welcome."
	}
	return "Tool stub for " + id + "."
}

func defaultResourceTitle(id string) string {
	if strings.EqualFold(id, "readme") {
		return "Readme"
	}
	return titleCaseID(id)
}

func defaultResourceDescription(id string) string {
	if strings.EqualFold(id, "readme") {
		return "A readme stub resource."
	}
	return "Resource stub for " + id + "."
}

func defaultResourceTextForID(id string) string {
	if strings.EqualFold(id, "readme") {
		return DefaultResourceText
	}
	return "This is the " + id + " stub."
}

func defaultPromptTitle(id string) string {
	if strings.EqualFold(id, "welcome") {
		return "Welcome"
	}
	return titleCaseID(id)
}

func defaultPromptDescription(id string) string {
	if strings.EqualFold(id, "welcome") {
		return "A friendly welcome prompt."
	}
	return "Prompt stub for " + id + "."
}

func defaultPromptTemplateForID(id string) string {
	if strings.EqualFold(id, "welcome") {
		return DefaultPromptTemplate
	}
	return "Prompt " + id + " stub"
}
