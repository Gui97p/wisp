package lsp

import (
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	leadingDot     = regexp.MustCompile(`(^|[^\w)\]"'.])\.\w*$`)
	declaredBefore = regexp.MustCompile(`([A-Za-z_][\w.]*)\s+[A-Za-z_]\w*\s*=\s*$`)
	comparedBefore = regexp.MustCompile(`([A-Za-z_][\w.]*)\s*(==|!=)\s*$`)
	assignedBefore = regexp.MustCompile(`(^|[^=!<>])\b([A-Za-z_][\w.]*)\s*=\s*$`)
	caseBefore     = regexp.MustCompile(`\bcase\b[^:]*$`)
	armBefore      = regexp.MustCompile(`=>\s*$`)
	switchWord     = regexp.MustCompile(`\bswitch\b`)
	returnBefore   = regexp.MustCompile(`\breturn\b[^;]*$`)
	coalesceBefore = regexp.MustCompile(`([A-Za-z_][\w.]*\s*\((?:[^()]|\([^()]*\))*\))\s*\?\?\s*(?:\|\w+\|\s*)?$`)
	functionHeader = regexp.MustCompile(`^\s*(?:(?:public|internal|shared)\s+)*func\b`)
)

func implicitDotItems(path string, pos protocol.Position, info *analyser.Info) ([]protocol.CompletionItem, bool) {
	prefix := linePrefix(path, int(pos.Line), int(pos.Character))
	loc := leadingDot.FindStringSubmatchIndex(prefix)
	if loc == nil {
		return nil, false
	}

	items := []protocol.CompletionItem{}
	if info == nil {
		return items, true
	}

	before := prefix[:loc[3]]
	scope := scopeAt(info, int(pos.Line)+1, int(pos.Character)+1)
	enum, ok := expectedEnum(before, textBefore(path, int(pos.Line), int(pos.Character)), scope, path, int(pos.Line))
	if !ok {
		return items, true
	}

	for _, member := range enum.MemberOrder {
		items = append(items, item(member, protocol.CompletionItemKindEnumMember, enum.Name))
	}
	return items, true
}

func expectedEnum(before, text string, scope *analyser.Scope, path string, line int) (analyser.NamedType, bool) {
	var tp analyser.Type

	switch {
	case armBefore.MatchString(before):
		if _, prefix, ok := switchHeader(text); ok {
			tp = contextType(prefix, text, scope, path, line)
		}
	case caseBefore.MatchString(before):
		if subject, _, ok := switchHeader(text); ok && subject != "" {
			tp = exprType(scope, subject)
		} else if ok {
			tp = contextType(before, text, scope, path, line)
		}
	default:
		tp = contextType(before, text, scope, path, line)
	}

	enum, ok := tp.(analyser.NamedType)
	return enum, ok && enum.Enum
}

func contextType(before, text string, scope *analyser.Scope, path string, line int) analyser.Type {
	if m := coalesceBefore.FindStringSubmatch(before); m != nil {
		return exprType(scope, m[1])
	}
	if returnBefore.MatchString(before) {
		return returnType(path, line, before, scope)
	}
	if enum, ok := structValueEnum(text, scope); ok {
		return enum
	}
	if callee, arg, ok := callContext(maskLiterals(before)); ok {
		if ft := calleeType(scope, callee); ft != nil && arg < len(ft.Params) {
			return ft.Params[arg]
		}
	}
	if m := declaredBefore.FindStringSubmatch(before); m != nil {
		return typeFromText(scope, m[1])
	}
	if m := comparedBefore.FindStringSubmatch(before); m != nil {
		return exprType(scope, m[1])
	}
	if m := assignedBefore.FindStringSubmatch(before); m != nil {
		return exprType(scope, m[2])
	}
	return nil
}

func typeFromText(scope *analyser.Scope, text string) analyser.Type {
	sym, ok := resolveChain(scope, text)
	if !ok || (sym.Kind != analyser.TYPE && sym.Kind != analyser.STRUCT) {
		return nil
	}
	return sym.Type
}

