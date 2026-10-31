package generator

import (
	"fmt"
	"strings"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
)

type TemplateData struct {
	GeneratorVersion string
	Module           string
	ServerName       string // snake_case; used for cmd/<name>
	DisplayName      string
	Title            string
	Version          string
	Description      string
	Instructions     string
	Protocol         string
	RegistryName     string
	OCIIdentifier    string
	Transport        TransportData
	Tools            []ToolData
	Prompts          []PromptData
	Resources        []ResourceData
	Features         FeaturesData
}

type TransportData struct {
	Type     string
	HTTPPort int
}

type FeaturesData struct {
	Tasks  bool
	Skills bool
	OAuth  bool
}

type ToolData struct {
	GeneratorVersion string
	ID               string
	GoName           string
	FileName         string
	Title            string
	Description      string
	InputSchema      string
	OutputSchema     string
	ReadOnly         bool
	Idempotent       bool
}

type PromptData struct {
	GeneratorVersion string
	ID               string
	GoName           string
	FileName         string
	Title            string
	Description      string
	Arguments        []PromptArgData
	HasRequiredArgs  bool
}

type PromptArgData struct {
	Name        string
	Title       string
	Description string
	Required    bool
}

type ResourceData struct {
	GeneratorVersion string
	ID               string
	GoName           string
	FileName         string
	Title            string
	Description      string
	URI              string
	URITemplate      string
	MIMEType         string
	Text             string
	TestURI          string
	IsTemplate       bool
}

// BuildTemplateData converts a validated config into template data, rejecting
// identifier collisions (two IDs mapping to the same Go identifier or file).
func BuildTemplateData(cfg *config.Config, generatorVersion string) (TemplateData, error) {
	if cfg == nil {
		return TemplateData{}, fmt.Errorf("config is nil")
	}

	data := TemplateData{
		GeneratorVersion: generatorVersion,
		Module:           cfg.Server.Module,
		ServerName:       utils.DefaultServerName(cfg.Server.Name),
		DisplayName:      cfg.Server.Name,
		Title:            cfg.Server.Title,
		Version:          cfg.Server.Version,
		Description:      cfg.Server.Description,
		Instructions:     cfg.Server.Instructions,
		Protocol:         cfg.Protocol,
		Transport: TransportData{
			Type:     cfg.Transport.Type,
			HTTPPort: cfg.Transport.HTTPPort,
		},
		Features: FeaturesData{
			Tasks:  cfg.Features.Tasks,
			Skills: cfg.Features.Skills,
			OAuth:  cfg.Features.OAuth,
		},
	}
	data.RegistryName, data.OCIIdentifier = registryNames(cfg.Server.Module)

	toolIdents := map[string]string{}
	toolFiles := map[string]string{}
	for _, t := range cfg.Tools {
		td := ToolData{
			GeneratorVersion: generatorVersion,
			ID:               t.ID,
			GoName:           utils.GoIdent(t.ID),
			FileName:         utils.GoFileName(t.ID) + ".go",
			Title:            t.Title,
			Description:      t.Description,
			InputSchema:      config.NormalizeJSONSchema(t.InputSchema),
			OutputSchema:     config.NormalizeJSONSchema(t.OutputSchema),
			ReadOnly:         t.ReadOnly,
			Idempotent:       t.Idempotent,
		}
		if prev, ok := toolIdents[td.GoName]; ok {
			return TemplateData{}, fmt.Errorf("tool ids %q and %q both map to Go identifier %s", prev, t.ID, td.GoName)
		}
		if prev, ok := toolFiles[td.FileName]; ok {
			return TemplateData{}, fmt.Errorf("tool ids %q and %q both map to file %s", prev, t.ID, td.FileName)
		}
		toolIdents[td.GoName] = t.ID
		toolFiles[td.FileName] = t.ID
		data.Tools = append(data.Tools, td)
	}

	promptIdents := map[string]string{}
	promptFiles := map[string]string{}
	for _, p := range cfg.Prompts {
		pd := PromptData{
			GeneratorVersion: generatorVersion,
			ID:               p.ID,
			GoName:           utils.GoIdent(p.ID),
			FileName:         utils.GoFileName(p.ID) + ".go",
			Title:            p.Title,
			Description:      p.Description,
		}
		if prev, ok := promptIdents[pd.GoName]; ok {
			return TemplateData{}, fmt.Errorf("prompt ids %q and %q both map to Go identifier %s", prev, p.ID, pd.GoName)
		}
		if prev, ok := promptFiles[pd.FileName]; ok {
			return TemplateData{}, fmt.Errorf("prompt ids %q and %q both map to file %s", prev, p.ID, pd.FileName)
		}
		promptIdents[pd.GoName] = p.ID
		promptFiles[pd.FileName] = p.ID
		for _, arg := range p.Arguments {
			pd.Arguments = append(pd.Arguments, PromptArgData{
				Name:        arg.Name,
				Title:       arg.Title,
				Description: arg.Description,
				Required:    arg.Required,
			})
			if arg.Required {
				pd.HasRequiredArgs = true
			}
		}
		data.Prompts = append(data.Prompts, pd)
	}

	resIdents := map[string]string{}
	resFiles := map[string]string{}
	for _, r := range cfg.Resources {
		testURI := r.URI
		if r.URITemplate != "" {
			testURI = strings.ReplaceAll(r.URITemplate, "{id}", r.ID)
		}
		rd := ResourceData{
			GeneratorVersion: generatorVersion,
			ID:               r.ID,
			GoName:           utils.GoIdent(r.ID),
			FileName:         utils.GoFileName(r.ID) + ".go",
			Title:            r.Title,
			Description:      r.Description,
			URI:              r.URI,
			URITemplate:      r.URITemplate,
			MIMEType:         r.MIMEType,
			Text:             r.Text,
			TestURI:          testURI,
			IsTemplate:       r.URITemplate != "",
		}
		if prev, ok := resIdents[rd.GoName]; ok {
			return TemplateData{}, fmt.Errorf("resource ids %q and %q both map to Go identifier %s", prev, r.ID, rd.GoName)
		}
		if prev, ok := resFiles[rd.FileName]; ok {
			return TemplateData{}, fmt.Errorf("resource ids %q and %q both map to file %s", prev, r.ID, rd.FileName)
		}
		resIdents[rd.GoName] = r.ID
		resFiles[rd.FileName] = r.ID
		data.Resources = append(data.Resources, rd)
	}

	return data, nil
}

