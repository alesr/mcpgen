package generator

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/alesr/mcpgen/internal/config"
	"github.com/alesr/mcpgen/internal/pkg/utils"
)

type TemplateData struct {
	Module            string
	ServerName        string
	ServerDisplayName string
	ServerTitle       string
	ServerVersion     string
	Instructions      string
	Transport         TransportData
	Tools             []ToolData
	Resources         []ResourceData
	Prompts           []PromptData
}

type TransportData struct {
	Type     string
	HTTPPort int
}

type ToolData struct {
	ID           string
	GoName       string
	Title        string
	Description  string
	InputSchema  string
	OutputSchema string
}

type ResourceData struct {
	ID          string
	GoName      string
	Title       string
	Description string
	URI         string
	URITemplate string
	MIMEType    string
	Text        string
	TestURI     string
}

type PromptData struct {
	ID           string
	GoName       string
	Title        string
	Description  string
	Template     string
	Role         string
	Arguments    []PromptArgData
	RequiredArgs []string
}

type PromptArgData struct {
	Name        string
	Title       string
	Description string
	Required    bool
}

func buildTemplateData(cfg *config.Config, serverName string) TemplateData {
	data := TemplateData{
		Module:            cfg.Server.Module,
		ServerName:        serverName,
		ServerDisplayName: cfg.Server.Name,
		ServerTitle:       cfg.Server.Title,
		ServerVersion:     cfg.Server.Version,
		Instructions:      cfg.Server.Description,
		Transport: TransportData{
			Type:     cfg.Transport.Type,
			HTTPPort: cfg.Transport.HTTPPort,
		},
	}

	for _, tool := range cfg.Tools {
		data.Tools = append(data.Tools, ToolData{
			ID:           tool.ID,
			GoName:       utils.GoIdent(tool.ID),
			Title:        tool.Title,
			Description:  tool.Description,
			InputSchema:  normalizeJSON(tool.InputSchema),
			OutputSchema: normalizeJSON(tool.OutputSchema),
		})
	}

	for _, res := range cfg.Resources {
		testURI := res.URI
		if res.URITemplate != "" {
			testURI = strings.ReplaceAll(res.URITemplate, "{id}", res.ID)
		}

		data.Resources = append(data.Resources, ResourceData{
			ID:          res.ID,
			GoName:      utils.GoIdent(res.ID),
			Title:       res.Title,
			Description: res.Description,
			URI:         res.URI,
			URITemplate: res.URITemplate,
			MIMEType:    res.MIMEType,
			Text:        res.Text,
			TestURI:     testURI,
		})
	}

	for _, prompt := range cfg.Prompts {
		p := PromptData{
			ID:          prompt.ID,
			GoName:      utils.GoIdent(prompt.ID),
			Title:       prompt.Title,
			Description: prompt.Description,
			Template:    prompt.Template,
			Role:        prompt.Role,
		}

		for _, arg := range prompt.Arguments {
			p.Arguments = append(p.Arguments, PromptArgData{
				Name:        arg.Name,
				Title:       arg.Title,
				Description: arg.Description,
				Required:    arg.Required,
			})

			if arg.Required {
				p.RequiredArgs = append(p.RequiredArgs, arg.Name)
			}
		}
		sort.Strings(p.RequiredArgs)
		data.Prompts = append(data.Prompts, p)
	}
	return data
}

// normalizeJSON returns a canonical JSON string when possible
// if raw is blank, it returns "{}". If cannot be parsed or re-marshaled,
// it returns raw unchanged to preserve the original input.
func normalizeJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}

	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}

	formatted, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return string(formatted)
}
