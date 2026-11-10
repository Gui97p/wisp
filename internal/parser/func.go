package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseFuncDeclaration() *ast.FuncDecl {
	return p.parseFunc(false)
}

func (p *Parser) parseInlineDeclaration() *ast.FuncDecl {
	if !p.expect(lexer.TOKEN_FUNC) {
		return nil
	}

	decl := p.parseFunc(false)
	if decl == nil {
		return nil
	}
	decl.Inline = true
	return decl
}

func (p *Parser) parseExternDeclaration() *ast.FuncDecl {
	if !p.expect(lexer.TOKEN_FUNC) {
		return nil
	}

	decl := p.parseFunc(true)
	if decl == nil {
		return nil
	}
	if decl.Receiver != nil {
		p.error("an extern function cannot have a receiver")
		return nil
	}

	decl.Extern = true
	return decl
}

func (p *Parser) parseFunc(extern bool) *ast.FuncDecl {
	decl := &ast.FuncDecl{}

	if p.peek.Type == lexer.TOKEN_LPAREN {
		p.advance()
		p.advance()

		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil
		}

		var name string
		var line, col, endLine, endCol int
		if p.peek.Type == lexer.TOKEN_IDENT {
			p.advance()
			name = p.current.Literal
			line, col, endLine, endCol = p.currentIdentPos()

			if !p.checkReservedName(name) {
				return nil
			}

		}
		decl.Receiver = &ast.Param{Name: name, Type: *ref, Line: line, Col: col, EndLine: endLine, EndCol: endCol}

		if !p.expect(lexer.TOKEN_RPAREN) {
			return nil
		}
	}

	if !p.expect(lexer.TOKEN_IDENT) {
		p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
		return nil
	}
	decl.Name = p.current.Literal
	line, col, endLine, endCol := p.currentIdentPos()
	decl.NameAt = ast.Position{Line: line, Col: col, EndLine: endLine, EndCol: endCol}

	if !p.checkReservedName(decl.Name) {
		return nil
	}

	if !p.expect(lexer.TOKEN_LPAREN) {
		return nil
	}
	decl.Params = p.parseFuncParamList()
	if p.cVariadic {
		p.cVariadic = false
		if !extern {
			p.error("a variadic without a name is only for extern functions")
			return nil
		}
		decl.CVariadic = true
	}

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	if p.native != nil {
		return p.finishNative(decl)
	}

	if p.peek.Type == lexer.TOKEN_LBRACE || p.peek.Type == lexer.TOKEN_ARROW || p.peek.Type == lexer.TOKEN_SEMICOLON || p.peek.Type == lexer.TOKEN_AS {
		decl.ReturnTypes = []ast.TypeRef{}
	} else {
		p.advance()
		types, fallible, ok := p.parseReturnSpec(true, extern)
		if !ok {
			return nil
		}
		decl.ReturnTypes = types
		decl.Fallible = fallible
	}

	if p.peek.Type == lexer.TOKEN_AS {
		p.advance()
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		alias := p.current.Literal
		if !p.checkReservedName(alias) {
			return nil
		}
		if extern {
			decl.ExternName = decl.Name
			decl.Name = alias
		} else {
			decl.CExport = alias
		}
	} else if extern {
		decl.ExternName = decl.Name
	}

	if extern {
		if p.peek.Type == lexer.TOKEN_LBRACE || p.peek.Type == lexer.TOKEN_ARROW {
			p.error("an extern function is defined in C and cannot have a body")
			return nil
		}
		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}
		return decl
	}

	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		p.errorf("function %s has no body: declare a function defined in C with extern", decl.Name)
		p.advance()
		return nil
	}

	if p.peek.Type == lexer.TOKEN_ARROW {
		p.advance()
		line, col := p.current.Line, p.current.Column

		stmt := p.parseReturnStatement()
		if stmt != nil {
			stmt.SetPos(line, col)
			stmt.SetEndPos(p.currentEnd())
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

func (p *Parser) parseAnyVariadic(params []ast.Param) []ast.Param {
	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}

	name := p.current.Literal
	line, col, endLine, endCol := p.currentIdentPos()
	if !p.checkReservedName(name) {
		return nil
	}

	return append(params, ast.Param{
		Name:     name,
		Variadic: true,
		Any:      true,
		Line:     line, Col: col, EndLine: endLine, EndCol: endCol,
	})
}

func (p *Parser) parseFuncParamList() []ast.Param {
	params := []ast.Param{}
	p.cVariadic = false

	if p.peek.Type == lexer.TOKEN_RPAREN {
		return params
	}

	var currentType ast.TypeRef

	p.advance()
	for (p.peek.Type != lexer.TOKEN_RPAREN || p.current.Type == lexer.TOKEN_LPAREN) && p.peek.Type != lexer.TOKEN_EOF {
		isConst := false
		if p.current.Type == lexer.TOKEN_CONST {
			isConst = true
			p.advance()
		}
		if p.current.Type == lexer.TOKEN_VARIADIC {
			return p.parseAnyVariadic(params)
		}
		if !p.isPointerType() {
			p.errorType(p.current.Type)
			return params
		}
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return params
		}
		p.advance()
		currentType = *ref

		if p.current.Type == lexer.TOKEN_VARIADIC {
			if !p.expect(lexer.TOKEN_IDENT) {
				return nil
			}

			line, col, endLine, endCol := p.currentIdentPos()
			params = append(params, ast.Param{
				Name:     p.current.Literal,
				Type:     currentType,
				Variadic: true,
				Line:     line, Col: col, EndLine: endLine, EndCol: endCol,
			})
			return params
		}

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return params
		}
		name := p.current.Literal
		line, col, endLine, endCol := p.currentIdentPos()

		if !p.checkReservedName(name) {
			return nil
		}

		params = append(params, ast.Param{
			Name:  name,
			Type:  currentType,
			Const: isConst,
			Line:  line, Col: col, EndLine: endLine, EndCol: endCol,
		})

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

			if p.current.Type == lexer.TOKEN_LPAREN || p.current.Type == lexer.TOKEN_STAR || p.current.Type == lexer.TOKEN_LBRACKET || p.current.Type == lexer.TOKEN_CONST ||
				(p.isStartType() && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR || p.peek.Type == lexer.TOKEN_VARIADIC || p.peek.Type == lexer.TOKEN_DOT)) {
				break
			}

			if p.current.Type == lexer.TOKEN_VARIADIC {
				if p.peek.Type == lexer.TOKEN_RPAREN {
					p.cVariadic = true
					return params
				}
				return p.parseAnyVariadic(params)
			}

			if p.current.Type != lexer.TOKEN_IDENT {
				p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
				return params
			}
			name := p.current.Literal
			line, col, endLine, endCol := p.currentIdentPos()

			if !p.checkReservedName(name) {
				return nil
			}

			params = append(params, ast.Param{
				Name:  name,
				Type:  currentType,
				Const: isConst,
				Line:  line, Col: col, EndLine: endLine, EndCol: endCol,
			})
		}
	}

	return params
}

