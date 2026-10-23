package lsp

import (
	"regexp"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	typeName = `\*{0,}(?:[A-Za-z_]\w*\.)?[A-Za-z_]?\w*`
	funcHead = `\bfunc\s+(?:\([^)]*\)\s*)?[A-Za-z_]\w*\s*`
)

var (
	structHeader    = regexp.MustCompile(`(?:^|[\s;}])struct\s+([A-Za-z_]\w*)$`)
	enumHeader      = regexp.MustCompile(`(?:^|[\s;}])enum\s+[A-Za-z_]\w*$`)
	funcHeader      = regexp.MustCompile(funcHead + `$`)
	returnHeader    = regexp.MustCompile(funcHead + `\((?:[^()]|\([^()]*\))*\)\s*!?\s*$`)
	typeSegment     = regexp.MustCompile(`^` + typeName + `$`)
	qualifiedType   = regexp.MustCompile(`^\*{0,}([A-Za-z_]\w*)\.`)
	mapTail         = regexp.MustCompile(`\bmap\[[^\[\]]*\]` + typeName + `$`)
	lambdaFirst     = regexp.MustCompile(`^\s*\*{0,}[A-Za-z_][\w.]*\s+[A-Za-z_]\w*\s*$`)
	returnTail      = regexp.MustCompile(`^(?:export\s+)?(?:extern\s+)?` + funcHead + `\((?:[^()]|\([^()]*\))*\)\s*!?\s*(` + typeName + `)$`)
	namePosition    = regexp.MustCompile(`^\*{0,}[\w.]+\s+\w*$`)
	declarationHead = regexp.MustCompile(`^(?:(?:export|extern)\s+)*`)
)

var topLevelKeywords = []string{"func", "struct", "enum", "type", "const", "import", "export", "internal", "shared", "extern"}

type typeContext struct {
	typing   bool
	topLevel bool
	own      string
	module   string
}

func typingType(segment string) (typeContext, bool) {
	segment = strings.TrimLeft(segment, " \t")
	if !typeSegment.MatchString(segment) {
		return typeContext{}, false
	}
	ctx := typeContext{typing: true}
	if m := qualifiedType.FindStringSubmatch(segment); m != nil {
		ctx.module = m[1]
	}
	return ctx, true
}

func afterLast(text, separators string) string {
	return text[strings.LastIndexAny(text, separators)+1:]
}

func typeContextAt(text string) (typeContext, bool) {
	if m := mapTail.FindString(text); m != "" {
		if ctx, ok := typingType(m[strings.LastIndex(m, "]")+1:]); ok {
			return ctx, true
		}
	}

	index, char, open := innermostBracket(text)
	if !open {
		return topLevelContext(afterLast(text, ";}"))
	}

	header := strings.TrimRight(text[:index], " \t\r\n")
	body := text[index+1:]

	switch char {
	case '{':
		if enumHeader.MatchString(header) {
			return typeContext{}, true
		}
		m := structHeader.FindStringSubmatch(header)
		if m == nil {
			return typeContext{}, false
		}
		segment := afterLast(body, ";\n")
		if strings.Contains(segment, ",") {
			return typeContext{own: m[1]}, true
		}
		ctx, _ := typingType(segment)
		ctx.own = m[1]
		return ctx, true
	case '(':
		switch {
		case funcHeader.MatchString(header):
			ctx, _ := typingType(afterLast(body, ","))
			return ctx, true
		case returnHeader.MatchString(header):
			ctx, _ := typingType(afterLast(body, ","))
			return ctx, true
		}
		if i := strings.LastIndex(body, ","); i >= 0 && lambdaFirst.MatchString(body[:strings.Index(body, ",")]) {
			ctx, _ := typingType(body[i+1:])
			return ctx, true
		}
	case '[':
		if strings.HasSuffix(header, "map") {
			ctx, _ := typingType(body)
			return ctx, true
		}
	}
	return typeContext{}, false
}

func topLevelContext(segment string) (typeContext, bool) {
	segment = strings.TrimLeft(segment, " \t\r\n")

	if m := returnTail.FindStringSubmatch(segment); m != nil {
		ctx, _ := typingType(m[1])
		return ctx, true
	}

	declaration := declarationHead.ReplaceAllString(segment, "")
	if typeSegment.MatchString(declaration) {
		ctx, _ := typingType(declaration)
		ctx.topLevel = !strings.HasPrefix(declaration, "*") && !strings.Contains(declaration, ".")
		return ctx, true
	}
	if namePosition.MatchString(declaration) {
		return typeContext{}, true
	}
	return typeContext{}, false
}

func isTypeSymbol(sym *analyser.Symbol) bool {
	return sym.Kind == analyser.STRUCT || sym.Kind == analyser.TYPE
}

func typeItems(info *analyser.Info, line, col int, ctx typeContext) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}

	if ctx.module != "" {
		if info == nil {
			return items
		}
		owner, ok := scopeAt(info, line, col).Resolve(ctx.module)
		if !ok {
			return items
		}
		if mt, isModule := owner.Type.(*analyser.ModuleType); isModule {
			for _, ex := range mt.Exports {
				if isTypeSymbol(ex) {
					items = append(items, symbolItem(ex, ""))
				}
			}
		}
		return items
	}

	if ctx.topLevel {
		for _, k := range topLevelKeywords {
			items = append(items, item(k, protocol.CompletionItemKindKeyword, ""))
		}
	}

	for _, p := range primitiveNames {
		items = append(items, item(p, protocol.CompletionItemKindClass, "type"))
	}

	seen := map[string]bool{}
	if info != nil {
		for _, sym := range scopeAt(info, line, col).Visible() {
			if isTypeSymbol(sym) || sym.Kind == analyser.MODULE {
				seen[sym.Name] = true
				items = append(items, symbolItem(sym, ""))
			}
		}
	}

	if ctx.own != "" && !seen[ctx.own] {
		items = append(items, item(ctx.own, protocol.CompletionItemKindStruct, "struct"))
	}
	return items
}
