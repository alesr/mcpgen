package config

import (
	"errors"
	"strings"
)

type (
	ServerConfig struct {
		Name        string `toml:"name"`
		Version     string `toml:"version"`
		Title       string `toml:"title"`
		Description string `toml:"description"`
		WebsiteURL  string `toml:"website_url"`
		Module      string `toml:"module"`
	}

	TransportConfig struct {
		Type     string `toml:"type"`
		HTTPPort int    `toml:"http_port"`
	}

	ToolConfig struct {
		ID           string `toml:"id"`
		Title        string `toml:"title"`
		Description  string `toml:"description"`
		InputSchema  string `toml:"input_schema"`
		OutputSchema string `toml:"output_schema"`
	}

	ResourceConfig struct {
		ID          string `toml:"id"`
		Title       string `toml:"title"`
		Description string `toml:"description"`
		URI         string `toml:"uri"`
		URITemplate string `toml:"uri_template"`
		MIMEType    string `toml:"mime_type"`
		Text        string `toml:"text"`
	}

	PromptConfig struct {
		ID          string                 `toml:"id"`
		Title       string                 `toml:"title"`
		Description string                 `toml:"description"`
		Role        string                 `toml:"role"`
		Template    string                 `toml:"template"`
		Arguments   []PromptArgumentConfig `toml:"argument"`
	}

	PromptArgumentConfig struct {
		Name        string `toml:"name"`
		Title       string `toml:"title"`
		Description string `toml:"description"`
		Required    bool   `toml:"required"`
	}
)

type Config struct {
	Server    ServerConfig     `toml:"server"`
	Tools     []ToolConfig     `toml:"tool"`
	Resources []ResourceConfig `toml:"resource"`
	Prompts   []PromptConfig   `toml:"prompt"`
	Transport TransportConfig  `toml:"transport"`
}

func (c *Config) Validate() error {
	errs := c.validateServer()
	errs = append(errs, c.validateTool()...)
	errs = append(errs, c.validateResource()...)
	errs = append(errs, c.validatePrompt()...)
	errs = append(errs, c.validateTransport()...)

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func titleCaseID(id string) string {
	parts := splitID(id)
	if len(parts) == 0 {
		return id
	}
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(strings.ToLower(p[1:]))
		}
		b.WriteString(" ")
	}
	return strings.TrimSpace(b.String())
}

func splitID(id string) []string {
	var parts []string
	var cur strings.Builder
	for _, r := range id {
		if r == '-' || r == '_' || r == ' ' {
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}
