package lsp

import (
	"bytes"
	"github.com/Gui97p/wisp/internal/parser"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var directiveTyping = regexp.MustCompile(`^\s*#[A-Za-z_]*$`)
var allowTyping = regexp.MustCompile(`^\s*#allow\s+([A-Za-z_,\s]*)$`)

var selectorTyping = regexp.MustCompile(`(\b(native|func|struct|type|enum|const|import)<[^<>;]*|^\s*#target\b.*)$`)

var keywords = []string{
	"let", "const", "func", "struct", "enum", "type", "interface", "implements", "if", "else", "switch", "case", "default",
	"for", "in", "loop", "until", "return", "break", "continue", "import", "public", "internal", "private", "extern", "as", "native", "var", "static",
	"true", "false", "null", "defer", "yield", "inline",
}

var primitiveNames = []string{
	"int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64",
	"float", "float32", "float64", "bool", "byte", "string",
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

var cTypeNames = map[string]bool{
	"c_char": true, "c_schar": true, "c_uchar": true, "c_short": true, "c_ushort": true,
	"c_int": true, "c_uint": true, "c_long": true, "c_ulong": true, "c_longlong": true, "c_ulonglong": true,
	"c_size": true, "c_ssize": true, "c_float": true, "c_double": true,
}

func completion(uri protocol.DocumentUri, pos protocol.Position, trigger string) []protocol.CompletionItem {
	items := completionItems(uri, pos, trigger)
	path := uriToPath(uri)
	if strings.HasPrefix(typedWord(linePrefix(path, int(pos.Line), int(pos.Character))), "c_") {
		return items
	}
	kept := items[:0:0]
	for _, it := range items {
		if !cTypeNames[it.Label] {
			kept = append(kept, it)
		}
	}
	return kept
}

func completionItems(uri protocol.DocumentUri, pos protocol.Position, trigger string) []protocol.CompletionItem {
	path := uriToPath(uri)
	line := int(pos.Line) + 1
	col := int(pos.Character) + 1

	if items, ok := importItems(path, pos); ok {
		return items
	}

	if trigger == "[" || trigger == "=" || trigger == "\"" || trigger == "/" {
		return nil
	}

	if items, ok := labelItems(path, pos); ok {
		return items
	}

	candidates := infoCandidates(path, pos)
	var info *analyser.Info
	if len(candidates) > 0 {
		info = candidates[0]
	}

	if items, ok := nativeItems(path, pos, info); ok {
		return items
	}

	if brace, ok := formatBraceAt(path, pos, candidates); ok {
		return formatBraceItems(path, pos, candidates, brace, trigger)
	}

	if insideLiteralOrComment(textBefore(path, int(pos.Line), int(pos.Character))) {
		return nil
	}

	handled := false
	for _, candidate := range candidates {
		items, ok := implicitDotItems(path, pos, candidate)
		if !ok {
			break
		}
		handled = true
		if len(items) > 0 {
			return items
		}
	}
	if handled {
		return []protocol.CompletionItem{}
	}

	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	suffix := lineSuffix(path, int(pos.Line), int(pos.Character))

	if items, ok := anonLiteralItems(path, pos); ok {
		return items
	}
	if info != nil {
		if fields, ok := structLiteralItems(textBefore(path, int(pos.Line), int(pos.Character)), scopeFor(info, line, col)); ok {
			return fields
		}
	}
	if info != nil && (trigger == " " || trigger == ":") {
		text := textBefore(path, int(pos.Line), int(pos.Character))
		if tp, ok := structValueEnum(text, scopeFor(info, line, col)); ok {
			return enumValueItems(tp)
		}
	}
	if m := allowTyping.FindStringSubmatch(lastLine(textBefore(path, int(pos.Line), int(pos.Character)))); m != nil {
		given := map[string]bool{}
		for _, name := range strings.Split(m[1], ",") {
			given[strings.TrimSpace(name)] = true
		}
		kind := protocol.CompletionItemKindEnumMember
		items := []protocol.CompletionItem{}
		for _, lint := range parser.Lints {
			if given[lint.Name] {
				continue
			}
			detail := "warning"
			doc := protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: lint.Doc}
			items = append(items, protocol.CompletionItem{Label: lint.Name, Kind: &kind, Detail: &detail, Documentation: doc})
		}
		return items
	}
	if line := lastLine(textBefore(path, int(pos.Line), int(pos.Character))); directiveTyping.MatchString(line) {
		start := uint32(utf16Len(line[:strings.Index(line, "#")+1]))
		rng := protocol.Range{Start: protocol.Position{Line: pos.Line, Character: start}, End: pos}
		return tagItems(path, pos, rng)
	}
	if selectorTyping.MatchString(lastLine(textBefore(path, int(pos.Line), int(pos.Character)))) {
		items := []protocol.CompletionItem{}
		for _, name := range target.Selectors() {
			items = append(items, item(name, protocol.CompletionItemKindEnumMember, "target"))
		}
		return items
	}
	if before := textBefore(path, int(pos.Line), int(pos.Character)); trigger == "<" || trigger == "," || trigger == " " {
		if ctx, ok := genericArgsContext(before, info, line, col); ok {
			if !ctx.typing {
				return []protocol.CompletionItem{}
			}
			return genericTypeItems(path, pos, info, line, col, ctx)
		}
	}
	if trigger == "{" || trigger == "," || trigger == " " || trigger == ":" || trigger == "<" {
		return nil
	}

	if ctx, ok := genericArgsContext(textBefore(path, int(pos.Line), int(pos.Character)), info, line, col); ok {
		if !ctx.typing {
			return []protocol.CompletionItem{}
		}
		return genericTypeItems(path, pos, info, line, col, ctx)
	}

	if ctx, ok := typeContextAt(textBefore(path, int(pos.Line), int(pos.Character))); ok {
		if !ctx.typing {
			return []protocol.CompletionItem{}
		}
		items := typeItems(info, line, col, ctx)
		if info != nil && ctx.module == "" {
			before := textBefore(path, int(pos.Line), int(pos.Character))
			items = append(items, unimportedTypeItems(path, scopeAt(info, line, col), typedWord(before))...)
		}
		return items
	}

	if endsInMemberDot(prefix) && len(candidates) > 0 {
		if items, found := typedMembers(path, candidates[0], pos, suffix); found {
			return items
		}
	}
	if base, ok := memberBase(prefix); ok {
		for _, candidate := range candidates {
			if items := sharedMemberItems(path, base, candidate, line, col, suffix); items != nil {
				return items
			}
		}
		return nil
	}

	items := []protocol.CompletionItem{}
	for _, k := range keywords {
		items = append(items, item(k, protocol.CompletionItemKindKeyword, ""))
	}
	for _, p := range primitiveNames {
		items = append(items, item(p, protocol.CompletionItemKindClass, "type"))
	}
	if before := textBefore(path, int(pos.Line), int(pos.Character)); typeCaseTyping.MatchString(lastLine(before)) && insideTypeSwitch(before) {
		items = append(items, typeGroupItems()...)
	}

	if info != nil {
		items = append(items, scopeItems(path, info, line, col, prefix, suffix)...)
	}
	return items
}

func scopeItems(path string, info *analyser.Info, line, col int, prefix, suffix string) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	scope := scopeAt(info, line, col)
	depths := scope.Depths()
	for _, sym := range scope.Visible() {
		if strings.HasPrefix(sym.Name, "__wisp_") {
			continue
		}
		it := symbolItem(sym, suffix)
		if (sym.Kind == analyser.VAR || sym.Kind == analyser.PARAM) && !sym.Global {
			it.SortText = ptr("0" + string(rune('0'+min(depths[sym], 9))) + sym.Name)
		} else if sym.Builtin && sym.Name != "Error" {
			it.SortText = ptr(builtinRank(sym.Name) + sym.Name)
		}
		items = append(items, it)
	}
	items = append(items, shortcutItems(path, scopeAt(info, line, col), typedWord(prefix), suffix)...)
	return items
}

