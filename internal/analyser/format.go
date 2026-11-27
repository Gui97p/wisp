package analyser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/parser"
)

type formatItem struct {
	expr       ast.Expression
	positional bool
	open       ast.Position
	close      ast.Position
	spec       ast.Position
}

type FormatSite struct {
	Opens  []ast.Position
	Closes []ast.Position
	Specs  []ast.Position
	Exprs  []ast.Expression
}

func indexAt(text string, line, col int) int {
	l, c := 1, 0
	for i := 0; i < len(text); i++ {
		b := text[i]
		switch {
		case b == '\n':
			l++
			c = 0
			continue
		case b&0xC0 == 0x80:
			continue
		case b >= 0xF0:
			c += 2
		default:
			c++
		}
		if l == line && c >= col {
			return i
		}
	}
	return len(text) - 1
}

func formatParam(fd *ast.FuncDecl) int {
	for k, p := range fd.Params {
		if p.Any && k > 0 && len(fd.Params[k-1].Type.Layers) == 0 && fd.Params[k-1].Type.Name == "string" && fd.Params[k-1].Type.Module == "" {
			return k - 1
		}
	}
	return -1
}

func (a *Analyser) expandFormat(expr *ast.CallExpr, ft *FuncType) {
	if ft == nil || ft.Decl == nil || ft.Decl.TagNamed("format") == nil {
		return
	}
	index := formatParam(ft.Decl)
	if index < 0 || index >= len(expr.Args) {
		return
	}
	literal, ok := expr.Args[index].(*ast.StringLiteral)
	if !ok {
		return
	}

	a.noteFormatLiteral(literal)
	rewritten, items, problem := a.scanFormat(literal)
	if problem != "" {
		a.errorf(literal, "%s", problem)
		return
	}
	a.noteFormatSite(items)
	inline := false
	for _, item := range items {
		if !item.positional {
			inline = true
		}
	}
	if !inline {
		return
	}
	if expr.Spread {
		a.errorf(literal, "a format string with {expressions} cannot be combined with a spread: write the values in the string, or pass them as arguments")
		return
	}

	explicit := expr.Args[index+1:]
	args := append([]ast.Expression{}, expr.Args[:index]...)
	text := &ast.StringLiteral{Value: rewritten}
	a.at(literal, text)
	args = append(args, text)
	next := 0
	for _, item := range items {
		if item.positional {
			if next < len(explicit) {
				args = append(args, explicit[next])
				next++
			}
			continue
		}
		args = append(args, item.expr)
	}
	args = append(args, explicit[next:]...)
	expr.Args = args
}

func (a *Analyser) noteFormatLiteral(literal *ast.StringLiteral) {
	at := ast.Position{}
	at.Line, at.Col = literal.Position()
	at.EndLine, at.EndCol = literal.EndPosition()
	if at.Line != 0 {
		a.info.FormatLiterals = append(a.info.FormatLiterals, at)
	}
}

func SplitFormatSpec(inner string) (source, spec string, hasSpec bool) {
	return splitSpec(inner)
}

func (a *Analyser) noteFormatSite(items []formatItem) {
	var site FormatSite
	for _, item := range items {
		if item.open.Line != 0 {
			site.Opens = append(site.Opens, item.open)
			site.Closes = append(site.Closes, item.close)
		}
		if item.spec.Line != 0 {
			site.Specs = append(site.Specs, item.spec)
		}
		if item.expr != nil {
			site.Exprs = append(site.Exprs, item.expr)
		}
	}
	if len(site.Opens) == 0 && len(site.Exprs) == 0 {
		return
	}
	a.info.FormatSites = append(a.info.FormatSites, site)
}

