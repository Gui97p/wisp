package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseFuncDeclaration() *ast.FuncDecl {
	decl := &ast.FuncDecl{}

	if p.peek.Type == lexer.TOKEN_LPAREN {
		p.advance()
		p.advance()

		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil
		}

		var name string
		if p.peek.Type == lexer.TOKEN_IDENT {
			p.advance()
			name = p.current.Literal

			if !p.checkReservedName(name) {
				return nil
			}

		}
		decl.Receiver = &ast.Param{Name: name, Type: *ref}

		if !p.expect(lexer.TOKEN_RPAREN) {
			return nil
		}
	}

	if !p.expect(lexer.TOKEN_IDENT) {
		p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
		return nil
	}
	decl.Name = p.current.Literal

	if !p.checkReservedName(decl.Name) {
		return nil
	}

	if !p.expect(lexer.TOKEN_LPAREN) {
		return nil
	}
	decl.Params = p.parseFuncParamList()

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_LBRACE || p.peek.Type == lexer.TOKEN_ARROW || p.peek.Type == lexer.TOKEN_SEMICOLON {
		decl.ReturnTypes = []ast.TypeRef{}
	} else {
		p.advance()
		decl.ReturnTypes = p.parseReturnTypes()
	}

	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		p.advance()
		return decl
	}

	if p.peek.Type == lexer.TOKEN_ARROW {
		p.advance()
		line, col := p.current.Line, p.current.Column

		stmt := p.parseReturnStatement()
		if stmt != nil {
			stmt.SetPos(line, col)
			stmt.SetEndPos(p.current.Line, p.current.Column)
			decl.Body = &ast.BlockStmt{Statements: []ast.Statement{stmt}}
		}
	} else {
		if !p.expect(lexer.TOKEN_LBRACE) {
			return nil
		}
		decl.Body = p.parseBlockStatement()

		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
	}

	return decl
}

func (p *Parser) parseFuncParamList() []ast.Param {
	params := []ast.Param{}

	if p.peek.Type == lexer.TOKEN_RPAREN {
		return params
	}

	var currentType ast.TypeRef

	p.advance()
	for p.peek.Type != lexer.TOKEN_RPAREN && p.peek.Type != lexer.TOKEN_EOF {
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
		currentType = *ref

		if p.current.Type == lexer.TOKEN_VARIADIC {
			if !p.expect(lexer.TOKEN_IDENT) {
				return nil
			}

			params = append(params, ast.Param{
				Name:     p.current.Literal,
				Type:     currentType,
				Variadic: true,
			})
			return params
		}

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

			if p.current.Type == lexer.TOKEN_LPAREN ||
				(p.isStartType() && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR || p.peek.Type == lexer.TOKEN_VARIADIC)) {
				break
			}

			if p.current.Type == lexer.TOKEN_VARIADIC {
				if !p.expect(lexer.TOKEN_IDENT) {
					return nil
				}

				params = append(params, ast.Param{
					Name:     p.current.Literal,
					Type:     currentType,
					Variadic: true,
				})
				return params
			}

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
	}

	return params
}

func (p *Parser) parseReturnTypes() []ast.TypeRef {
	returnTypes := []ast.TypeRef{}

	if p.current.Type == lexer.TOKEN_LPAREN {
		if p.looksLikeFuncTypeReturn() {
			ref := p.parseFuncTypePrefix()
			if ref == nil {
				return nil
			}
			return []ast.TypeRef{*ref}
		}

		for {
			p.advance()

			rType := p.parseTypeDefinitionPrefix()
			if rType == nil {
				return nil
			}

			if !p.parseArraySuffix(rType) {
				return nil
			}

			returnTypes = append(returnTypes, *rType)

			if p.peek.Type == lexer.TOKEN_COMMA {
				p.advance()
				continue
			}

			if !p.expect(lexer.TOKEN_RPAREN) {
				return nil
			}

			return returnTypes
		}
	} else if p.isStartType() || p.current.Type == lexer.TOKEN_STAR || p.current.Type == lexer.TOKEN_NOT {
		rType := p.parseTypeDefinitionPrefix()
		if rType == nil {
			return nil
		}

		if !p.parseArraySuffix(rType) {
			return nil
		}

		returnTypes = append(returnTypes, *rType)
		return returnTypes
	} else {
		p.errorType(p.current.Type)
		return returnTypes
	}
}

func (p *Parser) parseFuncTypePrefix() *ast.TypeRef {
	ref := &ast.TypeRef{IsFunc: true}

	if p.peek.Type != lexer.TOKEN_RPAREN {
		p.advance()
		for {
			if !p.isStartType() {
				p.errorType(p.current.Type)
				return nil
			}
			paramType := p.parseTypeDefinitionPrefix()
			if paramType == nil || !p.parseArraySuffix(paramType) {
				return nil
			}
			ref.FuncParams = append(ref.FuncParams, *paramType)

			if p.peek.Type != lexer.TOKEN_COMMA {
				break
			}
			p.advance()
			p.advance()
		}
	}

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	if p.peek.Type != lexer.TOKEN_ARROW {
		return ref
	}
	p.advance()
	p.advance()

	ref.FuncReturns = p.parseReturnTypes()
	return ref
}
