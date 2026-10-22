package lsp

import (
	"sort"
	"strings"
	"sync"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var keywords = []string{
	"let", "const", "func", "struct", "enum", "type", "if", "else", "switch", "case", "default",
	"for", "in", "loop", "until", "return", "break", "continue", "import", "export", "extern", "as", "native",
	"true", "false", "null", "defer",
}

var primitiveNames = []string{
	"int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64",
	"float", "float32", "float64", "bool", "char", "string", "map",
}

type analysis struct {
	program *ast.Program
	info    *analyser.Info
}

var (
	lastGood   = map[string]analysis{}
	lastGoodMu sync.Mutex
)

func setLastGood(path string, a analysis) {
	lastGoodMu.Lock()
	defer lastGoodMu.Unlock()
	lastGood[path] = a
}

func getLastGood(path string) (analysis, bool) {
	lastGoodMu.Lock()
	defer lastGoodMu.Unlock()
	a, ok := lastGood[path]
	return a, ok
}

func completion(uri protocol.DocumentUri, pos protocol.Position, trigger string) []protocol.CompletionItem {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	if items, ok := importItems(path, pos); ok {
		return items
	}

	if trigger == "[" || trigger == "=" || trigger == "\"" || trigger == "/" {
		return nil
	}

	if items, ok := nativeItems(path, pos, currentInfo(path)); ok {
		return items
	}

	if insideLiteralOrComment(textBefore(path, int(pos.Line), int(pos.Character))) {
		return nil
	}

	if items, ok := implicitDotItems(path, pos, currentInfo(path)); ok {
		return items
	}

	info := currentInfo(path)

	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	suffix := lineSuffix(path, int(pos.Line), int(pos.Character))

	if info != nil {
		if fields, ok := structLiteralItems(textBefore(path, int(pos.Line), int(pos.Character)), scopeFor(info, line, col)); ok {
			return fields
		}
	}
	if trigger == "{" || trigger == "," {
		return nil
	}

	if ctx, ok := typeContextAt(textBefore(path, int(pos.Line), int(pos.Character))); ok {
		if !ctx.typing {
			return []protocol.CompletionItem{}
		}
		return typeItems(info, line, col, ctx)
	}

	if base, ok := memberBase(prefix); ok {
		if info == nil {
			return nil
		}
		return memberItems(base, scopeAt(info, line, col), suffix)
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
			items = append(items, symbolItem(sym, suffix))
		}
	}
	return items
}

func currentInfo(path string) *analyser.Info {
	program, info, clean := analyseEntryChecked(path)
	if program != nil && info != nil && clean {
		setLastGood(path, analysis{program, info})
		return info
	}
	if prev, ok := getLastGood(path); ok {
		refreshModules(prev.info, info)
		return prev.info
	}
	return info
}

func refreshModules(prev, fresh *analyser.Info) {
	if prev == nil || fresh == nil || prev.Module == nil || fresh.Module == nil {
		return
	}
	for _, sym := range fresh.Module.Visible() {
		if sym.Kind != analyser.MODULE {
			continue
		}
		if old, ok := prev.Module.Resolve(sym.Name); ok && old.Kind == analyser.MODULE {
			old.Type = sym.Type
			continue
		}
		prev.Module.Define(sym)
	}
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
	for start > 0 && (isIdentByte(prefix[start-1]) || (prefix[start-1] == '.' && start > 1 && isIdentByte(prefix[start-2]))) {
		start--
	}
	if start == end-1 {
		return "", false
	}
	return prefix[start : end-1], true
}

func resolveChain(scope *analyser.Scope, chain string) (*analyser.Symbol, bool) {
	parts := strings.Split(chain, ".")
	sym, ok := scope.Resolve(parts[0])
	if !ok {
		return nil, false
	}
	for _, part := range parts[1:] {
		if module, isModule := sym.Type.(*analyser.ModuleType); isModule {
			next, found := module.Exports[part]
			if !found {
				return nil, false
			}
			sym = next
			continue
		}
		tp := chainFieldType(sym.Type, part)
		if tp == nil {
			return nil, false
		}
		sym = &analyser.Symbol{Name: part, Kind: analyser.VAR, Type: tp}
	}
	return sym, true
}

func chainFieldType(tp analyser.Type, field string) analyser.Type {
	if pt, ok := tp.(analyser.PointerType); ok {
		tp = pt.Element
	}
	if st, ok := tp.(*analyser.StructType); ok {
		return st.Fields[field]
	}
	return nil
}

func isIdentByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func scopeAt(info *analyser.Info, line, col int) *analyser.Scope {
	if scope := innermostScope(info, line, col); scope != nil {
		return scope
	}
	if info.Module != nil {
		return info.Module
	}
	return analyser.NewScope(nil)
}

