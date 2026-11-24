package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func ParseExpressionText(src string) (ast.Expression, string) {
	p := NewParser(lexer.NewLexer([]byte(src)))
	if p.current.Type == lexer.TOKEN_EOF {
		return nil, "an expression is expected"
	}
	expr := p.parseExpression()
	if len(p.errors) > 0 {
		return nil, p.errors[0].Message
	}
	if expr == nil {
		return nil, "an expression is expected"
	}
	if p.peek.Type != lexer.TOKEN_EOF {
		return nil, "unexpected " + p.peek.Type.DisplayName() + " after the expression"
	}
	return expr, ""
}
