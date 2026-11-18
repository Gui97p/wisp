package lsp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/parser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var tagToken = regexp.MustCompile(`#[A-Za-z_]\w*`)

func atFileTop(path string, line int) bool {
	lines, ok := bufferLines(path)
	if !ok {
		return false
	}
	for i := 0; i < line && i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return false
	}
	return true
}

func tagItems(path string, pos protocol.Position, rng protocol.Range) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindKeyword
	items := []protocol.CompletionItem{}
	if atFileTop(path, int(pos.Line)) {
		for _, d := range parser.Directives {
			detail := "directive"
			doc := protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: d.Doc}
			items = append(items, protocol.CompletionItem{Label: d.Name, Kind: &kind, Detail: &detail, Documentation: doc, TextEdit: protocol.TextEdit{Range: rng, NewText: d.Name + " "}})
		}
	}
	for _, spec := range analyser.Tags {
		detail := "tag: " + spec.Usage
		doc := protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: tagDoc(spec)}
		items = append(items, protocol.CompletionItem{Label: spec.Name, Kind: &kind, Detail: &detail, Documentation: doc, TextEdit: protocol.TextEdit{Range: rng, NewText: spec.Name}})
	}
	return items
}

func tagDoc(spec analyser.TagSpec) string {
	return spec.Doc + "\n\nApplies to: " + strings.Join(spec.Targets, ", ") + "."
}

func tagHover(path string, pos protocol.Position) *protocol.Hover {
	lines, ok := bufferLines(path)
	if !ok || int(pos.Line) >= len(lines) {
		return nil
	}
	text := lines[pos.Line]
	if !strings.HasPrefix(strings.TrimSpace(text), "#") {
		return nil
	}
	for _, loc := range tagToken.FindAllStringIndex(text, -1) {
		start := utf16Len(text[:loc[0]])
		end := utf16Len(text[:loc[1]])
		if int(pos.Character) < start || int(pos.Character) > end {
			continue
		}
		name := text[loc[0]+1 : loc[1]]
		rng := &protocol.Range{
			Start: protocol.Position{Line: pos.Line, Character: protocol.UInteger(start)},
			End:   protocol.Position{Line: pos.Line, Character: protocol.UInteger(end)},
		}
		for _, spec := range analyser.Tags {
			if spec.Name == name {
				return markdownHover(renderDoc(spec.Usage, tagDoc(spec)), rng)
			}
		}
		for _, d := range parser.Directives {
			if d.Name == name {
				return markdownHover(renderDoc("#"+d.Name, d.Doc), rng)
			}
		}
		return markdownHover(renderDoc("#"+name, fmt.Sprintf("Not a known tag. The known tags are: %s.", knownTags())), rng)
	}
	return nil
}

func knownTags() string {
	names := make([]string, len(analyser.Tags))
	for i, spec := range analyser.Tags {
		names[i] = "#" + spec.Name
	}
	return strings.Join(names, ", ")
}
