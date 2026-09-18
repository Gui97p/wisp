package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseConst() ([]ast.Param, []ast.Expression) {
	var vars []ast.Param
	var values []ast.Expression

	p.advance()

	isTypedDecl := p.isPrimitiveType() || p.current.Type == lexer.TOKEN_MAP || (p.current.Type == lexer.TOKEN_IDENT && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR))

	var currentType ast.TypeRef
	if isTypedDecl {
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil, nil
		}
		p.advance()
		currentType = *ref
	}

	if p.current.Type != lexer.TOKEN_IDENT {
		p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
		return nil, nil
	}
	name := p.current.Literal

	if !p.parseArraySuffix(&currentType) {
		return nil, nil
	}
	vars = append(vars, ast.Param{Name: name, Type: currentType})

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil, nil
		}
		name := p.current.Literal

		if !p.parseArraySuffix(&currentType) {
			return nil, nil
		}
		vars = append(vars, ast.Param{Name: name, Type: currentType})
	}

	if !p.expect(lexer.TOKEN_ASSIGN) {
		return nil, nil
	}

	p.advance()
	value := p.parseExpression()
	if value == nil {
		return nil, nil
	}
	values = append(values, value)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		value := p.parseExpression()
		if value == nil {
			return nil, nil
		}
		values = append(values, value)
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil, nil
	}

	return vars, values
}
