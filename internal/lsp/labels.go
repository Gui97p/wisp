package lsp

import (
	"fmt"
	"regexp"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type labelDef struct {
	name  string
	line  int
	col   int
	loop  bool
	value bool
}

var (
	labelHeader = regexp.MustCompile(`(?:^|=\s*|\(\s*|,\s*|\breturn\s+|\byield\s+):([A-Za-z_]\w*)\s*((?:for|loop|until)\b.*)?$`)
	jumpLabel   = regexp.MustCompile(`\b(break|continue|yield)\s*:(\w*)$`)
	jumpBare    = regexp.MustCompile(`\b(break|continue)\s+(\w*)$`)
	jumpColon   = regexp.MustCompile(`\b(break|continue|yield)\s*:\s*$`)
	wordAhead   = regexp.MustCompile(`^\w*`)
	wordBehind  = regexp.MustCompile(`\w*$`)
)

func labelsAt(path string, line, character int) []labelDef {
	text := textBefore(path, line, character)
	masked := maskLiterals(text)

	type frame struct{ label *labelDef }
	var stack []frame
	headerStart := 0
	row, col := 0, 0
	rows := make([]int, len(masked)+1)
	cols := make([]int, len(masked)+1)
	for i := 0; i < len(masked); i++ {
		rows[i], cols[i] = row, col
		if masked[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}

	for i := 0; i < len(masked); i++ {
		switch masked[i] {
		case '{':
			header := strings.TrimSpace(masked[headerStart:i])
			var label *labelDef
			if m := labelHeader.FindStringSubmatchIndex(header); m != nil {
				name := header[m[2]:m[3]]
				offset := strings.LastIndex(masked[headerStart:i], ":"+name)
				at := headerStart + offset
				label = &labelDef{name: name, line: rows[at], col: cols[at], loop: m[4] >= 0, value: !strings.HasPrefix(header, ":")}
			}
			stack = append(stack, frame{label})
			headerStart = i + 1
		case '}':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			headerStart = i + 1
		case ';':
			headerStart = i + 1
		}
	}

	var out []labelDef
	for _, f := range stack {
		if f.label != nil {
			out = append(out, *f.label)
		}
	}
	return out
}

func labelKind(l labelDef) string {
	switch {
	case l.loop:
		return "loop"
	case l.value:
		return "block that gives a value"
	}
	return "block"
}

func labelItems(path string, pos protocol.Position) ([]protocol.CompletionItem, bool) {
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	colon := true
	m := jumpLabel.FindStringSubmatch(prefix)
	if m == nil {
		m = jumpBare.FindStringSubmatch(prefix)
		colon = false
	}
	if m == nil || insideLiteralOrComment(textBefore(path, int(pos.Line), int(pos.Character))) {
		return nil, false
	}

	items := []protocol.CompletionItem{}
	labels := labelsAt(path, int(pos.Line), int(pos.Character))
	for i := len(labels) - 1; i >= 0; i-- {
		l := labels[i]
		if m[1] == "yield" && l.loop {
			continue
		}
		if m[1] != "yield" && !l.loop {
			continue
		}
		it := item(l.name, protocol.CompletionItemKindEnumMember, "label of a "+labelKind(l))
		it.SortText = ptr(fmt.Sprintf("%02d", len(labels)-i))
		if !colon {
			it.Label = ":" + l.name
			it.FilterText = ptr(l.name)
			it.InsertText = ptr(":" + l.name)
		}
		items = append(items, it)
	}
	return items, true
}

func labelRefAt(path string, pos protocol.Position) (string, protocol.Range, bool) {
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	suffix := lineSuffix(path, int(pos.Line), int(pos.Character))
	word := wordAhead.FindString(suffix)
	head := wordBehind.FindString(prefix)
	name := head + word
	if name == "" {
		return "", protocol.Range{}, false
	}
	if !jumpColon.MatchString(prefix[:len(prefix)-len(head)]) {
		return "", protocol.Range{}, false
	}
	start := protocol.UInteger(len([]rune(prefix)) - len([]rune(head)))
	return name, protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: start},
		End:   protocol.Position{Line: pos.Line, Character: start + protocol.UInteger(len([]rune(name)))},
	}, true
}

func labelHover(path string, pos protocol.Position) *protocol.Hover {
	name, rng, ok := labelRefAt(path, pos)
	if !ok {
		return nil
	}
	for _, l := range labelsAt(path, int(pos.Line), int(pos.Character)) {
		if l.name == name {
			doc := fmt.Sprintf("The label of a %s, declared on line %d.", labelKind(l), l.line+1)
			if l.value {
				doc += " `break :" + name + " value` leaves it with that value."
			}
			return markdownHover(renderDoc(":"+name, doc), &rng)
		}
	}
	return markdownHover(renderDoc(":"+name, "No loop or block around this has this label."), &rng)
}

func labelDefinition(path string, pos protocol.Position, uri protocol.DocumentUri) []protocol.Location {
	name, _, ok := labelRefAt(path, pos)
	if !ok {
		return nil
	}
	labels := labelsAt(path, int(pos.Line), int(pos.Character))
	for i := len(labels) - 1; i >= 0; i-- {
		l := labels[i]
		if l.name == name {
			at := protocol.Position{Line: protocol.UInteger(l.line), Character: protocol.UInteger(l.col + 1)}
			return []protocol.Location{{URI: uri, Range: protocol.Range{Start: at, End: protocol.Position{Line: at.Line, Character: at.Character + protocol.UInteger(len(l.name))}}}}
		}
	}
	return nil
}
