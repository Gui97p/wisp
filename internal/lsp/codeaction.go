package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/parser"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	suggestionPattern = regexp.MustCompile(`did you mean (\S+?)\?`)
	undeclaredPattern = regexp.MustCompile(`identifier (\w+) not declared in this scope`)
	unknownTypeModule = regexp.MustCompile(`unknown type (\w+)\.\w+`)
	missingImplements = regexp.MustCompile(`(\w+) has the methods of (\w+) but does not declare it`)
)

func importCandidatesFor(path, alias string) []module.ImportCandidate {
	root := module.FindProjectRoot(filepath.Dir(path))
	imported := importedPaths(path)
	cyclic := dependentFiles(path)
	candidates := module.ImportablePaths(root)
	byPath := pathIndex(candidates)

	var found []module.ImportCandidate
	for _, c := range candidates {
		if moduleAlias(c.Path) != alias || c.File == path || imported[c.Path] || cyclic[c.File] || !module.CanImport(path, c.File) || exportsNothing(path, c, byPath) {
			continue
		}
		found = append(found, c)
	}
	return found
}

func missingAlias(message string) (string, bool) {
	m := undeclaredPattern.FindStringSubmatch(message)
	if m == nil {
		m = unknownTypeModule.FindStringSubmatch(message)
	}
	if m == nil {
		return "", false
	}
	return m[1], true
}

func importActions(uri protocol.DocumentUri, d protocol.Diagnostic) []protocol.CodeAction {
	alias, ok := missingAlias(d.Message)
	if !ok {
		return nil
	}

	path := uriToPath(uri)
	var actions []protocol.CodeAction
	for _, c := range importCandidatesFor(path, alias) {
		edit, ok := importEdit(path, c.Path)
		if !ok {
			continue
		}
		kind := protocol.CodeActionKindQuickFix
		actions = append(actions, protocol.CodeAction{
			Title:       fmt.Sprintf("Add import %q", c.Path),
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{d},
			Edit:        &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}}},
		})
	}
	return actions
}

func wantsKind(only []protocol.CodeActionKind, kind protocol.CodeActionKind) bool {
	if len(only) == 0 {
		return true
	}
	for _, o := range only {
		if kind == o || strings.HasPrefix(string(kind), string(o)+".") {
			return true
		}
	}
	return false
}

func codeActions(uri protocol.DocumentUri, diagnostics []protocol.Diagnostic, only []protocol.CodeActionKind) []protocol.CodeAction {
	var actions []protocol.CodeAction
	if wantsKind(only, protocol.CodeActionKindSourceOrganizeImports) {
		if action := organizeImportsAction(uri); action != nil {
			actions = append(actions, *action)
		}
	}
	if !wantsKind(only, protocol.CodeActionKindQuickFix) {
		return actions
	}
	for _, d := range diagnostics {
		actions = append(actions, implementsActions(uri, d)...)
		actions = append(actions, importActions(uri, d)...)
		if action := removeImportAction(uri, d); action != nil {
			actions = append(actions, *action)
		}

		m := suggestionPattern.FindStringSubmatch(d.Message)
		if m == nil {
			continue
		}
		kind := protocol.CodeActionKindQuickFix
		diag := d
		actions = append(actions, protocol.CodeAction{
			Title:       "Replace with " + m[1],
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			Edit: &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{
				uri: {{Range: d.Range, NewText: m[1]}},
			}},
		})
	}
	return actions
}

func implementsActions(uri protocol.DocumentUri, d protocol.Diagnostic) []protocol.CodeAction {
	m := missingImplements.FindStringSubmatch(d.Message)
	if m == nil {
		return nil
	}
	typeName, interfaceName := m[1], m[2]
	path := uriToPath(uri)
	src, ok := fileBuffer[path]
	if !ok {
		return nil
	}

	_, info := analyseEntry(path)
	if info == nil || info.Module == nil {
		return nil
	}
	if symbol, found := info.Module.Resolve(interfaceName); !found || symbol.Kind != analyser.INTERFACE {
		return nil
	}

	edit, ok := implementsEdit(src, typeName, interfaceName)
	if !ok {
		return nil
	}
	kind := protocol.CodeActionKindQuickFix
	return []protocol.CodeAction{{
		Title:       fmt.Sprintf("Declare that %s implements %s", typeName, interfaceName),
		Kind:        &kind,
		Diagnostics: []protocol.Diagnostic{d},
		Edit:        &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}}},
	}}
}

func implementsEdit(src []byte, typeName, interfaceName string) (protocol.TextEdit, bool) {
	program := parser.NewParser(lexer.NewLexer(src)).ParseProgram()
	lines := splitLines(src)

	for _, decl := range program.Declarations {
		var implements []ast.TypeRef
		var underlying *ast.TypeRef
		switch d := decl.(type) {
		case *ast.StructDecl:
			if d.Name != typeName {
				continue
			}
			implements = d.Implements
		case *ast.TypeDecl:
			if d.Name != typeName {
				continue
			}
			implements = d.Implements
			if !d.IsEnum {
				underlying = &d.Underlying
			}
		default:
			continue
		}

		if len(implements) > 0 {
			last := implements[len(implements)-1]
			at := protocol.Position{Line: uint32(last.EndLine - 1), Character: uint32(last.EndCol)}
			return protocol.TextEdit{Range: protocol.Range{Start: at, End: at}, NewText: ", " + interfaceName}, true
		}
		if underlying != nil {
			at := protocol.Position{Line: uint32(underlying.EndLine - 1), Character: uint32(underlying.EndCol)}
			return protocol.TextEdit{Range: protocol.Range{Start: at, End: at}, NewText: " implements " + interfaceName}, true
		}

		startLine, _ := decl.Position()
		for l := startLine - 1; l < len(lines); l++ {
			index := strings.Index(lines[l], "{")
			if index < 0 {
				continue
			}
			text := "implements " + interfaceName + " "
			if index > 0 && lines[l][index-1] != ' ' && lines[l][index-1] != '\t' {
				text = " " + text
			}
			at := protocol.Position{Line: uint32(l), Character: uint32(columnOf(lines[l], index))}
			return protocol.TextEdit{Range: protocol.Range{Start: at, End: at}, NewText: text}, true
		}
	}
	return protocol.TextEdit{}, false
}