func (p *Parser) parseReturnSpec(header, extern bool) ([]ast.TypeRef, bool, bool) {
	fallible := false
	if p.current.Type == lexer.TOKEN_NOT {
		fallible = true
		switch p.peek.Type {
		case lexer.TOKEN_LBRACE, lexer.TOKEN_ARROW, lexer.TOKEN_SEMICOLON:
			return []ast.TypeRef{}, true, true
		}
		p.advance()
	}

	types := p.parseReturnTypes(header, extern)
	if types == nil {
		return nil, false, false
	}
	return types, fallible, true
}

func (p *Parser) parseReturnTypes(header, extern bool) []ast.TypeRef {
	returnTypes := []ast.TypeRef{}

	if p.current.Type == lexer.TOKEN_LPAREN {
		if p.looksLikeFuncTypeReturn(header, extern) {
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
	} else if p.isStartType() || p.current.Type == lexer.TOKEN_STAR {
		rType := p.parseTypeDefinitionPrefix()
		if rType == nil {
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
			if !p.isPointerType() {
				p.errorType(p.current.Type)
				return nil
			}
			paramType := p.parseTypeDefinitionPrefix()
			if paramType == nil {
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

	if p.current.Type == lexer.TOKEN_LPAREN && p.peek.Type == lexer.TOKEN_RPAREN {
		p.advance()
		ref.FuncReturns = []ast.TypeRef{}
		return ref
	}

	types, fallible, ok := p.parseReturnSpec(false, false)
	if !ok {
		return nil
	}
	ref.FuncReturns = types
	ref.FuncFallible = fallible
	return ref
}