type formatBrace struct {
	expr   string
	suffix string
	inSpec bool
}

func formatBraceAt(path string, pos protocol.Position, candidates []*analyser.Info) (formatBrace, bool) {
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	inner, inside := formatBraceExpr(prefix)
	if !inside {
		return formatBrace{}, false
	}
	line, col := int(pos.Line)+1, int(pos.Character)+1
	isFormat := false
	for _, info := range candidates {
		if info == nil {
			continue
		}
		for _, lit := range info.FormatLiterals {
			after := line > lit.Line || (line == lit.Line && col > lit.Col)
			before := line < lit.EndLine || (line == lit.EndLine && col <= lit.EndCol)
			if after && before {
				isFormat = true
			}
		}
	}
	if !isFormat {
		return formatBrace{}, false
	}
	source, _, hasSpec := analyser.SplitFormatSpec(inner)
	return formatBrace{
		expr:   source,
		suffix: lineSuffix(path, int(pos.Line), int(pos.Character)),
		inSpec: hasSpec,
	}, true
}

func formatBraceExpr(prefix string) (string, bool) {
	start := -1
	quote := byte(0)
	trimmed := strings.TrimLeft(prefix, " \t")
	if strings.HasPrefix(trimmed, "\\\\") {
		start = len(prefix) - len(trimmed) + 2
	} else {
		for i := 0; i < len(prefix); i++ {
			c := prefix[i]
			if quote == 0 {
				switch c {
				case '/':
					if i+1 < len(prefix) && prefix[i+1] == '/' {
						return "", false
					}
				case '"', '`':
					quote, start = c, i+1
				case '\'':
					for i++; i < len(prefix) && prefix[i] != '\''; i++ {
						if prefix[i] == '\\' {
							i++
						}
					}
				}
				continue
			}
			if c == '\\' && quote == '"' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
		}
		if quote == 0 {
			return "", false
		}
	}

	text := prefix[start:]
	open, depth := -1, 0
	for j := 0; j < len(text); j++ {
		switch c := text[j]; {
		case c == '\\':
			j++
		case open < 0 && c == '{':
			if j+1 < len(text) && text[j+1] == '{' {
				j++
				continue
			}
			open, depth = j+1, 0
		case open < 0 && c == '}':
			if j+1 < len(text) && text[j+1] == '}' {
				j++
			}
		case open >= 0 && c == '{':
			depth++
		case open >= 0 && c == '}':
			if depth == 0 {
				open = -1
			} else {
				depth--
			}
		}
	}
	if open < 0 {
		return "", false
	}
	return text[open:], true
}

