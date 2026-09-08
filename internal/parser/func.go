package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseFuncDeclaration() *ast.FuncDecl {
	decl := &ast.FuncDecl{}

	if !p.expect(lexer.TOKEN_IDENT) {
		p.errors = append(p.errors, "expected function name in declaration")
		return nil
	}
	decl.Name = p.current.Literal

	if !p.expect(lexer.TOKEN_LPAREN) {
		return nil
	}
	decl.Params = p.parseFuncParamList()

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}
	decl.ReturnTypes = nil //p.parseReturnTypes()

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}
	decl.Body = nil //p.parseBlockStatement()

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return decl
}

func (p *Parser) parseFuncParamList() []ast.Param {
	params := []ast.Param{}

	if p.peek.Type == lexer.TOKEN_RPAREN {
		return params
	}
	p.advance()

	var currentType string

	for p.peek.Type != lexer.TOKEN_RPAREN && p.peek.Type != lexer.TOKEN_EOF {
		if !p.isStartType() {
			p.errors = append(p.errors, "expected valid type")
			return params
		}
		currentType = p.current.Literal
		p.advance()

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errors = append(p.errors, "expected identifier")
			return params
		}

		params = append(params, ast.Param{
			Name: p.current.Literal,
			Type: currentType,
		})

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

			if p.isStartType() && p.peek.Type == lexer.TOKEN_IDENT {
				break
			}

			if p.current.Type != lexer.TOKEN_IDENT {
				p.errors = append(p.errors, "expected identifier")
				return params
			}

			params = append(params, ast.Param{
				Name: p.current.Literal,
				Type: currentType,
			})
		}
	}

	return params
}
