package config

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/jsonschema-go/jsonschema"
)

//go:embed schema.json
var schemaFS embed.FS

// SchemaJSON returns the versioned JSON Schema describing mcpgen.json.
func SchemaJSON() (json.RawMessage, error) {
	raw, err := schemaFS.ReadFile("schema.json")
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("embedded schema.json is invalid: %w", err)
	}
	return json.RawMessage(raw), nil
}

// Load reads and validates the mcpgen.json in dir: JSON parse, validation
// against the embedded versioned JSON Schema, and semantic validation.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, FileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w in %s; fix: run this command from an mcpgen-generated project root (or use `mcpgen new` to create one)", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Version gate first, so unknown versions keep their sentinel error
	// (mapped to exit code 2 by callers) even though the JSON Schema also
	// constrains version to 1.
	var head struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("parse %s: %w; fix: ensure %s is valid JSON", path, err, FileName)
	}
	if head.Version != ConfigVersion {
		return nil, fmt.Errorf("%w: got %d, mcpgen v1 supports only version %d; fix: set \"version\": %d in %s",
			ErrConfigVersionUnsupported, head.Version, ConfigVersion, ConfigVersion, FileName)
	}

	if err := ValidateAgainstSchema(raw); err != nil {
		return nil, fmt.Errorf("invalid %s: does not validate against the mcpgen JSON Schema: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}
	return &cfg, nil
}

// ValidateAgainstSchema validates raw mcpgen.json content against the embedded
// versioned JSON Schema (draft 2020-12).
func ValidateAgainstSchema(raw []byte) error {
	schemaRaw, err := SchemaJSON()
	if err != nil {
		return err
	}
	var sch jsonschema.Schema
	if err := json.Unmarshal(schemaRaw, &sch); err != nil {
		return fmt.Errorf("embedded schema.json: %w", err)
	}
	resolved, err := sch.Resolve(nil)
	if err != nil {
		return fmt.Errorf("resolve embedded schema: %w", err)
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return err
	}
	if err := resolved.Validate(instance); err != nil {
		return err
	}
	return nil
}

// LoadExisting is like Load but reports a missing file as ErrConfigNotFound
// without the wrapped fix text (for callers that add their own hint).
func LoadExisting(dir string) (*Config, error) {
	cfg, err := Load(dir)
	if err != nil && errors.Is(err, ErrConfigNotFound) {
		return nil, ErrConfigNotFound
	}
	return cfg, err
}

// Save writes cfg as mcpgen.json in dir with deterministic formatting.
func Save(dir string, cfg *Config) error {
	raw, err := Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, FileName), raw, 0o644)
}

// Marshal encodes cfg deterministically (stable field order, 2-space indent,
// trailing newline).
func Marshal(cfg *Config) ([]byte, error) {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", FileName, err)
	}
	return append(raw, '\n'), nil
}

// Path returns the path of the config file inside dir.
func Path(dir string) string { return filepath.Join(dir, FileName) }

// ManifestPath returns the path of the ownership manifest inside dir.
func ManifestPath(dir string) string { return filepath.Join(dir, ManifestDir, ManifestFile) }

// IsProject reports whether dir looks like an mcpgen-generated project.
func IsProject(dir string) bool {
	if _, err := os.Stat(Path(dir)); err != nil {
		return false
	}
	_, err := os.Stat(ManifestPath(dir))
	return err == nil
}
