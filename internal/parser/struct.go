package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseStructDeclaration() *ast.StructDecl {
	decl := &ast.StructDecl{}

	if !p.expect(lexer.TOKEN_IDENT) {
		p.error("expected struct name in declaration")
		return nil
	}
	decl.Name = p.current.Literal

	if !p.checkReservedName(decl.Name) {
		return nil
	}

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}
	decl.Members = p.parseStructParamList()

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return decl
}

func (p *Parser) parseStructParamList() []ast.Param {
	params := []ast.Param{}

	if p.peek.Type == lexer.TOKEN_RBRACE {
		return params
	}

	for p.peek.Type != lexer.TOKEN_RBRACE && p.peek.Type != lexer.TOKEN_EOF {
		p.advance()
		if !p.isStartType() {
			p.errorType(p.current.Type)
			return params
		}
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return params
		}
		if !p.parseArraySuffix(ref) {
			return nil
		}
		p.advance()
		currentType := *ref

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return params
		}

		name := p.current.Literal

		if !p.checkReservedName(name) {
			return nil
		}

		params = append(params, ast.Param{
			Name: name,
			Type: currentType,
		})

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

			if p.current.Type != lexer.TOKEN_IDENT {
				p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
				return params
			}
			name := p.current.Literal

			if !p.checkReservedName(name) {
				return nil
			}

			params = append(params, ast.Param{
				Name: name,
				Type: currentType,
			})
		}

		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}
	}

	return params
}
