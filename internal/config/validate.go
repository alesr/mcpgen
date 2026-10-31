package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/mod/module"
)

func (c *Config) validate() []error {
	errs := make([]error, 0)

	if c.Version != ConfigVersion {
		errs = append(errs, fmt.Errorf("%w: got %d, mcpgen v1 supports only version %d; fix: set \"version\": %d in %s",
			ErrConfigVersionUnsupported, c.Version, ConfigVersion, ConfigVersion, FileName))
	}

	errs = append(errs, c.validateServer()...)
	errs = append(errs, c.validateTransport()...)
	errs = append(errs, c.validateProtocol()...)
	errs = append(errs, c.validateTools()...)
	errs = append(errs, c.validatePrompts()...)
	errs = append(errs, c.validateResources()...)
	return errs
}

func (c *Config) validateServer() []error {
	errs := make([]error, 0)
	s := &c.Server

	if strings.TrimSpace(s.Name) == "" {
		errs = append(errs, ErrServerNameRequired)
	}
	if strings.TrimSpace(s.Title) == "" {
		s.Title = s.Name
	}
	if strings.TrimSpace(s.Version) == "" {
		s.Version = DefaultServerVersion
	}
	if strings.TrimSpace(s.Description) == "" {
		s.Description = DefaultServerDescription
	}
	if strings.TrimSpace(s.Instructions) == "" {
		s.Instructions = s.Description
	}
	if s.Module == "" {
		s.Module = DefaultModulePath(s.Name)
	}
	if err := module.CheckPath(s.Module); err != nil {
		errs = append(errs, fmt.Errorf("%w: %q", ErrServerModuleInvalid, s.Module))
	}
	return errs
}

func (c *Config) validateTransport() []error {
	errs := make([]error, 0)
	t := &c.Transport

	if strings.TrimSpace(t.Type) == "" {
		t.Type = DefaultTransport
	}
	t.Type = strings.ToLower(strings.TrimSpace(t.Type))
	if t.Type != "http" && t.Type != "stdio" {
		errs = append(errs, fmt.Errorf("%w: %q", ErrTransportTypeInvalid, t.Type))
		t.Type = DefaultTransport
	}
	if t.HTTPPort == 0 {
		t.HTTPPort = DefaultHTTPPort
	}
	if t.HTTPPort < 1 || t.HTTPPort > 65535 {
		errs = append(errs, fmt.Errorf("%w: %d", ErrTransportPortInvalid, t.HTTPPort))
	}
	return errs
}

func (c *Config) validateProtocol() []error {
	if c.Protocol == "" {
		c.Protocol = ProtocolCompat
	}
	switch c.Protocol {
	case ProtocolCompat, ProtocolLatestOnly:
		return nil
	default:
		return []error{fmt.Errorf("%w: %q (expected %q or %q)", ErrProtocolInvalid, c.Protocol, ProtocolCompat, ProtocolLatestOnly)}
	}
}

