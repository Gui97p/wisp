package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/Gui97p/wisp/internal/module"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	suggestionPattern = regexp.MustCompile(`did you mean (\S+?)\?`)
	undeclaredPattern = regexp.MustCompile(`identifier (\w+) not declared in this scope`)
	unknownTypeModule = regexp.MustCompile(`unknown type (\w+)\.\w+`)
)

func importActions(uri protocol.DocumentUri, d protocol.Diagnostic) []protocol.CodeAction {
	m := undeclaredPattern.FindStringSubmatch(d.Message)
	if m == nil {
		m = unknownTypeModule.FindStringSubmatch(d.Message)
	}
	if m == nil {
		return nil
	}

	path := uriToPath(uri)
	root := module.FindProjectRoot(filepath.Dir(path))
	imported := importedPaths(path)
	cyclic := dependentFiles(path)

	var actions []protocol.CodeAction
	candidates := module.ImportablePaths(root)
	byPath := pathIndex(candidates)
	for _, c := range candidates {
		if moduleAlias(c.Path) != m[1] || c.File == path || imported[c.Path] || cyclic[c.File] || !module.CanImport(path, c.File) || exportsNothing(path, c, byPath) {
			continue
		}
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

func codeActions(uri protocol.DocumentUri, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
	var actions []protocol.CodeAction
	for _, d := range diagnostics {
		actions = append(actions, importActions(uri, d)...)

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
