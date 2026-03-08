package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/mod/module"
)

func (c *Config) validateServer() []error {
	errs := make([]error, 0)
	if strings.TrimSpace(c.Server.Name) == "" {
		errs = append(errs, ErrServerNameRequired)
	}

	// server.version is optional

	if strings.TrimSpace(c.Server.Title) == "" {
		c.Server.Title = DefaultServerTitle
	}

	if strings.TrimSpace(c.Server.Description) == "" {
		c.Server.Description = DefaultServerDescription
	}

	if c.Server.Module == "" {
		c.Server.Module = defaultModulePath(c.Server.Name)
	}

	if err := module.CheckPath(c.Server.Module); err != nil {
		errs = append(errs, fmt.Errorf("%w: %q", ErrServerModuleInvalid, c.Server.Module))
	}
	return errs
}

func (c *Config) validateTool() []error {
	if c.Tool == nil {
		return nil
	}

	err := make([]error, 0)
	t := c.Tool
	if strings.TrimSpace(t.ID) == "" {
		return []error{errors.New("tool.id is required")}
	}

	if strings.TrimSpace(t.Title) == "" {
		t.Title = defaultToolTitle(t.ID)
	}

	if strings.TrimSpace(t.Description) == "" {
		t.Description = defaultToolDescription(t.ID)
	}

	if t.InputSchema == "" {
		t.InputSchema = defaultJSONSchemaObject
	}

	if e := validateSchemaObject(t.InputSchema, "tool "+t.ID+" input_schema"); e != nil {
		err = append(err, e)
	}

	if t.OutputSchema == "" {
		t.OutputSchema = defaultJSONSchemaObject
	}

	if e := validateSchemaObject(t.OutputSchema, "tool "+t.ID+" output_schema"); e != nil {
		err = append(err, e)
	}

	return err
}

func (c *Config) validateResource() []error {
	if c.Resource == nil {
		return nil
	}

	err := make([]error, 0)
	r := c.Resource
	if strings.TrimSpace(r.ID) == "" {
		return []error{errors.New("resource.id is required")}
	}

	if strings.TrimSpace(r.Title) == "" {
		r.Title = defaultResourceTitle(r.ID)
	}

	if strings.TrimSpace(r.Description) == "" {
		r.Description = defaultResourceDescription(r.ID)
	}

	if r.URI == "" && r.URITemplate == "" {
		err = append(err, fmt.Errorf("resource %q must set either uri or uri_template", r.ID))
	}

	if r.URI != "" && r.URITemplate != "" {
		err = append(err, fmt.Errorf("resource %q must set only one of uri or uri_template", r.ID))
	}

	if r.URI != "" {
		if e := validateURI(r.URI); e != nil {
			err = append(err, fmt.Errorf("resource %q uri invalid: %w", r.ID, e))
		}
	}

	if strings.TrimSpace(r.Text) == "" {
		r.Text = defaultResourceTextForID(r.ID)
	}

	return err
}

func (c *Config) validatePrompt() []error {
	if c.Prompt == nil {
		return nil
	}

	err := make([]error, 0)
	p := c.Prompt
	if strings.TrimSpace(p.ID) == "" {
		return []error{errors.New("prompt.id is required")}
	}

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
		p.Role = defaultPromptRole
	}

	for i, arg := range p.Arguments {
		if strings.TrimSpace(arg.Name) == "" {
			err = append(err, fmt.Errorf("prompt %q argument[%d].name is required", p.ID, i))
		}
	}

	return err
}

func (c *Config) validateTransport() []error {
	errs := make([]error, 0)
	if strings.TrimSpace(c.Transport.Type) == "" {
		c.Transport.Type = DefaultTransport
	}

	if c.Transport.Type != "http" && c.Transport.Type != "stdio" {
		errs = append(errs, fmt.Errorf("%w: %q", ErrTransportTypeInvalid, c.Transport.Type))
		c.Transport.Type = DefaultTransport
	}

	if c.Transport.HTTPPort == 0 {
		c.Transport.HTTPPort = DefaultHTTPPort
	}

	if c.Transport.HTTPPort < 1 || c.Transport.HTTPPort > 65535 {
		errs = append(errs, fmt.Errorf("%w: %d", ErrTransportPortInvalid, c.Transport.HTTPPort))
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
		return ErrURIMissingScheme
	}
	return nil
}
