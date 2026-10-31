// Package config defines the mcpgen project configuration model (mcpgen.json, version 1).
//
// mcpgen.json is the single source of truth for a generated project. It is
// consumed by `mcpgen new`, `mcpgen add`, and `mcpgen check`.
package config

import (
	"encoding/json"
	"errors"
)

// ConfigVersion is the only project config format version supported by mcpgen v1.
const ConfigVersion = 1

// FileName is the name of the project config file inside a generated project.
const FileName = "mcpgen.json"

// ManifestDir / ManifestFile locate the ownership manifest inside a project.
const (
	ManifestDir  = ".mcpgen"
	ManifestFile = "manifest.json"
)

type (
	// Config is the root of mcpgen.json.
	Config struct {
		Version   int        `json:"version"`
		Server    Server     `json:"server"`
		Transport Transport  `json:"transport"`
		Protocol  string     `json:"protocol,omitempty"`
		Tools     []Tool     `json:"tools,omitempty"`
		Prompts   []Prompt   `json:"prompts,omitempty"`
		Resources []Resource `json:"resources,omitempty"`
		Features  Features   `json:"features,omitempty"`
	}

	Server struct {
		Name         string `json:"name"`
		Title        string `json:"title,omitempty"`
		Version      string `json:"version,omitempty"`
		Module       string `json:"module"`
		Description  string `json:"description,omitempty"`
		Instructions string `json:"instructions,omitempty"`
	}

	Transport struct {
		Type     string `json:"type"`
		HTTPPort int    `json:"httpPort,omitempty"`
	}

	// Tool describes one MCP tool. InputSchema/OutputSchema hold raw JSON
	// Schema documents.
	Tool struct {
		ID           string          `json:"id"`
		Title        string          `json:"title,omitempty"`
		Description  string          `json:"description,omitempty"`
		InputSchema  json.RawMessage `json:"inputSchema,omitempty"`
		OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
		ReadOnly     bool            `json:"readOnly,omitempty"`
		Idempotent   bool            `json:"idempotent,omitempty"`
	}

	Prompt struct {
		ID          string           `json:"id"`
		Title       string           `json:"title,omitempty"`
		Description string           `json:"description,omitempty"`
		Arguments   []PromptArgument `json:"arguments,omitempty"`
	}

	PromptArgument struct {
		Name        string `json:"name"`
		Title       string `json:"title,omitempty"`
		Description string `json:"description,omitempty"`
		Required    bool   `json:"required,omitempty"`
	}

	Resource struct {
		ID          string `json:"id"`
		Title       string `json:"title,omitempty"`
		Description string `json:"description,omitempty"`
		URI         string `json:"uri,omitempty"`
		URITemplate string `json:"uriTemplate,omitempty"`
		MIMEType    string `json:"mimeType,omitempty"`
		Text        string `json:"text,omitempty"`
	}

	Features struct {
		Tasks  bool `json:"tasks,omitempty"`
		Skills bool `json:"skills,omitempty"`
		OAuth  bool `json:"oauth,omitempty"`
	}
)

// Protocol modes for ServerOptions.SupportedProtocolVersions.
const (
	ProtocolCompat     = "compat"      // SDK default set; negotiates down to legacy versions
	ProtocolLatestOnly = "latest-only" // only MCP 2026-07-28
)

// Validate fills defaults and returns all validation errors joined.
func (c *Config) Validate() error {
	errs := c.validate()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// FindTool returns the tool with the given ID, or nil.
func (c *Config) FindTool(id string) *Tool {
	for i := range c.Tools {
		if c.Tools[i].ID == id {
			return &c.Tools[i]
		}
	}
	return nil
}

// FindPrompt returns the prompt with the given ID, or nil.
func (c *Config) FindPrompt(id string) *Prompt {
	for i := range c.Prompts {
		if c.Prompts[i].ID == id {
			return &c.Prompts[i]
		}
	}
	return nil
}

// FindResource returns the resource with the given ID, or nil.
func (c *Config) FindResource(id string) *Resource {
	for i := range c.Resources {
		if c.Resources[i].ID == id {
			return &c.Resources[i]
		}
	}
	return nil
}