func formatBraceItems(path string, pos protocol.Position, candidates []*analyser.Info, brace formatBrace, trigger string) []protocol.CompletionItem {
	if brace.inSpec || (trigger != "" && trigger != ".") {
		return []protocol.CompletionItem{}
	}
	line, col := int(pos.Line)+1, int(pos.Character)+1
	if endsInMemberDot(brace.expr) && len(candidates) > 0 {
		if items, found := typedMembers(path, candidates[0], pos, brace.suffix); found {
			return items
		}
	}
	if base, ok := memberBase(brace.expr); ok {
		for _, candidate := range candidates {
			if items := sharedMemberItems(path, base, candidate, line, col, brace.suffix); items != nil {
				return items
			}
		}
		return []protocol.CompletionItem{}
	}
	items := []protocol.CompletionItem{}
	for _, p := range primitiveNames {
		items = append(items, item(p, protocol.CompletionItemKindClass, "type"))
	}
	for _, k := range []string{"true", "false", "null"} {
		items = append(items, item(k, protocol.CompletionItemKindKeyword, ""))
	}
	if len(candidates) > 0 && candidates[0] != nil {
		items = append(items, scopeItems(path, candidates[0], line, col, brace.expr, brace.suffix)...)
	}
	return items
}

func infoCandidates(path string, pos protocol.Position) []*analyser.Info {
	program, fresh, clean := analyseEntryChecked(path)
	if clean && fresh != nil {
		candidates := []*analyser.Info{currentInfo(path)}
		if insideLiteralOrComment(textBefore(path, int(pos.Line), int(pos.Character))) {
			if info := patchedInfo(path, pos); info != nil {
				candidates = append([]*analyser.Info{info}, candidates...)
			}
		}
		return candidates
	}

	var candidates []*analyser.Info
	if info := patchedInfo(path, pos); info != nil {
		candidates = append(candidates, info)
	}
	if program != nil && fresh != nil {
		candidates = append(candidates, fresh)
	}
	if prev, ok := getLastGood(path); ok {
		refreshModules(prev.info, fresh)
		candidates = append(candidates, prev.info)
	}
	return candidates
}

