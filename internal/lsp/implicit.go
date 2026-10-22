package lsp

import (
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	leadingDot      = regexp.MustCompile(`(^|[^\w)\]"'.])\.\w*$`)
	declaredBefore  = regexp.MustCompile(`([A-Za-z_][\w.]*)\s+[A-Za-z_]\w*\s*=\s*$`)
	comparedBefore  = regexp.MustCompile(`([A-Za-z_][\w.]*)\s*(==|!=)\s*$`)
	assignedBefore  = regexp.MustCompile(`(^|[^=!<>])\b([A-Za-z_][\w.]*)\s*=\s*$`)
	caseBefore      = regexp.MustCompile(`\bcase\b[^:]*$`)
	returnBefore    = regexp.MustCompile(`\breturn\b[^;]*$`)
	functionReturns = regexp.MustCompile(`^\s*(?:export\s+)?func\b.*\)\s*!?\s*([A-Za-z_][\w.]*)\s*(?:\{|=>)`)
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
	case caseBefore.MatchString(before):
		if subject, ok := switchSubject(text); ok {
			tp = exprType(scope, subject)
		}
	case returnBefore.MatchString(before):
		tp = returnType(path, line, before, scope)
	default:
		if callee, arg, ok := callContext(before); ok {
			if ft := calleeType(scope, callee); ft != nil && arg < len(ft.Params) {
				tp = ft.Params[arg]
				break
			}
		}
		if m := declaredBefore.FindStringSubmatch(before); m != nil {
			tp = typeFromText(scope, m[1])
		} else if m := comparedBefore.FindStringSubmatch(before); m != nil {
			tp = exprType(scope, m[1])
		} else if m := assignedBefore.FindStringSubmatch(before); m != nil {
			tp = exprType(scope, m[2])
		}
	}

	enum, ok := tp.(analyser.NamedType)
	return enum, ok && enum.Enum
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
		m := functionReturns.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		return typeFromText(scope, m[1])
	}
	return nil
}

func switchSubject(text string) (string, bool) {
	type block struct {
		subject  string
		isSwitch bool
	}
	var stack []block
	headerStart := 0

	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '{':
			header := strings.TrimSpace(text[headerStart:i])
			b := block{}
			if strings.HasPrefix(header, "switch") {
				subject := strings.TrimSpace(strings.TrimPrefix(header, "switch"))
				if strings.HasPrefix(subject, "let") {
					if eq := strings.Index(subject, "="); eq >= 0 {
						subject = subject[eq+1:]
					}
				}
				b.subject, b.isSwitch = strings.TrimSpace(subject), true
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
		return "", false
	}
	return stack[len(stack)-1].subject, true
}
