package generator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/alesr/mcpgen/internal/pkg/utils"
)

// structDef builds a named Go struct type definition from a JSON Schema object
// (a schema whose root type is "object"). It is a typed scaffold for tool
// handlers: the raw schema stored in mcpgen.json remains the authoritative
// source used for input/output validation by the SDK; the generated struct is
// what the handler receives and returns.
//
// Properties that map cleanly to Go types become fields; nested objects are
// rendered as anonymous inline structs; anything that cannot be represented as
// a single Go type (unions, $ref, unconstrained objects/arrays) falls back to
// any or map[string]any.
func structDef(name, rawSchema string) string {
	var schema map[string]any
	if err := json.Unmarshal([]byte(rawSchema), &schema); err != nil {
		schema = map[string]any{}
	}
	fields := objectFields(schema)
	var b strings.Builder
	b.WriteString("type " + name + " struct {")
	if len(fields) == 0 {
		b.WriteString("}")
		return b.String()
	}
	b.WriteString("\n")
	writeFields(&b, fields)
	b.WriteString("}")
	return b.String()
}

// goField is one struct field derived from a JSON Schema property.
type goField struct {
	Name        string // exported Go field name
	JSONName    string // JSON key (json tag)
	Type        string // Go type expression
	Description string // schema description, used as a doc comment
	OmitEmpty   bool   // whether the json tag carries ",omitempty"
}

// objectFields extracts the ordered struct fields for a schema's properties.
// Property order is sorted for deterministic output; the "required" list drives
// the omitempty flag.
func objectFields(schema map[string]any) []goField {
	props, _ := schema["properties"].(map[string]any)
	required := requiredSet(schema)
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	used := map[string]int{}
	fields := make([]goField, 0, len(keys))
	for _, k := range keys {
		ps, _ := props[k].(map[string]any)
		if ps == nil {
			ps = map[string]any{}
		}
		fields = append(fields, goField{
			Name:        uniqueGoIdent(utils.GoIdent(k), used),
			JSONName:    k,
			Type:        goType(ps),
			Description: descriptionOf(ps),
			OmitEmpty:   !required[k],
		})
	}
	return fields
}

// writeFields appends field declarations (doc comment + field + tag) to b.
func writeFields(b *strings.Builder, fields []goField) {
	for _, f := range fields {
		if f.Description != "" {
			b.WriteString("\t// " + sanitizeComment(f.Description) + "\n")
		}
		tag := f.JSONName
		if f.OmitEmpty {
			tag += ",omitempty"
		}
		fmt.Fprintf(b, "\t%s %s `json:%q`\n", f.Name, f.Type, tag)
	}
}

// goType maps a property schema to a Go type expression.
func goType(schema map[string]any) string {
	switch t := schema["type"].(type) {
	case string:
		switch t {
		case "string":
			return "string"
		case "integer":
			return "int64"
		case "number":
			return "float64"
		case "boolean":
			return "bool"
		case "null":
			return "any"
		case "object":
			return inlineStructType(schema)
		case "array":
			return arrayType(schema)
		default:
			return "any"
		}
	case []any:
		// A union type is not representable as a single Go type without a
		// custom unmarshaler; fall back to any.
		return "any"
	default:
		// No explicit type: infer from keywords, else fall back to any.
		if _, ok := schema["properties"].(map[string]any); ok {
			return inlineStructType(schema)
		}
		if _, ok := schema["items"].(map[string]any); ok {
			return arrayType(schema)
		}
		return "any"
	}
}

// inlineStructType renders a nested object as an anonymous struct type, or
// map[string]any when it has no named properties (unconstrained object).
func inlineStructType(schema map[string]any) string {
	fields := objectFields(schema)
	if len(fields) == 0 {
		return "map[string]any"
	}
	var b strings.Builder
	b.WriteString("struct {\n")
	writeFields(&b, fields)
	b.WriteString("}")
	return b.String()
}

// arrayType maps an array schema to a Go slice type.
func arrayType(schema map[string]any) string {
	items, _ := schema["items"].(map[string]any)
	if items == nil {
		return "[]any"
	}
	return "[]" + goType(items)
}

// requiredSet returns the set of property names listed in the schema's
// "required" array.
func requiredSet(schema map[string]any) map[string]bool {
	out := map[string]bool{}
	req, _ := schema["required"].([]any)
	for _, r := range req {
		if s, ok := r.(string); ok {
			out[s] = true
		}
	}
	return out
}

// descriptionOf returns the schema description, trimmed.
func descriptionOf(schema map[string]any) string {
	if d, ok := schema["description"].(string); ok {
		return strings.TrimSpace(d)
	}
	return ""
}

// uniqueGoIdent returns base, or base suffixed with a number if base has
// already been used, keeping generated field names unique.
func uniqueGoIdent(base string, used map[string]int) string {
	if used[base] == 0 {
		used[base] = 1
		return base
	}
	used[base]++
	return fmt.Sprintf("%s%d", base, used[base])
}

// sanitizeComment collapses a schema description onto a single line so it can
// be emitted as a Go doc comment.
func sanitizeComment(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
