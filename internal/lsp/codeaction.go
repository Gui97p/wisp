package lsp

import (
	"regexp"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var suggestionPattern = regexp.MustCompile(`did you mean (\S+?)\?`)

func codeActions(uri protocol.DocumentUri, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
	var actions []protocol.CodeAction
	for _, d := range diagnostics {
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
