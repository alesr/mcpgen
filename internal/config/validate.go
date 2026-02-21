package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

func (c *Config) validateServer() []error {
	errs := make([]error, 0)
	if strings.TrimSpace(c.Server.Name) == "" {
		errs = append(errs, errors.New("server name"))
	}

	// server.version is optional

	if strings.TrimSpace(c.Server.Title) == "" {
		c.Server.Title = c.Server.Name
	}

	if strings.TrimSpace(c.Server.Description) == "" {
		c.Server.Description = "Generated MCP server."
	}

	if c.Server.Module == "" {
		c.Server.Module = defaultModulePath(c.Server.Name)
	}

	if !isValidModulePath(c.Server.Module) {
		errs = append(errs, fmt.Errorf("server.module is not a valid module path: %q", c.Server.Module))
	}
	return errs
}

func (c *Config) validateTool() []error {
	var errs []error
	toolIDs := make(map[string]bool)

	for i := range c.Tools {
		t := &c.Tools[i]
		if strings.TrimSpace(t.ID) == "" {
			errs = append(errs, fmt.Errorf("tool[%d].id is required", i))
			continue
		}

		if toolIDs[t.ID] {
			errs = append(errs, fmt.Errorf("tool id %q is duplicated", t.ID))
		}

		toolIDs[t.ID] = true
		if strings.TrimSpace(t.Title) == "" {
			t.Title = defaultToolTitle(t.ID)
		}

		if strings.TrimSpace(t.Description) == "" {
			t.Description = defaultToolDescription(t.ID)
		}

		if t.InputSchema == "" {
			t.InputSchema = `{"type":"object"}`
		}

		if err := validateSchemaObject(t.InputSchema, "tool "+t.ID+" input_schema"); err != nil {
			errs = append(errs, err)
		}

		if t.OutputSchema == "" {
			t.OutputSchema = `{"type":"object"}`
		}

		if err := validateSchemaObject(t.OutputSchema, "tool "+t.ID+" output_schema"); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (c *Config) validateResource() []error {
	errs := make([]error, 0)
	resourceIDs := make(map[string]bool)

	for i := range c.Resources {
		r := &c.Resources[i]
		if strings.TrimSpace(r.ID) == "" {
			errs = append(errs, fmt.Errorf("resource[%d].id is required", i))
			continue
		}

		if resourceIDs[r.ID] {
			errs = append(errs, fmt.Errorf("resource id %q is duplicated", r.ID))
		}

		resourceIDs[r.ID] = true
		if strings.TrimSpace(r.Title) == "" {
			r.Title = defaultResourceTitle(r.ID)
		}

		if strings.TrimSpace(r.Description) == "" {
			r.Description = defaultResourceDescription(r.ID)
		}

		if r.URI == "" && r.URITemplate == "" {
			errs = append(errs, fmt.Errorf("resource %q must set either uri or uri_template", r.ID))
		}

		if r.URI != "" && r.URITemplate != "" {
			errs = append(errs, fmt.Errorf("resource %q must set only one of uri or uri_template", r.ID))
		}

		if r.URI != "" {
			if err := validateURI(r.URI); err != nil {
				errs = append(errs, fmt.Errorf("resource %q uri invalid: %w", r.ID, err))
			}
		}

		if strings.TrimSpace(r.Text) == "" {
			r.Text = defaultResourceTextForID(r.ID)
		}
	}
	return errs
}

func (c *Config) validatePrompt() []error {
	errs := make([]error, 0)
	promptIDs := make(map[string]bool)

	for i := range c.Prompts {
		p := &c.Prompts[i]
		if strings.TrimSpace(p.ID) == "" {
			errs = append(errs, fmt.Errorf("prompt[%d].id is required", i))
			continue
		}

		if promptIDs[p.ID] {
			errs = append(errs, fmt.Errorf("prompt id %q is duplicated", p.ID))
		}

		promptIDs[p.ID] = true
		if strings.TrimSpace(p.Title) == "" {
			p.Title = defaultPromptTitle(p.ID)
		}

		if strings.TrimSpace(p.Description) == "" {
			p.Description = defaultPromptDescription(p.ID)
		}

		if strings.TrimSpace(p.Template) == "" {
			p.Template = defaultPromptTemplateForID(p.ID)
		}

		if strings.TrimSpace(p.Role) == "" {
			p.Role = "user"
		}

		for j, arg := range p.Arguments {
			if strings.TrimSpace(arg.Name) == "" {
				errs = append(
					errs,
					fmt.Errorf("prompt %q argument[%d].name is required", p.ID, j),
				)
			}
		}
	}
	return errs
}

func (c *Config) validateTransport() []error {
	errs := make([]error, 0)
	if strings.TrimSpace(c.Transport.Type) == "" {
		c.Transport.Type = DefaultTransport
	}

	if c.Transport.Type != "http" && c.Transport.Type != "stdio" {
		errs = append(errs, errors.New("transport type"))
		c.Transport.Type = DefaultTransport
	}

	if c.Transport.HTTPPort == 0 {
		c.Transport.HTTPPort = DefaultHTTPPort
	}

	if c.Transport.HTTPPort < 1 || c.Transport.HTTPPort > 65535 {
		errs = append(errs, errors.New("port"))
	}
	return errs
}

func validateSchemaObject(raw string, label string) error {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return fmt.Errorf("%s must be valid JSON: %w", label, err)
	}

	if t, ok := obj["type"]; ok && t != "object" {
		return fmt.Errorf("%s must have type=object", label)
	}
	return nil
}

func validateURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		return errors.New("missing scheme")
	}
	return nil
}

func isValidModulePath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return false
	}

	return !slices.ContainsFunc(parts, func(p string) bool {
		return p == "" || strings.Contains(p, " ")
	})
}
