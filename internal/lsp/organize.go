package lsp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var unusedImportPattern = regexp.MustCompile(`^import (\S+) is never used`)

func importLineRange(lines []string, d protocol.Diagnostic) (protocol.Range, bool) {
	start, end := int(d.Range.Start.Line), int(d.Range.End.Line)
	if start < 0 || end >= len(lines) || start > end || !strings.HasPrefix(strings.TrimSpace(lines[start]), "import ") {
		return protocol.Range{}, false
	}
	return protocol.Range{
		Start: protocol.Position{Line: uint32(start), Character: 0},
		End:   protocol.Position{Line: uint32(end + 1), Character: 0},
	}, true
}

func removeImportAction(uri protocol.DocumentUri, d protocol.Diagnostic) *protocol.CodeAction {
	m := unusedImportPattern.FindStringSubmatch(diagnosticText(d))
	if m == nil {
		return nil
	}
	lines, ok := bufferLines(uriToPath(uri))
	if !ok {
		return nil
	}
	rng, ok := importLineRange(lines, d)
	if !ok {
		return nil
	}
	kind := protocol.CodeActionKindQuickFix
	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Remove unused import %s", m[1]),
		Kind:        &kind,
		Diagnostics: []protocol.Diagnostic{d},
		Edit:        &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {{Range: rng}}}},
	}
}

func organizeImportsAction(uri protocol.DocumentUri) *protocol.CodeAction {
	path := uriToPath(uri)
	lines, ok := bufferLines(path)
	if !ok {
		return nil
	}

	pa, err := analyseProject(path)
	if err != nil || pa == nil {
		return nil
	}
	m := pa.moduleOf(path)
	if m == nil || m.mod.ParseError != nil {
		return nil
	}

	var edits []protocol.TextEdit
	removed := map[uint32]bool{}
	var missing []string
	seen := map[string]bool{}
	for _, d := range diagnoseFile(path)[path] {
		if unusedImportPattern.MatchString(diagnosticText(d)) {
			if rng, ok := importLineRange(lines, d); ok && !removed[rng.Start.Line] {
				removed[rng.Start.Line] = true
				edits = append(edits, protocol.TextEdit{Range: rng})
			}
			continue
		}
		alias, isMissing := missingAlias(diagnosticText(d))
		if !isMissing {
			continue
		}
		candidates := importCandidatesFor(path, alias)
		if len(candidates) == 1 && !seen[candidates[0].Path] {
			seen[candidates[0].Path] = true
			missing = append(missing, candidates[0].Path)
		}
	}

	if len(missing) > 0 {
		sort.Slice(missing, func(a, b int) bool {
			sa, sb := strings.HasPrefix(missing[a], "std/"), strings.HasPrefix(missing[b], "std/")
			if sa != sb {
				return sa
			}
			return missing[a] < missing[b]
		})
		first, ok := importEdit(path, missing[0])
		if ok {
			var text strings.Builder
			for _, p := range missing {
				fmt.Fprintf(&text, "import %q;\n", p)
			}
			if strings.HasSuffix(first.NewText, "\n\n") {
				text.WriteString("\n")
			}
			first.NewText = text.String()
			edits = append(edits, first)
		}
	}

	if len(edits) == 0 {
		return nil
	}
	sort.Slice(edits, func(a, b int) bool { return edits[a].Range.Start.Line < edits[b].Range.Start.Line })
	kind := protocol.CodeActionKindSourceOrganizeImports
	return &protocol.CodeAction{
		Title: "Organize imports",
		Kind:  &kind,
		Edit:  &protocol.WorkspaceEdit{Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: edits}},
	}
}
