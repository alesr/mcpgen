package adder

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"

	"github.com/alesr/mcpgen/internal/cli"
)

// Marker region names inside the registration files.
const (
	RegionTools     = "tools"
	RegionPrompts   = "prompts"
	RegionResources = "resources"
)

func beginMarker(region string) string { return "// mcpgen:begin " + region }
func endMarker(region string) string   { return "// mcpgen:end " + region }

// InsertIntoMarkerRegion inserts stmt before the `// mcpgen:end <region>`
// marker in filePath. Marker positions are located via go/parser + go/ast
// (never regex); the file is then rewritten with go/format.
func InsertIntoMarkerRegion(filePath, region, stmt string) error {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return cli.NewExitError(cli.ExitFailure, "add", cli.CodeIO, "read %s: %v", filePath, err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return cli.NewExitError(cli.ExitConflict, "add", cli.CodeConflict,
			"could not parse %s: %v; fix: restore the mcpgen:begin/end markers in %s", filePath, err, filePath)
	}

	beginPos, endPos, err := findMarkers(fset, f, region)
	if err != nil {
		return err
	}
	_ = beginPos

	// Insert the statement on its own line(s) immediately before the end
	// marker's line.
	endOffset := fset.Position(endPos).Offset
	lineStart := lineStartOffset(src, endOffset)

	var b strings.Builder
	b.Write(src[:lineStart])
	b.WriteString(stmt)
	b.WriteByte('\n')
	b.Write(src[lineStart:])

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return cli.NewExitError(cli.ExitConflict, "add", cli.CodeConflict,
			"insertion produced invalid Go in %s: %v; fix: restore the mcpgen:begin/end markers in %s", filePath, err, filePath)
	}
	if err := os.WriteFile(filePath, formatted, 0o644); err != nil {
		return cli.NewExitError(cli.ExitPermission, "add", cli.CodePermission, "write %s: %v", filePath, err)
	}
	return nil
}

// EnsureImport adds the import for pkgPath (package pkgName) to filePath if
// missing, so a registration inserted into a previously empty marker region
// still compiles. AST-based: parses the file, checks the import specs.
func EnsureImport(filePath, pkgPath, pkgName string) error {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == pkgPath {
			return nil // already imported
		}
	}

	// Find the import block and insert the new spec after "import (".
	importPos := f.Name.End() // fallback
	for _, d := range f.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			importPos = gd.Lparen + 1
			break
		}
	}
	offset := fset.Position(importPos).Offset
	var b strings.Builder
	b.Write(src[:offset])
	b.WriteString("\n\t" + strconv.Quote(pkgPath))
	b.Write(src[offset:])

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("adding import to %s: %w", filePath, err)
	}
	return os.WriteFile(filePath, formatted, 0o644)
}

// findMarkers locates the begin/end marker comments for region in the AST.
func findMarkers(fset *token.FileSet, f *ast.File, region string) (begin, end token.Pos, err error) {
	beginText, endText := beginMarker(region), endMarker(region)
	var foundBegin, foundEnd bool
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			text := strings.TrimSpace(c.Text)
			if text == beginText {
				begin, foundBegin = c.Pos(), true
			}
			if text == endText {
				end, foundEnd = c.Pos(), true
			}
		}
	}
	if !foundBegin || !foundEnd {
		return 0, 0, cli.NewExitError(cli.ExitConflict, "add", cli.CodeConflict,
			"mcpgen markers %q/%q not found; fix: restore the mcpgen:begin/end markers in the registration file",
			beginText, endText)
	}
	return begin, end, nil
}

func lineStartOffset(src []byte, offset int) int {
	i := offset
	for i > 0 && src[i-1] != '\n' {
		i--
	}
	return i
}

// RegionContains reports whether the marker region already mentions the given
// MCP name (e.g. `Name: "get_forecast"`).
func RegionContains(filePath, region, name string) (bool, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return false, err
	}
	_, endPos, err := findMarkers(fset, f, region)
	if err != nil {
		return false, err
	}
	beginPos, _, _ := findMarkers(fset, f, region)
	lo, hi := fset.Position(beginPos).Offset, fset.Position(endPos).Offset
	if lo > hi || hi > len(src) {
		return false, fmt.Errorf("bad marker offsets")
	}
	return strings.Contains(string(src[lo:hi]), fmt.Sprintf("Name: %q", name)), nil
}
