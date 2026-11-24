package analyser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/parser"
)

type formatItem struct {
	expr       ast.Expression
	positional bool
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

	rewritten, items, problem := a.scanFormat(literal)
	if problem != "" {
		a.errorf(literal, "%s", problem)
		return
	}
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

func (a *Analyser) scanFormat(literal *ast.StringLiteral) (string, []formatItem, string) {
	text := literal.Value
	var out strings.Builder
	var items []formatItem

	at := ast.Position{}
	at.Line, at.Col = literal.Position()
	at.EndLine, at.EndCol = literal.EndPosition()

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
			if hasSpec {
				placeholder = "{:" + spec + "}"
			}
			if strings.TrimSpace(source) == "" {
				out.WriteString(text[i : end+1])
				items = append(items, formatItem{positional: true})
			} else {
				parsed, problem := parser.ParseExpressionText(source)
				if problem != "" {
					message := "the expression {" + strings.TrimSpace(source) + "} in the format string is not valid: " + problem
					if strings.Contains(source, "?") {
						message += " (a ternary can be written without parentheses, so check the other parts of it)"
					}
					return "", nil, message
				}
				out.WriteString(placeholder)
				items = append(items, formatItem{expr: ast.Clone(parsed, ast.CloneOptions{At: &at})})
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
		case '?':
			if depth == 0 {
				if j+1 < len(inner) && inner[j+1] == '?' {
					j++
				} else {
					pending++
				}
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