func patchedInfo(path string, pos protocol.Position) *analyser.Info {
	text, ok := fileBuffer[path]
	if !ok {
		return nil
	}
	offset := cursorOffset(text, int(pos.Line), int(pos.Character))
	if offset <= 0 || text[offset-1] != '.' {
		return nil
	}

	semicolon := missingSemicolonOffset(text, offset)
	repairs := []int{-1}
	if semicolon >= 0 {
		repairs = append(repairs, semicolon)
	}
	fillers := []string{"_", "_;", "_ {}", "_ {};"}
	if startsWithCondition(text, offset) {
		fillers = []string{"_ {}", "_", "_;", "_ {};"}
	}
	for _, repair := range repairs {
		for _, filler := range fillers {
			patched := make([]byte, 0, len(text)+len(filler)+1)
			if repair >= 0 {
				patched = append(patched, text[:repair]...)
				patched = append(patched, ';')
				patched = append(patched, text[repair:offset]...)
			} else {
				patched = append(patched, text[:offset]...)
			}
			patched = append(patched, filler...)
			patched = append(patched, text[offset:]...)

			buffers := maps.Clone(fileBuffer)
			buffers[path] = patched
			pa, err := runProjectAnalysis(path, buffers)
			if err != nil || pa == nil {
				continue
			}
			if m := pa.moduleOf(path); m != nil && m.mod.ParseError == nil {
				return m.info
			}
		}
	}
	return nil
}

func startsWithCondition(text []byte, offset int) bool {
	lineStart := bytes.LastIndexByte(text[:offset], '\n') + 1
	line := bytes.TrimLeft(text[lineStart:offset], " \t")
	for _, word := range []string{"loop ", "if ", "for ", "switch ", "until ", "else if "} {
		if bytes.HasPrefix(line, []byte(word)) {
			return true
		}
	}
	return false
}

func missingSemicolonOffset(text []byte, offset int) int {
	lineStart := bytes.LastIndexByte(text[:offset], '\n') + 1
	end := lineStart
	for end > 0 {
		lineBegin := bytes.LastIndexByte(text[:end-1], '\n') + 1
		line := bytes.TrimRight(text[lineBegin:end-1], " \t\r")
		if len(bytes.TrimSpace(line)) == 0 {
			end = lineBegin
			continue
		}
		if bytes.Contains(line, []byte("//")) {
			return -1
		}
		switch line[len(line)-1] {
		case ';', '{', '(', ',', '[':
			return -1
		}
		return lineBegin + len(line)
	}
	return -1
}

