package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
)

// validateServerJSON structurally validates the MCP Registry descriptor
// (server.json) in dir: required fields, semver version, and an OCI package
// block (the only registry-publishable artifact path for a Go server).
func validateServerJSON(dir string) (msg, fix string, err error) {
	path := filepath.Join(dir, "server.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "server.json missing", fmt.Sprintf("regenerate with `mcpgen new --force` in %s to restore server.json", dir), fmt.Errorf("server.json not found")
		}
		return "", "", err
	}

	var doc struct {
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Packages    []struct {
			RegistryType string `json:"registryType"`
			Identifier   string `json:"identifier"`
			Version      string `json:"version"`
			Transport    struct {
				Type string `json:"type"`
			} `json:"transport"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Sprintf("server.json is not valid JSON: %v", err),
			fmt.Sprintf("fix server.json in %s (or regenerate with `mcpgen new --force`)", dir), err
	}

	var problems []string
	if strings.TrimSpace(doc.Name) == "" || strings.ContainsAny(doc.Name, " \t") {
		problems = append(problems, "name must be a non-empty reverse-DNS identifier (e.g. io.github.acme/weather)")
	}
	if strings.TrimSpace(doc.Description) == "" {
		problems = append(problems, "description is required")
	}
	if !semver.IsValid("v" + strings.TrimPrefix(doc.Version, "v")) {
		problems = append(problems, fmt.Sprintf("version %q is not valid semver", doc.Version))
	}
	if len(doc.Packages) == 0 {
		problems = append(problems, "at least one package is required")
	}
	for i, p := range doc.Packages {
		if p.RegistryType != "oci" {
			problems = append(problems, fmt.Sprintf("packages[%d].registryType must be \"oci\" for Go servers (got %q)", i, p.RegistryType))
		}
		if strings.TrimSpace(p.Identifier) == "" {
			problems = append(problems, fmt.Sprintf("packages[%d].identifier is required (the OCI image, e.g. ghcr.io/acme/weather)", i))
		}
		if !semver.IsValid("v" + strings.TrimPrefix(p.Version, "v")) {
			problems = append(problems, fmt.Sprintf("packages[%d].version %q is not valid semver", i, p.Version))
		}
		switch p.Transport.Type {
		case "stdio", "streamable-http", "sse", "":
		default:
			problems = append(problems, fmt.Sprintf("packages[%d].transport.type %q is not a known MCP transport", i, p.Transport.Type))
		}
	}

	if len(problems) > 0 {
		return fmt.Sprintf("server.json invalid: %s", strings.Join(problems, "; ")),
			fmt.Sprintf("fix the fields above in %s (see the MCP Registry server.json spec); publishing itself is done with the registry CLI", path),
			fmt.Errorf("server.json invalid: %d problem(s)", len(problems))
	}
	return "", "", nil
}