func exprType(scope *analyser.Scope, text string) analyser.Type {
	text = strings.TrimSpace(text)
	if open := strings.Index(text, "("); open > 0 && strings.HasSuffix(text, ")") {
		if ft := calleeType(scope, text[:open]); ft != nil && len(ft.Returns) > 0 {
			return ft.Returns[0]
		}
		return nil
	}
	sym, ok := resolveChain(scope, text)
	if !ok || sym.Kind == analyser.TYPE || sym.Kind == analyser.STRUCT || sym.Kind == analyser.MODULE {
		return nil
	}
	return sym.Type
}

func calleeType(scope *analyser.Scope, callee string) *analyser.FuncType {
	if base, member, dotted := cutLast(callee, "."); dotted {
		sym, ok := resolveChain(scope, base)
		if !ok {
			return nil
		}
		if module, isModule := sym.Type.(*analyser.ModuleType); isModule {
			if export, found := module.Exports[member]; found {
				ft, _ := export.Type.(*analyser.FuncType)
				return ft
			}
			return nil
		}
		return analyser.MethodsOf(sym.Type)[member]
	}
	sym, ok := scope.Resolve(callee)
	if !ok {
		return nil
	}
	ft, _ := sym.Type.(*analyser.FuncType)
	return ft
}

func cutLast(s, sep string) (string, string, bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

func returnType(path string, line int, before string, scope *analyser.Scope) analyser.Type {
	lines, ok := bufferLines(path)
	if !ok {
		return nil
	}
	for i := line; i >= 0 && i < len(lines); i-- {
		if !functionHeader.MatchString(lines[i]) {
			continue
		}
		types := functionReturnTypes(lines[i])
		index := returnIndex(before)
		if index >= len(types) {
			return nil
		}
		return typeFromText(scope, types[index])
	}
	return nil
}

func returnIndex(before string) int {
	loc := returnBefore.FindStringIndex(before)
	if loc == nil {
		return 0
	}
	text := maskLiterals(before[loc[0]+len("return"):])
	depth, index := 0, 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				index++
			}
		}
	}
	return index
}

func functionReturnTypes(header string) []string {
	rest := strings.TrimSpace(header)
	for _, word := range []string{"public", "internal", "shared"} {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, word))
	}
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "func"))
	if strings.HasPrefix(rest, "(") {
		rest = skipGroup(rest)
	}
	rest = strings.TrimSpace(rest)
	open := strings.Index(rest, "(")
	if open < 0 {
		return nil
	}
	rest = strings.TrimSpace(skipGroup(rest[open:]))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "!"))
	if strings.HasPrefix(rest, "(") {
		end := len(rest) - len(skipGroup(rest))
		return splitTopLevel(rest[1 : end-1])
	}
	if cut := strings.IndexAny(rest, "{="); cut >= 0 {
		rest = rest[:cut]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil
	}
	return []string{rest}
}

func skipGroup(text string) string {
	depth := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[i+1:]
			}
		}
	}
	return ""
}

func splitTopLevel(text string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}
	return append(parts, strings.TrimSpace(text[start:]))
}

func switchHeader(text string) (subject, prefix string, ok bool) {
	type block struct {
		subject  string
		prefix   string
		isSwitch bool
	}
	masked := maskLiterals(text)
	var stack []block
	headerStart := 0

	for i := 0; i < len(masked); i++ {
		switch masked[i] {
		case '{':
			header := masked[headerStart:i]
			b := block{}
			if loc := switchWord.FindStringIndex(header); loc != nil {
				subject := strings.TrimSpace(text[headerStart+loc[1] : i])
				if strings.HasPrefix(subject, "let") {
					if eq := strings.Index(subject, "="); eq >= 0 {
						subject = subject[eq+1:]
					}
				}
				b.subject, b.prefix, b.isSwitch = strings.TrimSpace(subject), text[headerStart:headerStart+loc[0]], true
			}
			stack = append(stack, b)
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

	if len(stack) == 0 || !stack[len(stack)-1].isSwitch {
		return "", "", false
	}
	top := stack[len(stack)-1]
	return top.subject, top.prefix, true
}
