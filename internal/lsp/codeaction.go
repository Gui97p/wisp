package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/module"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	suggestionPattern = regexp.MustCompile(`did you mean (\S+?)\?`)
	undeclaredPattern = regexp.MustCompile(`identifier (\w+) not declared in this scope`)
	unknownTypeModule = regexp.MustCompile(`unknown type (\w+)\.\w+`)
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