func cursorOffset(text []byte, line, character int) int {
	offset := 0
	for l := 0; l < line; l++ {
		next := bytes.IndexByte(text[offset:], '\n')
		if next < 0 {
			return -1
		}
		offset += next + 1
	}
	for c := 0; c < character && offset < len(text) && text[offset] != '\n'; c++ {
		_, size := utf8.DecodeRune(text[offset:])
		offset += size
	}
	return offset
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

func endsInMemberDot(prefix string) bool {
	end := len(prefix)
	for end > 0 && isIdentByte(prefix[end-1]) {
		end--
	}
	return end >= 1 && prefix[end-1] == '.' && !(end >= 2 && prefix[end-2] == '.')
}

func memberBase(prefix string) (string, bool) {
	end := len(prefix)
	for end > 0 && isIdentByte(prefix[end-1]) {
		end--
	}
	if end == 0 || prefix[end-1] != '.' {
		return "", false
	}
	if end >= 2 && prefix[end-2] == '.' {
		return "", false
	}
	dot := end - 1
	start := dot
	for {
		seg := start
		for seg > 0 && prefix[seg-1] == '!' {
			seg--
		}
		for seg > 0 && (prefix[seg-1] == ']' || prefix[seg-1] == ')') {
			var open int
			if prefix[seg-1] == ']' {
				open = matchingBracket(prefix, seg-1)
			} else {
				open = matchingParen(prefix, seg-1)
			}
			if open < 0 {
				return "", false
			}
			seg = open
		}
		word := seg
		for word > 0 && isIdentByte(prefix[word-1]) {
			word--
		}
		if word == seg {
			break
		}
		start = word
		if word > 1 && prefix[word-1] == '.' && (isIdentByte(prefix[word-2]) || prefix[word-2] == ']' || prefix[word-2] == ')') {
			start = word - 1
			continue
		}
		break
	}
	if start == dot {
		return "", false
	}
	return prefix[start:dot], true
}

func matchingBracket(text string, close int) int {
	depth := 0
	for i := close; i >= 0; i-- {
		switch text[i] {
		case ']':
			depth++
		case '[':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func matchingParen(text string, close int) int {
	depth := 0
	for i := close; i >= 0; i-- {
		switch text[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitChain(chain string) []string {
	var parts []string
	depth, from := 0, 0
	for i := 0; i < len(chain); i++ {
		switch chain[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case '.':
			if depth == 0 {
				parts = append(parts, chain[from:i])
				from = i + 1
			}
		}
	}
	return append(parts, chain[from:])
}

func splitIndexes(part string) (string, int) {
	count := 0
	for strings.HasSuffix(part, "]") {
		open := matchingBracket(part, len(part)-1)
		if open < 0 {
			break
		}
		part = part[:open]
		count++
	}
	return part, count
}

func indexed(tp analyser.Type, count int) analyser.Type {
	for ; count > 0 && tp != nil; count-- {
		switch t := tp.(type) {
		case analyser.ArrayType:
			tp = t.Element
		case analyser.SpanType:
			tp = t.Element
		case analyser.PointerType:
			tp = t.Element
		default:
			return nil
		}
	}
	return tp
}

func splitBangs(part string) (string, int) {
	bangs := 0
	for strings.HasSuffix(part, "!") {
		part = part[:len(part)-1]
		bangs++
	}
	return part, bangs
}

func unwrapped(tp analyser.Type, bangs int) analyser.Type {
	if bangs == 0 || tp == nil {
		return tp
	}
	if element, ok := analyser.OptionalElement(tp); ok {
		return element
	}
	return tp
}

func splitCall(part string) (string, bool) {
	if !strings.HasSuffix(part, ")") {
		return part, false
	}
	open := matchingParen(part, len(part)-1)
	if open < 0 {
		return part, false
	}
	return part[:open], true
}

func callResult(ft *analyser.FuncType) analyser.Type {
	if ft == nil || len(ft.Returns) == 0 {
		return nil
	}
	return ft.Returns[0]
}

func resolveChain(scope *analyser.Scope, chain string) (*analyser.Symbol, bool) {
	parts := splitChain(chain)
	firstText, firstBangs := splitBangs(parts[0])
	first, count := splitIndexes(firstText)
	name, firstIsCall := splitCall(first)
	sym, ok := scope.Resolve(name)
	if !ok {
		return nil, false
	}
	if firstIsCall {
		ft, isFunc := sym.Type.(*analyser.FuncType)
		result := callResult(ft)
		if !isFunc || result == nil {
			return nil, false
		}
		sym = &analyser.Symbol{Name: name, Kind: analyser.VAR, Type: result}
	}
	if count > 0 {
		tp := indexed(sym.Type, count)
		if tp == nil {
			return nil, false
		}
		sym = &analyser.Symbol{Name: name, Kind: analyser.VAR, Type: tp}
	}
	if firstBangs > 0 {
		sym = &analyser.Symbol{Name: name, Kind: analyser.VAR, Type: unwrapped(sym.Type, firstBangs)}
	}
	for _, part := range parts[1:] {
		partText, bangs := splitBangs(part)
		member, count := splitIndexes(partText)
		field, isCall := splitCall(member)
		if module, isModule := sym.Type.(*analyser.ModuleType); isModule {
			next, found := module.Exports[field]
			if !found {
				return nil, false
			}
			sym = next
			if isCall {
				ft, isFunc := sym.Type.(*analyser.FuncType)
				result := callResult(ft)
				if !isFunc || result == nil {
					return nil, false
				}
				sym = &analyser.Symbol{Name: field, Kind: analyser.VAR, Type: result}
			}
			continue
		}
		if isCall {
			result := callResult(analyser.MethodsOf(sym.Type)[field])
			if result == nil {
				return nil, false
			}
			sym = &analyser.Symbol{Name: field, Kind: analyser.VAR, Type: unwrapped(indexed(result, count), bangs)}
			if sym.Type == nil {
				return nil, false
			}
			continue
		}
		tp := unwrapped(indexed(chainFieldType(sym.Type, field), count), bangs)
		if tp == nil {
			return nil, false
		}
		sym = &analyser.Symbol{Name: field, Kind: analyser.VAR, Type: tp}
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

func sharedMemberItems(path, base string, info *analyser.Info, line, col int, suffix string) []protocol.CompletionItem {
	scopes := tiedScopes(info, line, col)
	if len(scopes) < 2 || !sharedHead(scopes[0], base) {
		return memberItems(path, base, scopeAt(info, line, col), suffix)
	}
	var lists [][]protocol.CompletionItem
	for _, scope := range scopes {
		items := memberItems(path, base, scope, suffix)
		if items == nil {
			return nil
		}
		lists = append(lists, items)
	}
	return commonItems(lists)
}

func sharedHead(scope *analyser.Scope, base string) bool {
	head, _, _ := strings.Cut(base, ".")
	sym, ok := scope.Resolve(head)
	return ok && sym.Shared
}

func memberItems(path, base string, scope *analyser.Scope, suffix string) []protocol.CompletionItem {
	sym, ok := resolveChain(scope, base)
	if !ok {
		return nil
	}
	return symbolMembers(path, sym, suffix)
}

func typedMembers(path string, info *analyser.Info, pos protocol.Position, suffix string) ([]protocol.CompletionItem, bool) {
	if info == nil {
		return nil, false
	}
	line, col := int(pos.Line)+1, int(pos.Character)+1
	var lists [][]protocol.CompletionItem
	shared := false
	for expr := range info.Types {
		member, ok := expr.(*ast.MemberExpr)
		if !ok || member.Field != "_" || member.FieldAt.Line != line || member.FieldAt.Col != col {
			continue
		}
		tp, known := info.Types[member.Object]
		if call, isCall := member.Object.(*ast.CallExpr); isCall && !known {
			if returns := info.CallReturns[call]; len(returns) > 0 {
				tp, known = returns[0], true
			}
		}
		if !known {
			return nil, false
		}
		if _, bad := tp.(analyser.InvalidType); bad {
			return nil, false
		}
		sym := &analyser.Symbol{Kind: analyser.VAR, Type: tp}
		if mt, isModule := tp.(*analyser.ModuleType); isModule {
			sym.Type = mt
		}
		items := symbolMembers(path, sym, suffix)
		if items == nil {
			return nil, false
		}
		if id, isIdent := member.Object.(*ast.IdentLiteral); isIdent {
			if bound := info.Idents[id]; bound != nil && bound.Shared {
				shared = true
			}
		}
		lists = append(lists, items)
	}
	if len(lists) == 0 {
		return nil, false
	}
	if !shared {
		return lists[0], true
	}
	return commonItems(lists), true
}

func commonItems(lists [][]protocol.CompletionItem) []protocol.CompletionItem {
	common := lists[0]
	for _, items := range lists[1:] {
		have := map[string]bool{}
		for _, it := range items {
			have[it.Label] = true
		}
		kept := make([]protocol.CompletionItem, 0, len(common))
		for _, it := range common {
			if have[it.Label] {
				kept = append(kept, it)
			}
		}
		common = kept
	}
	return common
}

func symbolMembers(path string, sym *analyser.Symbol, suffix string) []protocol.CompletionItem {
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
	if st, ok := t.(*analyser.StructType); ok && st.Optional == nil {
		for _, f := range st.Order {
			if hiddenField(f) || !analyser.FieldAccessible(st, f, path) {
				continue
			}
			items = append(items, item(f, protocol.CompletionItemKindField, st.Fields[f].String()))
		}
	}
	if _, ok := t.(analyser.ErrorType); ok {
		for _, f := range []string{"domain", "code", "message", "file", "line"} {
			items = append(items, item(f, protocol.CompletionItemKindField, ""))
		}
	}

	methods := accessibleMethods(path, t)
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
	case analyser.INTERFACE:
		kind = protocol.CompletionItemKindInterface
	case analyser.MODULE:
		kind = protocol.CompletionItemKindModule
	}
	detail := ""
	if sym.Type != nil {
		detail = sym.Type.String()
	}
	if help, builtin := isBuiltin(sym); builtin {
		it := item(sym.Name, kind, help.signature)
		it.Documentation = protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: help.doc}
		return callable(it, len(help.params), suffix)
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

func rankOf(kind protocol.CompletionItemKind) string {
	switch kind {
	case protocol.CompletionItemKindVariable:
		return "1"
	case protocol.CompletionItemKindField, protocol.CompletionItemKindEnumMember:
		return "2"
	case protocol.CompletionItemKindMethod:
		return "3"
	case protocol.CompletionItemKindFunction:
		return "4"
	case protocol.CompletionItemKindConstant:
		return "5"
	case protocol.CompletionItemKindStruct, protocol.CompletionItemKindClass, protocol.CompletionItemKindEnum, protocol.CompletionItemKindInterface:
		return "6"
	case protocol.CompletionItemKindModule:
		return "7"
	case protocol.CompletionItemKindKeyword:
		return "8"
	}
	return "9"
}

func item(label string, kind protocol.CompletionItemKind, detail string) protocol.CompletionItem {
	it := protocol.CompletionItem{Label: label, Kind: &kind, SortText: ptr(rankOf(kind) + label)}
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
	if variant, dotted := strings.CutPrefix(chain, "."); dotted {
		tp := expectedEnum(lastLine(text[:start]), text[:start], scope)
		if pointer, isPointer := tp.(analyser.PointerType); isPointer {
			tp = pointer.Element
		}
		sum, isSum := tp.(*analyser.StructType)
		if variant == "" {
			if isSum && sum.Optional != nil {
				sum, isSum = sum.Optional.(*analyser.StructType)
			}
			if !isSum || sum.Sum != nil {
				return nil, false
			}
			return sum, true
		}
		if !isSum || sum.Sum == nil {
			return nil, false
		}
		return variantPayload(sum, variant)
	} else if module, name, qualified := strings.Cut(chain, "."); qualified {
		owner, ok := scope.Resolve(module)
		if !ok {
			return nil, false
		}
		if sum, isSum := owner.Type.(*analyser.StructType); isSum && sum.Sum != nil {
			return variantPayload(sum, name)
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

func variantPayload(sum *analyser.StructType, name string) (*analyser.StructType, bool) {
	variant, ok := sum.Sum.Find(name)
	if !ok || variant.Payload == nil {
		return nil, false
	}
	return variant.Payload, true
}

func structLiteralItems(text string, scope *analyser.Scope) ([]protocol.CompletionItem, bool) {
	ctx, ok := structLiteralContext(text, scope)
	if !ok || ctx.hasKey {
		return nil, false
	}

	items := []protocol.CompletionItem{}
	if ctx.st.Optional != nil {
		return items, true
	}
	for _, field := range ctx.st.Order {
		if ctx.used[field] || hiddenField(field) {
			continue
		}
		it := item(field, protocol.CompletionItemKindField, analyser.TypeText(ctx.st.Fields[field]))
		text := field + ": "
		it.InsertText = &text
		items = append(items, it)
	}
	return items, true
}

func enumValueItems(tp analyser.Type) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	name, members, _ := dotMembers(tp)
	for _, member := range members {
		it := item("."+member, protocol.CompletionItemKindEnumMember, name)
		text := "." + member
		it.InsertText = &text
		items = append(items, it)
	}
	return items
}

type literalContext struct {
	st     *analyser.StructType
	used   map[string]bool
	key    string
	hasKey bool
}

func structLiteralContext(text string, scope *analyser.Scope) (literalContext, bool) {
	brace, ok := innermostOpen(text)
	if !ok {
		return literalContext{}, false
	}
	st, ok := literalType(text, brace, scope)
	if !ok {
		return literalContext{}, false
	}

	used, key, hasKey := scanLiteralBody(text[brace+1:])
	return literalContext{st: st, used: used, key: key, hasKey: hasKey}, true
}

func scanLiteralBody(body string) (map[string]bool, string, bool) {
	used := map[string]bool{}
	depth := 0
	segment := strings.Builder{}
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

	if key, _, hasKey := strings.Cut(segment.String(), ":"); hasKey {
		return used, strings.TrimSpace(key), true
	}
	return used, "", false
}

func structValueEnum(text string, scope *analyser.Scope) (analyser.Type, bool) {
	ctx, ok := structLiteralContext(text, scope)
	if !ok || !ctx.hasKey {
		return nil, false
	}
	tp := ctx.st.Fields[ctx.key]
	_, _, isChoice := dotMembers(tp)
	return tp, isChoice
}

func lastLine(text string) string {
	if i := strings.LastIndex(text, "\n"); i >= 0 {
		return text[i+1:]
	}
	return text
}

func hiddenField(name string) bool {
	return strings.HasPrefix(name, "$")
}

func anonLiteralItems(path string, pos protocol.Position) ([]protocol.CompletionItem, bool) {
	text, ok := fileBuffer[path]
	if !ok {
		return nil, false
	}
	offset := cursorOffset(text, int(pos.Line), int(pos.Character))
	if offset <= 0 {
		return nil, false
	}
	before := string(text[:offset])
	brace, ok := innermostOpen(before)
	if !ok || brace == 0 || before[brace-1] != '.' {
		return nil, false
	}
	used, _, hasKey := scanLiteralBody(before[brace+1:])
	if hasKey {
		return nil, false
	}

	start := offset
	for start > brace+1 && isIdentByte(text[start-1]) {
		start--
	}
	patched := make([]byte, 0, len(text)+8)
	patched = append(patched, text[:start]...)
	patched = append(patched, "__: 0"...)
	patched = append(patched, text[offset:]...)

	buffers := maps.Clone(fileBuffer)
	buffers[path] = patched
	pa, err := runProjectAnalysis(path, buffers)
	if err != nil || pa == nil {
		return nil, false
	}
	m := pa.moduleOf(path)
	if m == nil || m.merged == nil {
		return nil, false
	}

	var literal *ast.StructLiteral
	walkProgram(m.merged, func(expr ast.Expression) {
		if lit, isLit := expr.(*ast.StructLiteral); isLit && slices.Contains(lit.Keys, "__") {
			literal = lit
		}
	})
	if literal == nil {
		return nil, false
	}
	st, ok := m.info.Types[literal].(*analyser.StructType)
	if !ok {
		return nil, false
	}
	items := []protocol.CompletionItem{}
	for _, field := range st.Order {
		if used[field] || hiddenField(field) {
			continue
		}
		it := item(field, protocol.CompletionItemKindField, analyser.TypeText(st.Fields[field]))
		text := field + ": "
		it.InsertText = &text
		items = append(items, it)
	}
	return items, true
}

func genericTypeItems(path string, pos protocol.Position, info *analyser.Info, line, col int, ctx typeContext) []protocol.CompletionItem {
	items := typeItems(info, line, col, ctx)
	if info != nil && ctx.module == "" {
		before := textBefore(path, int(pos.Line), int(pos.Character))
		items = append(items, unimportedTypeItems(path, scopeAt(info, line, col), typedWord(before))...)
	}
	return items
}

func builtinRank(name string) string {
	switch name {
	case "alignof", "offsetof", "staticAssert", "saturate", "truncate":
		return "8y"
	}
	return "7"
}