func memberItems(base string, scope *analyser.Scope, suffix string) []protocol.CompletionItem {
	sym, ok := resolveChain(scope, base)
	if !ok {
		return nil
	}

	var items []protocol.CompletionItem

	switch t := sym.Type.(type) {
	case *analyser.ModuleType:
		for _, ex := range t.Exports {
			items = append(items, symbolItem(ex, suffix))
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
		items = append(items, callable(item(name, protocol.CompletionItemKindMethod, analyser.FuncSignature(name, methods[name], nil)), len(methods[name].Params), suffix))
	}
	return items
}

func symbolItem(sym *analyser.Symbol, suffix string) protocol.CompletionItem {
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
	if ft, ok := sym.Type.(*analyser.FuncType); ok && sym.Kind == analyser.FUNC {
		return callable(item(sym.Name, kind, analyser.FuncSignature(sym.Name, ft, nil)), len(ft.Params), suffix)
	}
	it := item(sym.Name, kind, detail)
	return it
}

func callable(it protocol.CompletionItem, params int, suffix string) protocol.CompletionItem {
	if strings.HasPrefix(strings.TrimLeft(suffix, " \t"), "(") {
		return it
	}

	snippet := protocol.InsertTextFormatSnippet
	it.InsertTextFormat = &snippet
	if params == 0 {
		text := it.Label + "()"
		it.InsertText = &text
		return it
	}

	text := it.Label + "($0)"
	it.InsertText = &text
	it.Command = &protocol.Command{Title: "Parameter hints", Command: "editor.action.triggerParameterHints"}
	return it
}

func item(label string, kind protocol.CompletionItemKind, detail string) protocol.CompletionItem {
	it := protocol.CompletionItem{Label: label, Kind: &kind}
	if detail != "" {
		it.Detail = &detail
	}
	return it
}

func innermostOpen(text string) (int, bool) {
	index, char, ok := innermostBracket(text)
	if !ok || char != '{' {
		return 0, false
	}
	return index, true
}

func innermostBracket(text string) (int, byte, bool) {
	type open struct {
		char  byte
		index int
	}
	var stack []open

	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '"', '`', '\'':
			for i++; i < len(text) && text[i] != c; i++ {
				if text[i] == '\\' && c != '`' {
					i++
				}
			}
		case '/':
			if i+1 < len(text) && text[i+1] == '/' {
				for i < len(text) && text[i] != '\n' {
					i++
				}
			} else if i+1 < len(text) && text[i+1] == '*' {
				end := strings.Index(text[i+2:], "*/")
				if end < 0 {
					return 0, 0, false
				}
				i += end + 3
			}
		case '{', '(', '[':
			stack = append(stack, open{c, i})
		case '}', ')', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	if len(stack) == 0 {
		return 0, 0, false
	}
	top := stack[len(stack)-1]
	return top.index, top.char, true
}

func literalType(text string, brace int, scope *analyser.Scope) (*analyser.StructType, bool) {
	end := brace
	for end > 0 && (text[end-1] == ' ' || text[end-1] == '\t' || text[end-1] == '\n') {
		end--
	}

	start := end
	for start > 0 && (isIdentByte(text[start-1]) || text[start-1] == '.') {
		start--
	}
	chain := text[start:end]
	if chain == "" {
		return nil, false
	}

	before := start
	for before > 0 && (text[before-1] == ' ' || text[before-1] == '\t' || text[before-1] == '\n') {
		before--
	}
	if before > 0 && !strings.ContainsRune("=(,:[{;>", rune(text[before-1])) && !strings.HasSuffix(text[:before], "return") {
		return nil, false
	}

	var sym *analyser.Symbol
	var found bool
	if module, name, qualified := strings.Cut(chain, "."); qualified {
		owner, ok := scope.Resolve(module)
		if !ok {
			return nil, false
		}
		mt, isModule := owner.Type.(*analyser.ModuleType)
		if !isModule {
			return nil, false
		}
		sym, found = mt.Exports[name], mt.Exports[name] != nil
	} else {
		sym, found = scope.Resolve(chain)
	}
	if !found || sym.Kind != analyser.STRUCT {
		return nil, false
	}

	st, ok := sym.Type.(*analyser.StructType)
	return st, ok
}

func structLiteralItems(text string, scope *analyser.Scope) ([]protocol.CompletionItem, bool) {
	brace, ok := innermostOpen(text)
	if !ok {
		return nil, false
	}
	st, ok := literalType(text, brace, scope)
	if !ok {
		return nil, false
	}

	used := map[string]bool{}
	depth := 0
	segment := strings.Builder{}
	body := text[brace+1:]
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch c {
		case '"', '`', '\'':
			segment.WriteByte(c)
			for i++; i < len(body) && body[i] != c; i++ {
				segment.WriteByte(body[i])
				if body[i] == '\\' && c != '`' && i+1 < len(body) {
					i++
					segment.WriteByte(body[i])
				}
			}
			if i < len(body) {
				segment.WriteByte(c)
			}
			continue
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				if key, _, hasKey := strings.Cut(segment.String(), ":"); hasKey {
					used[strings.TrimSpace(key)] = true
				}
				segment.Reset()
				continue
			}
		}
		segment.WriteByte(c)
	}

	if strings.Contains(segment.String(), ":") {
		return nil, false
	}

	items := []protocol.CompletionItem{}
	for _, field := range st.Order {
		if used[field] {
			continue
		}
		it := item(field, protocol.CompletionItemKindField, analyser.TypeText(st.Fields[field]))
		text := field + ": "
		it.InsertText = &text
		items = append(items, it)
	}
	return items, true
}
