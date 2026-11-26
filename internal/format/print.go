package format

import (
	"strings"

	"github.com/Gui97p/wisp/internal/lexer"
)

type opener struct {
	line  int
	brace bool
}

type printer struct {
	src     []byte
	tokens  []token
	roles   []role
	gaps    []gap
	natives map[int]int

	out        strings.Builder
	line       int
	hasContent bool
	lineIndent int
	stack      []opener
}

func isCloser(k lexer.TokenType) bool {
	return k == lexer.TOKEN_RPAREN || k == lexer.TOKEN_RBRACKET || k == lexer.TOKEN_RBRACE
}

func isOpener(k lexer.TokenType) bool {
	return k == lexer.TOKEN_LPAREN || k == lexer.TOKEN_LBRACKET || k == lexer.TOKEN_LBRACE
}

func (p *printer) write(s string) {
	p.out.WriteString(s)
	p.line += strings.Count(s, "\n")
}

func (p *printer) lineBreak(n int) {
	if p.out.Len() > 0 {
		p.write(strings.Repeat("\n", min(n, 2)))
	}
	p.hasContent = false
}

func (p *printer) indent(n int) {
	p.write(strings.Repeat("\t", n))
	p.lineIndent = n
}

func (p *printer) depthFor(i int) int {
	closers := 0
	for j := i; j < len(p.tokens); j++ {
		if !isCloser(p.tokens[j].kind) {
			break
		}
		if j > i && (p.gaps[j].nlAfter > 0 || len(p.gaps[j].comments) > 0) {
			break
		}
		closers++
	}
	frames := p.stack
	if closers > len(frames) {
		closers = len(frames)
	}
	frames = frames[:len(frames)-closers]

	seen := map[int]bool{}
	for _, f := range frames {
		seen[f.line] = true
	}
	depth := len(seen)

	if len(frames) > 0 && !frames[len(frames)-1].brace {
		return depth
	}
	return depth + p.continuation(i)
}

func (p *printer) continuation(i int) int {
	if i == 0 {
		return 0
	}
	pr, nr := p.roles[i-1], p.roles[i]
	nk := p.tokens[i].kind
	switch {
	case pr == roleBinary || pr == roleTernaryQ || pr == roleTernaryColon:
		return 1
	case nr == roleBinary && !isCloser(nk) && nk != lexer.TOKEN_ARROW:
		return 1
	case nr == roleTernaryQ || nr == roleTernaryColon:
		return 1
	case nr == roleMemberDot:
		return 1
	}
	return 0
}

func (p *printer) print() []byte {
	n := len(p.tokens)
	for i := 0; i <= n; i++ {
		p.gap(i)
		if i == n {
			break
		}
		t := p.tokens[i]
		p.write(t.text)
		p.hasContent = true

		if isOpener(t.kind) {
			p.stack = append(p.stack, opener{line: p.line, brace: t.kind == lexer.TOKEN_LBRACE})
		}
		if isCloser(t.kind) && len(p.stack) > 0 {
			p.stack = p.stack[:len(p.stack)-1]
		}

		if closeAt, ok := p.natives[i]; ok {
			p.native(i, closeAt)
			i = closeAt
			p.write("}")
			p.stack = p.stack[:len(p.stack)-1]
		}
	}

	out := p.out.String()
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil
	}
	return []byte(out + "\n")
}

func (p *printer) gap(i int) {
	g := p.gaps[i]
	lastLineComment := false

	for _, c := range g.comments {
		if c.nlBefore == 0 && p.hasContent {
			if strings.HasPrefix(c.text, "//") {
				p.write("  " + c.text)
				lastLineComment = true
			} else {
				p.write(" " + c.text)
			}
			continue
		}
		p.lineBreak(max(c.nlBefore, 1))
		p.indent(p.commentIndent(i))
		p.write(c.text)
		p.hasContent = true
		lastLineComment = strings.HasPrefix(c.text, "//")
	}

	if i == len(p.tokens) {
		return
	}

	nl := g.nlAfter
	if lastLineComment && nl == 0 {
		nl = 1
	}
	switch {
	case i == 0 && len(g.comments) == 0:
	case nl == 0:
		if i > 0 && p.hasContent && p.needSpace(i) {
			p.write(" ")
		}
	default:
		p.lineBreak(nl)
		p.indent(p.depthFor(i))
	}
}

