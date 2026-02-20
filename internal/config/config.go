package config

import (
	"errors"
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
		joined := make([]error, 0, len(errs))
		for _, err := range errs {
			joined = append(joined, errors.New(err))
		}
		return errors.Join(joined...)
	}
	return nil
}
