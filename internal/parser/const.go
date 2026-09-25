package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseConstIdentifiers() []ast.Param {
	var vars []ast.Param

	p.advance()

	isTypedDecl := p.isPrimitiveType() || p.current.Type == lexer.TOKEN_MAP || (p.current.Type == lexer.TOKEN_IDENT && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR))

	var currentType ast.TypeRef
	if isTypedDecl {
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil
		}
		if !p.parseArraySuffix(ref) {
			return nil
		}
		p.advance()
		currentType = *ref
	}

	if p.current.Type != lexer.TOKEN_IDENT {
		p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
		return nil
	}
	name := p.current.Literal
	line, col, endLine, endCol := p.currentIdentPos()

	if !p.checkReservedName(name) {
		return nil
	}

	vars = append(vars, ast.Param{Name: name, Type: currentType, Line: line, Col: col, EndLine: endLine, EndCol: endCol})

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil
		}
		name := p.current.Literal
		line, col, endLine, endCol := p.currentIdentPos()

		if !p.checkReservedName(name) {
			return nil
		}

		vars = append(vars, ast.Param{Name: name, Type: currentType, Line: line, Col: col, EndLine: endLine, EndCol: endCol})
	}

	if !p.expect(lexer.TOKEN_ASSIGN) {
		return nil
	}

	return vars
}

func (p *Parser) parseConstValues() []ast.Expression {
	var values []ast.Expression

	value := p.parseExpression()
	if value == nil {
		return nil
	}
	values = append(values, value)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		value := p.parseExpression()
		if value == nil {
			return nil
		}
		values = append(values, value)
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return values
}
