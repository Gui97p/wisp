package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseStructDeclaration() *ast.StructDecl {
	decl := &ast.StructDecl{}

	selector, at, ok := p.optionalSelector()
	if !ok {
		return nil
	}
	decl.Selector, decl.SelectorAt = selector, at

	if !p.expect(lexer.TOKEN_IDENT) {
		p.error("expected struct name in declaration")
		return nil
	}
	decl.Name = p.current.Literal

	if !p.checkReservedName(decl.Name) {
		return nil
	}

	typeParams, ok := p.optionalTypeParams()
	if !ok {
		return nil
	}
	decl.TypeParams = typeParams

	implements, ok := p.optionalImplements()
	if !ok {
		return nil
	}
	decl.Implements = implements

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
		doc := p.docBefore(p.current.Line)
		reach := ast.ReachNone
		if p.oldSharedWord(true) {
			reach = ast.ReachProject
			p.advance()
		}
		switch p.current.Type {
		case lexer.TOKEN_INTERNAL:
			reach = ast.ReachProject
			p.advance()
		case lexer.TOKEN_PRIVATE:
			reach = ast.ReachFile
			p.advance()
		case lexer.TOKEN_PUBLIC:
			reach = ast.ReachPublic
			p.advance()
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
		currentType := *ref

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
			Doc:   doc,
			Reach: reach,
			Line:  line, Col: col, EndLine: endLine, EndCol: endCol,
		})

		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()

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
				Doc:   doc,
				Reach: reach,
				Line:  line, Col: col, EndLine: endLine, EndCol: endCol,
			})
		}

		p.endStatement()
	}

	return params
}