// registryNames derives the MCP Registry server name (reverse-DNS of the
// module host) and a default OCI identifier from the Go module path.
// github.com/acme/weather -> io.github.acme/weather, ghcr.io/acme/weather
func registryNames(module string) (registryName, ociIdentifier string) {
	parts := strings.Split(module, "/")
	host := parts[0]
	rest := strings.Join(parts[1:], "/")

	// Registry names use the reverse-DNS convention with well-known
	// forges mapped to their registry prefix (github.com -> io.github).
	prefix := map[string]string{
		"github.com":    "io.github",
		"gitlab.com":    "io.gitlab",
		"bitbucket.org": "org.bitbucket",
		"codeberg.org":  "org.codeberg",
	}[host]
	if prefix == "" {
		labels := strings.Split(host, ".")
		for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
			labels[i], labels[j] = labels[j], labels[i]
		}
		prefix = strings.Join(labels, ".")
	}
	registryName = prefix
	if rest != "" {
		if _, isForge := map[string]bool{"github.com": true, "gitlab.com": true, "bitbucket.org": true, "codeberg.org": true}[host]; isForge {
			// io.github.<owner>/<repo...>
			parts := strings.SplitN(rest, "/", 2)
			registryName = prefix + "." + parts[0]
			if len(parts) == 2 {
				registryName += "/" + parts[1]
			}
		} else {
			registryName = prefix + "/" + rest
		}
	}

	switch host {
	case "github.com":
		ociIdentifier = "ghcr.io/" + rest
	default:
		ociIdentifier = strings.ToLower(module)
	}
	if rest == "" {
		ociIdentifier = strings.ToLower(host)
	}
	return registryName, ociIdentifier
}