func (p *printer) commentIndent(i int) int {
	if i >= len(p.tokens) {
		return 0
	}
	return p.depthFor(i)
}

func (p *printer) native(open, closeAt int) {
	raw := string(p.src[p.tokens[open].end:p.tokens[closeAt].start])
	trimmed := strings.TrimSpace(raw)
	switch {
	case trimmed == "":
		return
	case !strings.Contains(raw, "\n"):
		p.write(" " + trimmed + " ")
		return
	}

	lines := strings.Split(raw, "\n")
	if head := strings.TrimSpace(lines[0]); head != "" {
		p.write(" " + head)
	}
	body := lines[1:]
	if len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}

	common := ""
	first := true
	for _, line := range body {
		if strings.TrimSpace(line) == "" {
			continue
		}
		ws := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if first {
			common, first = ws, false
			continue
		}
		k := 0
		for k < len(common) && k < len(ws) && common[k] == ws[k] {
			k++
		}
		common = common[:k]
	}

	inner := strings.Repeat("\t", p.lineIndent+1)
	for _, line := range body {
		p.write("\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		p.write(inner + strings.TrimRight(strings.TrimPrefix(line, common), " \t\r"))
	}
	p.write("\n" + strings.Repeat("\t", p.lineIndent))
}

func (p *printer) needSpace(i int) bool {
	prev, next := p.tokens[i-1], p.tokens[i]
	pr, nr := p.roles[i-1], p.roles[i]
	pk, nk := prev.kind, next.kind

	switch {
	case nk == lexer.TOKEN_SEMICOLON || nk == lexer.TOKEN_COMMA:
		return false
	case pk == lexer.TOKEN_HASH:
		return false
	case nr == roleHash:
		return true
	case nk == lexer.TOKEN_RPAREN || nk == lexer.TOKEN_RBRACKET:
		return false
	case pk == lexer.TOKEN_LPAREN || pk == lexer.TOKEN_LBRACKET:
		return false
	case pk == lexer.TOKEN_DOT:
		return false
	case nk == lexer.TOKEN_DOT:
		return nr == rolePrefixDot
	case pr == roleRange || nr == roleRange:
		return false
	case nr == rolePostfix:
		return false
	case nr == roleDeferBang:
		return false
	case pr == roleDeferBang:
		return true
	case nr == roleDeclBang:
		return true
	case pr == roleDeclBang:
		return nk == lexer.TOKEN_LBRACE || nk == lexer.TOKEN_ARROW
	case pr == roleUnary || pr == roleOptionalQ:
		return false
	case pr == roleTypeBracketClose:
		return false
	case nr == roleGenericOpen || pr == roleGenericOpen || nr == roleGenericClose:
		return false
	case pr == roleGenericClose && (nk == lexer.TOKEN_LPAREN || nr == roleLiteralOpen):
		return false
	case nk == lexer.TOKEN_LPAREN:
		if operandEnd(prev, pr) {
			return false
		}
		return true
	case nk == lexer.TOKEN_LBRACKET:
		return !operandEnd(prev, pr)
	case nk == lexer.TOKEN_LBRACE:
		return nr != roleLiteralOpen
	case pk == lexer.TOKEN_LBRACE:
		if nk == lexer.TOKEN_RBRACE {
			return false
		}
		return pr == roleBlockOpen
	case nk == lexer.TOKEN_RBRACE:
		return nr == roleBlockClose
	case nr == roleTernaryColon || pr == roleTernaryColon:
		return true
	case nr == roleKeyColon:
		return false
	case pr == roleKeyColon:
		return true
	case nr == roleLabelColon:
		return true
	case pr == roleLabelColon:
		return false
	case nr == roleTightColon || pr == roleTightColon:
		return false
	case nr == roleTernaryQ || pr == roleTernaryQ:
		return true
	case nr == roleCapOpen:
		return true
	case pr == roleCapOpen:
		return false
	case nr == roleCapClose:
		return false
	case pr == roleCapClose:
		return true
	case nr == roleSelectorOpen || pr == roleSelectorOpen || nr == roleSelectorClose:
		return false

	case pr == roleSelectorClose:
		return true
	case nr == roleDeclDots:
		return true
	case pr == roleDeclDots:
		return false
	case nr == roleSpreadDots:
		return false
	}
	return true
}