func (c *Config) validateTools() []error {
	errs := make([]error, 0)
	seen := make(map[string]bool, len(c.Tools))
	for i := range c.Tools {
		t := &c.Tools[i]
		if strings.TrimSpace(t.ID) == "" {
			errs = append(errs, fmt.Errorf("tools[%d].id is required", i))
			continue
		}
		if seen[t.ID] {
			errs = append(errs, fmt.Errorf("%w: tools[%d] id %q", ErrDuplicateID, i, t.ID))
			continue
		}
		seen[t.ID] = true

		if strings.TrimSpace(t.Title) == "" {
			t.Title = DefaultToolTitle(t.ID)
		}
		if strings.TrimSpace(t.Description) == "" {
			t.Description = DefaultToolDescription(t.ID)
		}
		if len(t.InputSchema) == 0 {
			t.InputSchema = json.RawMessage(defaultJSONSchemaObject)
		}
		if e := validateSchemaObject(t.InputSchema, "tools["+t.ID+"] inputSchema"); e != nil {
			errs = append(errs, e)
		}
		if len(t.OutputSchema) == 0 {
			t.OutputSchema = json.RawMessage(defaultJSONSchemaObject)
		}
		if e := validateSchemaObject(t.OutputSchema, "tools["+t.ID+"] outputSchema"); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

func (c *Config) validatePrompts() []error {
	errs := make([]error, 0)
	seen := make(map[string]bool, len(c.Prompts))
	for i := range c.Prompts {
		p := &c.Prompts[i]
		if strings.TrimSpace(p.ID) == "" {
			errs = append(errs, fmt.Errorf("prompts[%d].id is required", i))
			continue
		}
		if seen[p.ID] {
			errs = append(errs, fmt.Errorf("%w: prompts[%d] id %q", ErrDuplicateID, i, p.ID))
			continue
		}
		seen[p.ID] = true

		if strings.TrimSpace(p.Title) == "" {
			p.Title = DefaultPromptTitle(p.ID)
		}
		if strings.TrimSpace(p.Description) == "" {
			p.Description = DefaultPromptDescription(p.ID)
		}
		argSeen := make(map[string]bool, len(p.Arguments))
		for j, arg := range p.Arguments {
			if strings.TrimSpace(arg.Name) == "" {
				errs = append(errs, fmt.Errorf("prompts[%q].arguments[%d].name is required", p.ID, j))
				continue
			}
			if argSeen[arg.Name] {
				errs = append(errs, fmt.Errorf("prompts[%q].arguments[%d] duplicate name %q", p.ID, j, arg.Name))
			}
			argSeen[arg.Name] = true
		}
	}
	return errs
}

func (c *Config) validateResources() []error {
	errs := make([]error, 0)
	seen := make(map[string]bool, len(c.Resources))
	for i := range c.Resources {
		r := &c.Resources[i]
		if strings.TrimSpace(r.ID) == "" {
			errs = append(errs, fmt.Errorf("resources[%d].id is required", i))
			continue
		}
		if seen[r.ID] {
			errs = append(errs, fmt.Errorf("%w: resources[%d] id %q", ErrDuplicateID, i, r.ID))
			continue
		}
		seen[r.ID] = true

		if strings.TrimSpace(r.Title) == "" {
			r.Title = DefaultResourceTitle(r.ID)
		}
		if strings.TrimSpace(r.Description) == "" {
			r.Description = DefaultResourceDescription(r.ID)
		}
		if r.URI == "" && r.URITemplate == "" {
			errs = append(errs, fmt.Errorf("resources[%q] must set either uri or uriTemplate", r.ID))
		}
		if r.URI != "" && r.URITemplate != "" {
			errs = append(errs, fmt.Errorf("resources[%q] must set only one of uri or uriTemplate", r.ID))
		}
		if r.URI != "" {
			if e := validateURI(r.URI); e != nil {
				errs = append(errs, fmt.Errorf("resources[%q] uri invalid: %w", r.ID, e))
			}
		}
		if strings.TrimSpace(r.MIMEType) == "" {
			r.MIMEType = DefaultResourceMIMEType
		}
		if strings.TrimSpace(r.Text) == "" {
			r.Text = DefaultResourceTextForID(r.ID)
		}
	}
	return errs
}

func validateSchemaObject(raw json.RawMessage, label string) error {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("%s must be valid JSON: %w", label, err)
	}
	if t, ok := obj["type"]; ok && !typeAllowsObject(t) {
		return fmt.Errorf("%s must have type=object (or a type array containing \"object\")", label)
	}
	return nil
}

// typeAllowsObject reports whether a JSON Schema "type" value permits object:
// either the string "object" or a type array that contains it.
func typeAllowsObject(t any) bool {
	switch v := t.(type) {
	case string:
		return v == "object"
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok && s == "object" {
				return true
			}
		}
	}
	return false
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

// NormalizeJSONSchema returns a compact canonical JSON encoding of raw, or the
// default object schema when raw is empty. Invalid JSON is returned unchanged.
func NormalizeJSONSchema(raw json.RawMessage) string {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
		return defaultJSONSchemaObject
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}
