package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) closeAngle() bool {
	switch p.peek.Type {
	case lexer.TOKEN_GT:
		p.advance()
		return true
	case lexer.TOKEN_SHIFT_RIGHT:
		p.peek.Type = lexer.TOKEN_GT
		p.peek.Literal = ">"
		p.peek.Column++
		return true
	}
	p.errorExpected(lexer.TOKEN_GT, p.peek.Type)
	return false
}

func (p *Parser) startsValueArg() bool {
	switch p.current.Type {
	case lexer.TOKEN_INT_LITERAL, lexer.TOKEN_FLOAT_LITERAL, lexer.TOKEN_CHAR_LITERAL, lexer.TOKEN_STRING_LITERAL,
		lexer.TOKEN_TRUE, lexer.TOKEN_FALSE, lexer.TOKEN_MINUS:
		return true
	}
	return false
}

func (p *Parser) parseTypeArgs() ([]ast.TypeArg, bool) {
	var args []ast.TypeArg
	for {
		p.advance()
		if p.startsValueArg() {
			value := p.parsePrefix()
			if value == nil {
				return nil, false
			}
			args = append(args, ast.TypeArg{Expr: value})
		} else {
			ref := p.parseTypeDefinitionPrefix()
			if ref == nil {
				return nil, false
			}
			args = append(args, ast.TypeArg{Type: ref})
		}

		if p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			continue
		}
		if !p.closeAngle() {
			return nil, false
		}
		return args, true
	}
}

func (p *Parser) parseTypeParams() ([]ast.TypeParam, bool) {
	var params []ast.TypeParam
	for {
		p.advance()

		param := ast.TypeParam{}
		if p.current.Type == lexer.TOKEN_CONST {
			p.advance()
			ref := p.parseTypeDefinitionPrefix()
			if ref == nil {
				return nil, false
			}
			param.Const, param.Type = true, *ref
			if !p.expect(lexer.TOKEN_IDENT) {
				return nil, false
			}
		} else if p.current.Type != lexer.TOKEN_IDENT {
			p.errorf("expected the name of a type parameter, got %s", p.current.Type.DisplayName())
			return nil, false
		}

		param.Name = p.current.Literal
		param.At = p.currentPosition()
		if !p.checkReservedName(param.Name) {
			return nil, false
		}
		params = append(params, param)

		if p.peek.Type == lexer.TOKEN_COLON {
			p.error("a type parameter has no bound yet: it takes any type")
			return nil, false
		}
		if p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			continue
		}
		if !p.closeAngle() {
			return nil, false
		}
		return params, true
	}
}

func (p *Parser) optionalTypeParams() ([]ast.TypeParam, bool) {
	if p.peek.Type != lexer.TOKEN_LT {
		return nil, true
	}
	p.advance()
	return p.parseTypeParams()
}

func isNameExpr(e ast.Expression) bool {
	switch v := e.(type) {
	case *ast.IdentLiteral:
		return true
	case *ast.MemberExpr:
		_, ok := v.Object.(*ast.IdentLiteral)
		return ok
	}
	return false
}

func (p *Parser) tryGenericRef(left ast.Expression) *ast.GenericExpr {
	cp := p.checkpoint()
	p.advance()
	args, ok := p.parseTypeArgs()
	if ok && (p.peek.Type == lexer.TOKEN_LPAREN || (p.allowStructLiteral && p.peek.Type == lexer.TOKEN_LBRACE) ||
		(p.typeOperands && (p.peek.Type == lexer.TOKEN_RPAREN || p.peek.Type == lexer.TOKEN_COMMA))) {
		return &ast.GenericExpr{Base: left, Args: args}
	}
	p.restore(cp)
	return nil
}
