package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseFuncDeclaration() *ast.FuncDecl {
	decl := &ast.FuncDecl{}

	if !p.expect(lexer.TOKEN_IDENT) {
		p.error("expected function name in declaration")
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

	if p.peek.Type == lexer.TOKEN_ARROW {
		p.advance()

		stmt := p.parseReturnStatement()
		if stmt != nil {
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
		name, depth := p.parseTypePrefix()
		currentType.Name = name
		currentType.PointerDepth = depth

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return params
		}
		name = p.current.Literal

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

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

			if p.isStartType() && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR) {
				break
			}

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

func (p *Parser) parseReturnTypes() []ast.TypeRef {
	returnTypes := []ast.TypeRef{}

	if p.peek.Type == lexer.TOKEN_LBRACE || p.peek.Type == lexer.TOKEN_ARROW {
		return returnTypes
	}
	p.advance()

	if p.current.Type == lexer.TOKEN_LPAREN {
		for {
			p.advance()

			rType := ast.TypeRef{}
			rType.PointerDepth = p.parsePointerDepth()

			if !p.isStartType() {
				p.errorType(p.current.Type)
				return nil
			}

			rType.Name = p.current.Literal

			isArray, size, ok := p.parseArraySuffix()
			if !ok {
				return nil
			}
			rType.IsArray = isArray
			rType.ArraySize = size

			returnTypes = append(returnTypes, rType)

			if p.peek.Type == lexer.TOKEN_COMMA {
				p.advance()
				continue
			}

			if !p.expect(lexer.TOKEN_RPAREN) {
				return nil
			}

			return returnTypes
		}
	} else if p.isStartType() || p.current.Type == lexer.TOKEN_STAR {
		rType := ast.TypeRef{}
		rType.PointerDepth = p.parsePointerDepth()

		if !p.isStartType() {
			p.errorType(p.current.Type)
			return nil
		}
		rType.Name = p.current.Literal

		isArray, size, ok := p.parseArraySuffix()
		if !ok {
			return nil
		}
		rType.IsArray = isArray
		rType.ArraySize = size

		returnTypes = append(returnTypes, rType)
		return returnTypes
	} else {
		p.errorType(p.current.Type)
		return returnTypes
	}
}
