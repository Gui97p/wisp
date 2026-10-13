package lsp

import (
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var keywords = []string{
	"let", "const", "func", "struct", "enum", "type", "if", "else", "switch", "case", "default",
	"for", "in", "loop", "until", "return", "break", "continue", "import", "export", "as", "native",
	"true", "false", "null",
}

var primitiveNames = []string{
	"int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64",
	"float32", "float64", "bool", "char", "string", "map",
}

type analysis struct {
	program *ast.Program
	info    *analyser.Info
}

var lastGood = map[string]analysis{}

func completion(uri protocol.DocumentUri, pos protocol.Position) []protocol.CompletionItem {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	program, info, clean := analyseEntryChecked(path)
	if program != nil && info != nil && clean {
		lastGood[path] = analysis{program, info}
	} else if prev, ok := lastGood[path]; ok {
		program, info = prev.program, prev.info
	}

	prefix := linePrefix(path, int(pos.Line), int(pos.Character))

	if base, ok := memberBase(prefix); ok {
		if info == nil {
			return nil
		}
		return memberItems(base, scopeAt(info, line, col))
	}

	items := []protocol.CompletionItem{}
	for _, k := range keywords {
		items = append(items, item(k, protocol.CompletionItemKindKeyword, ""))
	}
	for _, p := range primitiveNames {
		items = append(items, item(p, protocol.CompletionItemKindClass, "type"))
	}

	if info != nil {
		for _, sym := range scopeAt(info, line, col).Visible() {
			items = append(items, symbolItem(sym))
		}
	}
	return items
}

func linePrefix(path string, line, character int) string {
	src, ok := fileBuffer[path]
	if !ok {
		return ""
	}
	lines := strings.Split(string(src), "\n")
	if line >= len(lines) {
		return ""
	}
	runes := []rune(lines[line])
	if character > len(runes) {
		character = len(runes)
	}
	return string(runes[:character])
}

func memberBase(prefix string) (string, bool) {
	end := len(prefix)
	for end > 0 && isIdentByte(prefix[end-1]) {
		end--
	}
	if end == 0 || prefix[end-1] != '.' {
		return "", false
	}
	start := end - 1
	for start > 0 && isIdentByte(prefix[start-1]) {
		start--
	}
	if start == end-1 {
		return "", false
	}
	return prefix[start : end-1], true
}

func isIdentByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func scopeAt(info *analyser.Info, line, col int) *analyser.Scope {
	var best *analyser.Scope
	var bestSize int
	for node, scope := range info.Scopes {
		if !contains(node, line, col) {
			continue
		}
		sl, _ := node.Position()
		el, _ := node.EndPosition()
		size := el - sl
		if best == nil || size < bestSize {
			best, bestSize = scope, size
		}
	}
	if best == nil {
		return analyser.NewScope(nil)
	}
	return best
}

func memberItems(base string, scope *analyser.Scope) []protocol.CompletionItem {
	sym, ok := scope.Resolve(base)
	if !ok {
		return nil
	}

	var items []protocol.CompletionItem

	switch t := sym.Type.(type) {
	case *analyser.ModuleType:
		for _, ex := range t.Exports {
			items = append(items, symbolItem(ex))
		}
		return items
	}

	t := sym.Type
	if sym.Kind == analyser.STRUCT || sym.Kind == analyser.TYPE {
		if nt, ok := t.(analyser.NamedType); ok && nt.Enum {
			for _, m := range nt.MemberOrder {
				items = append(items, item(m, protocol.CompletionItemKindEnumMember, nt.Name))
			}
		}
		return items
	}

	if pt, ok := t.(analyser.PointerType); ok {
		t = pt.Element
	}
	if st, ok := t.(*analyser.StructType); ok {
		for _, f := range st.Order {
			items = append(items, item(f, protocol.CompletionItemKindField, st.Fields[f].String()))
		}
	}
	if _, ok := t.(analyser.ErrorType); ok {
		for _, f := range []string{"domain", "code", "message", "file", "line"} {
			items = append(items, item(f, protocol.CompletionItemKindField, ""))
		}
	}

	methods := analyser.MethodsOf(t)
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		items = append(items, item(name, protocol.CompletionItemKindMethod, methods[name].String()))
	}
	return items
}

func symbolItem(sym *analyser.Symbol) protocol.CompletionItem {
	kind := protocol.CompletionItemKindVariable
	switch sym.Kind {
	case analyser.CONST:
		kind = protocol.CompletionItemKindConstant
	case analyser.FUNC:
		kind = protocol.CompletionItemKindFunction
	case analyser.STRUCT:
		kind = protocol.CompletionItemKindStruct
	case analyser.TYPE:
		kind = protocol.CompletionItemKindClass
	case analyser.MODULE:
		kind = protocol.CompletionItemKindModule
	}
	detail := ""
	if sym.Type != nil {
		detail = sym.Type.String()
	}
	return item(sym.Name, kind, detail)
}

func item(label string, kind protocol.CompletionItemKind, detail string) protocol.CompletionItem {
	it := protocol.CompletionItem{Label: label, Kind: &kind}
	if detail != "" {
		it.Detail = &detail
	}
	return it
}
