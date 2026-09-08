package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseFuncDeclaration() *ast.FuncDecl {
	decl := &ast.FuncDecl{}

	if !p.expect(lexer.TOKEN_IDENT) {
		p.error(p.current, "expected function name in declaration")
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
	decl.ReturnTypes = p.parseReturnTypes()

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

	var currentType string

	p.advance()
	for p.peek.Type != lexer.TOKEN_RPAREN && p.peek.Type != lexer.TOKEN_EOF {
		if !p.isStartType() {
			p.errorType(p.current.Type, p.current)
			return params
		}
		currentType = p.current.Literal
		p.advance()

		isPointer := false
		if p.current.Type == lexer.TOKEN_STAR {
			p.advance()
			isPointer = true
		}

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type, p.current)
			return params
		}

		params = append(params, ast.Param{
			Name:      p.current.Literal,
			Type:      currentType,
			IsPointer: isPointer,
		})

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

			if p.isStartType() && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR) {
				break
			}

			isPointer := false
			if p.current.Type == lexer.TOKEN_STAR {
				p.advance()
				isPointer = true
			}

			if p.current.Type != lexer.TOKEN_IDENT {
				p.errorExpected(lexer.TOKEN_IDENT, p.current.Type, p.current)
				return params
			}

			params = append(params, ast.Param{
				Name:      p.current.Literal,
				Type:      currentType,
				IsPointer: isPointer,
			})
		}
	}

	return params
}

func (p *Parser) parseReturnTypes() []ast.ReturnType {
	returnTypes := []ast.ReturnType{}

	if p.peek.Type == lexer.TOKEN_LBRACE {
		return returnTypes
	}
	p.advance()

	if p.current.Type == lexer.TOKEN_LPAREN {
		for {
			p.advance()

			isPointer := false
			if p.current.Type == lexer.TOKEN_STAR {
				p.advance()
				isPointer = true
			}

			if !p.isStartType() {
				p.errorType(p.current.Type, p.current)
				return returnTypes
			}

			returnTypes = append(returnTypes, ast.ReturnType{
				Type:      p.current.Literal,
				IsPointer: isPointer,
			})

			if p.peek.Type == lexer.TOKEN_RPAREN {
				p.advance()
				return returnTypes
			}

			if !p.expect(lexer.TOKEN_COMMA) {
				p.errorExpected(lexer.TOKEN_COMMA, p.current.Type, p.current)
				return returnTypes
			}
		}
	} else if p.isStartType() || p.current.Type == lexer.TOKEN_STAR {
		isPointer := false
		if p.current.Type == lexer.TOKEN_STAR {
			p.advance()
			isPointer = true
		}
		returnTypes = append(returnTypes, ast.ReturnType{
			Type:      p.current.Literal,
			IsPointer: isPointer,
		})
		return returnTypes
	} else {
		p.errorType(p.current.Type, p.current)
		return returnTypes
	}
}
