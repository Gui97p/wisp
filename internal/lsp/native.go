package lsp

import (
	"regexp"
	"slices"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var nativeHeader = regexp.MustCompile(`\bnative\s*(\w*)\s*\(([^)]*)\)\s*$`)

func nativeItems(path string, pos protocol.Position, info *analyser.Info) ([]protocol.CompletionItem, bool) {
	text := textBefore(path, int(pos.Line), int(pos.Character))
	start, open := openStringStart(text)
	if !open {
		return nil, false
	}

	header := nativeHeader.FindStringSubmatch(text[:start])
	if header == nil {
		return nil, false
	}
	if header[1] == "lua" {
		return nil, false
	}

	items := []protocol.CompletionItem{}
	content := text[start+1:]
	brace := strings.LastIndex(content, "{")
	if brace < 0 || strings.Contains(content[brace:], "}") || info == nil {
		return items, true
	}

	scope := scopeAt(info, int(pos.Line)+1, int(pos.Character)+1)
	bound := nativeBoundNames(header[2])
	parts := strings.Split(strings.TrimPrefix(content[brace+1:], "&"), ".")

	if len(parts) == 1 {
		for _, name := range bound {
			detail := ""
			if sym, ok := scope.Resolve(name); ok && sym.Type != nil {
				detail = sym.Type.String()
			}
			items = append(items, item(name, protocol.CompletionItemKindVariable, detail))
		}
		return items, true
	}

	if !slices.Contains(bound, parts[0]) {
		return items, true
	}
	sym, ok := scope.Resolve(parts[0])
	if !ok {
		return items, true
	}

	tp := sym.Type
	for _, part := range parts[1 : len(parts)-1] {
		tp = nativeMemberType(tp, part)
		if tp == nil {
			return items, true
		}
	}

	if st := nativeStructOf(tp); st != nil {
		for _, field := range st.Order {
			items = append(items, item(field, protocol.CompletionItemKindField, st.Fields[field].String()))
		}
	} else if nativeWordPair(tp) {
		items = append(items,
			item("ptr", protocol.CompletionItemKindField, "int"),
			item("len", protocol.CompletionItemKindField, "int"))
	}
	return items, true
}

func nativeBoundNames(list string) []string {
	var names []string
	for _, entry := range strings.Split(list, ",") {
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			continue
		}
		name := fields[len(fields)-1]
		if name == "in" || name == "out" || name == "inout" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func nativeMemberType(tp analyser.Type, part string) analyser.Type {
	if st := nativeStructOf(tp); st != nil {
		return st.Fields[part]
	}
	if nativeWordPair(tp) && (part == "ptr" || part == "len") {
		return analyser.PrimitiveType{Name: "int"}
	}
	return nil
}

func nativeStructOf(tp analyser.Type) *analyser.StructType {
	switch v := tp.(type) {
	case *analyser.StructType:
		return v
	case analyser.NamedType:
		return nativeStructOf(v.Underlying)
	}
	return nil
}

func nativeWordPair(tp analyser.Type) bool {
	switch v := tp.(type) {
	case analyser.PrimitiveType:
		return v.Name == "string"
	case analyser.SpanType:
		return true
	case analyser.NamedType:
		return nativeWordPair(v.Underlying)
	}
	return false
}
