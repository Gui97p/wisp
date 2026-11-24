package lsp

import (
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	leadingDot     = regexp.MustCompile(`(^|[^\w)\]"'.!])\.\w*$`)
	declaredBefore = regexp.MustCompile(`([A-Za-z_][\w.]*)\s+[A-Za-z_]\w*\s*=\s*$`)
	comparedBefore = regexp.MustCompile(`([A-Za-z_][\w.]*)\s*(==|!=)\s*$`)
	assignedBefore = regexp.MustCompile(`(^|[^=!<>])\b([A-Za-z_][\w.]*)\s*=\s*$`)
	caseWord       = regexp.MustCompile(`\bcase\b`)
	armBefore      = regexp.MustCompile(`=>\s*$`)
	yieldBefore    = regexp.MustCompile(`\b(?:yield|break\s+:\w+)\s*$`)
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
	name, members, ok := dotMembers(expectedEnum(before, textBefore(path, int(pos.Line), int(pos.Character)), scope))
	if !ok {
		return items, true
	}

	for _, member := range members {
		items = append(items, item(member, protocol.CompletionItemKindEnumMember, name))
	}
	return items, true
}

func dotMembers(tp analyser.Type) (string, []string, bool) {
	if pointer, ok := tp.(analyser.PointerType); ok {
		tp = pointer.Element
	}
	if optional, ok := tp.(*analyser.StructType); ok && optional.Optional != nil {
		tp = optional.Optional
	}
	switch v := tp.(type) {
	case analyser.NamedType:
		if v.Enum {
			return v.Name, v.MemberOrder, true
		}
	case *analyser.StructType:
		if v.Sum != nil {
			names := make([]string, 0, len(v.Sum.Variants))
			for _, variant := range v.Sum.Variants {
				names = append(names, variant.Name)
			}
			return v.Name, names, true
		}
	}
	return "", nil, false
}

func expectedEnum(before, text string, scope *analyser.Scope) analyser.Type {
	var tp analyser.Type

	switch {
	case yieldBefore.MatchString(before):
		tp = yieldType(text, scope)
	case armBefore.MatchString(before):
		if _, prefix, ok := switchHeader(text); ok {
			tp = contextType(prefix, text, scope)
		}
	case inCaseHeader(before):
		if subject, _, ok := switchHeader(text); ok && subject != "" {
			tp = exprType(scope, subject)
		} else if ok {
			tp = contextType(before, text, scope)
		}
	default:
		tp = contextType(before, text, scope)
	}

	return tp
}

func yieldType(text string, scope *analyser.Scope) analyser.Type {
	masked := maskLiterals(text)
	var opens []int
	for i := 0; i < len(masked); i++ {
		switch masked[i] {
		case '{':
			opens = append(opens, i)
		case '}':
			if len(opens) > 0 {
				opens = opens[:len(opens)-1]
			}
		}
	}
	for i := len(opens) - 1; i >= 0; i-- {
		head := masked[:opens[i]]
		start := strings.LastIndexAny(head, ";{}\n")
		statement := head[start+1:]
		if functionHeader.MatchString(statement) {
			return nil
		}
		if tp := contextType(statement, text, scope); tp != nil {
			return tp
		}
	}
	return nil
}

func contextType(before, text string, scope *analyser.Scope) analyser.Type {
	if m := coalesceBefore.FindStringSubmatch(before); m != nil {
		return exprType(scope, m[1])
	}
	if returnBefore.MatchString(before) {
		return returnType(text, before, scope)
	}
	if tp, ok := structValueEnum(text, scope); ok {
		return tp
	}
	if callee, arg, ok := callContext(maskLiterals(before)); ok {
		if ft := calleeType(scope, callee); ft != nil && arg < len(ft.Params) {
			return ft.Params[arg]
		}
		if tp := genericParamType(calleeGeneric(scope, callee), arg); tp != nil {
			return tp
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
	text = strings.TrimPrefix(strings.TrimSpace(text), "?")
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

func calleeGeneric(scope *analyser.Scope, callee string) *analyser.Symbol {
	var sym *analyser.Symbol
	if base, member, dotted := cutLast(callee, "."); dotted {
		owner, ok := resolveChain(scope, base)
		if !ok {
			return nil
		}
		if module, isModule := owner.Type.(*analyser.ModuleType); isModule {
			sym = module.Exports[member]
		}
	} else if found, ok := scope.Resolve(callee); ok {
		sym = found
	}
	if sym == nil {
		return nil
	}
	if _, isGeneric := sym.Type.(analyser.GenericType); !isGeneric {
		return nil
	}
	return sym
}

func genericParamType(sym *analyser.Symbol, arg int) analyser.Type {
	if sym == nil || sym.Generic == nil || sym.Generic.Func == nil || sym.Generic.Scope == nil {
		return nil
	}
	params := sym.Generic.Func.Params
	if arg >= len(params) || params[arg].Variadic || params[arg].Any {
		return nil
	}
	return typeFromText(sym.Generic.Scope, refText(params[arg].Type))
}

func cutLast(s, sep string) (string, string, bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

func returnType(text, before string, scope *analyser.Scope) analyser.Type {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
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

func inCaseHeader(before string) bool {
	found := caseWord.FindAllStringIndex(before, -1)
	if len(found) == 0 {
		return false
	}
	return !strings.Contains(before[found[len(found)-1][0]:], "=>")
}
