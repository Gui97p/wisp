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
		name, depth := p.parseTypePrefix()
		currentType := ast.TypeRef{Name: name, PointerDepth: depth}

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return params
		}

		isArray, size, ok := p.parseArraySuffix()
		if !ok {
			return nil
		}
		currentType.IsArray = isArray
		currentType.ArraySize = size

		params = append(params, ast.Param{
			Name: p.current.Literal,
			Type: currentType,
		})

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

			currentType.PointerDepth = p.parsePointerDepth()

			if p.current.Type != lexer.TOKEN_IDENT {
				p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
				return params
			}
			name := p.current.Literal

			isArray, size, ok := p.parseArraySuffix()
			if !ok {
				return nil
			}
			currentType.IsArray = isArray
			currentType.ArraySize = size

			params = append(params, ast.Param{
				Name: name,
				Type: currentType,
			})
		}
	}

	return params
}
