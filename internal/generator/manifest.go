package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/alesr/mcpgen/internal/config"
)

// Ownership markers used in generated file headers and the manifest.
const (
	OwnerMCPGen = "mcpgen"
	OwnerUser   = "user"
)

// Manifest is the ownership manifest stored at .mcpgen/manifest.json inside a
// generated project. It records, per file, who owns it and the sha256 of the
// content at generation time — used to detect user edits on regeneration.
type Manifest struct {
	Generator string                   `json:"generator"`
	Version   string                   `json:"version"`
	Files     map[string]ManifestEntry `json:"files"`
}

type ManifestEntry struct {
	Owner  string `json:"owner"`
	SHA256 string `json:"sha256"`
}

// LoadManifest reads the manifest in dir; a missing manifest yields an empty one.
func LoadManifest(dir string) (*Manifest, error) {
	path := config.ManifestPath(dir)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Manifest{Files: map[string]ManifestEntry{}}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m.Files == nil {
		m.Files = map[string]ManifestEntry{}
	}
	return &m, nil
}

// SaveManifest writes the manifest deterministically (sorted keys).
func SaveManifest(dir string, m *Manifest) error {
	if err := os.MkdirAll(filepath.Join(dir, config.ManifestDir), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(config.ManifestPath(dir), raw, 0o644)
}

// HashFile returns the hex sha256 of the file at path.
func HashFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// SortedPaths returns the manifest file paths in sorted order.
func (m *Manifest) SortedPaths() []string {
	paths := make([]string, 0, len(m.Files))
	for p := range m.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