func (a *Analyser) scanFormat(literal *ast.StringLiteral) (string, []formatItem, string) {
	text := literal.Value
	var out strings.Builder
	var items []formatItem

	at := ast.Position{}
	at.Line, at.Col = literal.Position()
	at.EndLine, at.EndCol = literal.EndPosition()

	mapped := len(literal.Map) == len(text)
	pointAt := func(index int) ast.Position {
		if !mapped || index < 0 || index >= len(literal.Map) {
			return ast.Position{}
		}
		p := literal.Map[index]
		return ast.Position{Line: p.Line, Col: p.Col, EndLine: p.Line, EndCol: p.Col}
	}
	span := func(from, to int) ast.Position {
		start, end := pointAt(from), pointAt(to)
		if start.Line == 0 || start.Line != end.Line {
			return ast.Position{}
		}
		return ast.Position{Line: start.Line, Col: start.Col, EndLine: end.Line, EndCol: end.Col}
	}

	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == '{' && i+1 < len(text) && text[i+1] == '{':
			out.WriteString("{{")
			i += 2
		case c == '}' && i+1 < len(text) && text[i+1] == '}':
			out.WriteString("}}")
			i += 2
		case c == '{':
			end := closingBrace(text, i+1)
			if end < 0 {
				return "", nil, "the format string has a { that is never closed: write {{ for a brace"
			}
			inner := text[i+1 : end]
			source, spec, hasSpec := splitSpec(inner)
			placeholder := "{}"
			item := formatItem{open: pointAt(i), close: pointAt(end)}
			if hasSpec {
				placeholder = "{:" + spec + "}"
				item.spec = span(i+1+len(source), end-1)
			}
			if strings.TrimSpace(source) == "" {
				out.WriteString(text[i : end+1])
				item.positional = true
				items = append(items, item)
			} else {
				parsed, problem := parser.ParseExpressionText(source)
				if problem != "" {
					message := "the expression {" + strings.TrimSpace(source) + "} in the format string is not valid: " + problem
					if strings.Contains(source, "=>") {
						message += " (an if that gives a value can be written without parentheses, so check the other parts of it)"
					}
					return "", nil, message
				}
				out.WriteString(placeholder)
				options := ast.CloneOptions{At: &at}
				if mapped {
					base := i + 1
					options = ast.CloneOptions{Remap: func(line, col int) (int, int) {
						p := pointAt(base + indexAt(source, line, col))
						if p.Line == 0 {
							return at.Line, at.Col
						}
						return p.Line, p.Col
					}}
				}
				item.expr = ast.Clone(parsed, options)
				items = append(items, item)
			}
			i = end + 1
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String(), items, ""
}

func skipQuoted(text string, i int) int {
	quote := text[i]
	j := i + 1
	for j < len(text) && text[j] != quote {
		if text[j] == '\\' && quote != '`' {
			j++
		}
		j++
	}
	return j
}

func closingBrace(text string, from int) int {
	depth := 0
	for j := from; j < len(text); j++ {
		switch text[j] {
		case '"', '\'', '`':
			j = skipQuoted(text, j)
		case '(', '[', '{':
			depth++
		case ')', ']':
			depth--
		case '}':
			if depth == 0 {
				return j
			}
			depth--
		}
	}
	return -1
}

func splitSpec(inner string) (string, string, bool) {
	depth := 0
	pending := 0
	for j := 0; j < len(inner); j++ {
		switch inner[j] {
		case '"', '\'', '`':
			j = skipQuoted(inner, j)
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '=':
			if depth == 0 && j+1 < len(inner) && inner[j+1] == '>' {
				pending++
				j++
			}
		case ':':
			if depth == 0 {
				if pending > 0 {
					pending--
					continue
				}
				return inner[:j], inner[j+1:], true
			}
		}
	}
	return inner, "", false
}

func (a *Analyser) errorCall(near ast.Node, args ...ast.Expression) *ast.CallExpr {
	name := &ast.IdentLiteral{Value: "Error", Synthetic: true}
	a.at(near, name)
	call := &ast.CallExpr{Name: name, Args: args}
	a.at(near, call)
	return call
}

func messageTemplate(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == '{' && i+1 < len(text) && text[i+1] == '{':
			out.WriteByte('{')
			i += 2
		case c == '}' && i+1 < len(text) && text[i+1] == '}':
			out.WriteByte('}')
			i += 2
		case c == '{':
			end := closingBrace(text, i+1)
			if end < 0 {
				out.WriteString(text[i:])
				return out.String()
			}
			out.WriteString("...")
			i = end + 1
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String()
}

func isHereParam(fd *ast.FuncDecl) bool {
	return len(fd.Params) > 0 && fd.Params[0].Type.Name == "Here" && len(fd.Params[0].Type.Layers) == 0 && fd.Params[0].Type.Module == ""
}

func (a *Analyser) supplyHere(expr *ast.CallExpr, ft *FuncType) {
	if ft == nil || ft.Decl == nil || ft.Decl.TagNamed("caller") == nil || !isHereParam(ft.Decl) {
		return
	}
	if len(expr.Args) > 0 {
		if call, ok := expr.Args[0].(*ast.CallExpr); ok {
			if name, isIdent := call.Name.(*ast.IdentLiteral); isIdent && name.Synthetic && name.Value == "Error" {
				return
			}
		}
	}
	template := "here"
	if index := formatParam(ft.Decl); index == 1 && len(expr.Args) > 0 {
		if literal, isLiteral := expr.Args[0].(*ast.StringLiteral); isLiteral {
			template = messageTemplate(literal.Value)
			if strings.Trim(template, ".") == "" {
				template = "error"
			}
		}
	}
	text := &ast.StringLiteral{Value: template}
	a.at(expr, text)
	expr.Args = append([]ast.Expression{a.errorCall(expr, text)}, expr.Args...)
}
