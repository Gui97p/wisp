package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseConstIdentifiers() []ast.Param {
	var vars []ast.Param

	p.advance()

	isTypedDecl := p.isPrimitiveType() || (p.current.Type == lexer.TOKEN_IDENT && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR))

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

func (p *Parser) parseConstDeclarations() []ast.Declaration {
	if p.constGroupAhead() {
		return p.parseConstGroup()
	}
	decl := p.parseConstDeclaration()
	if decl == nil {
		return nil
	}
	return []ast.Declaration{decl}
}

func (p *Parser) constGroupAhead() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	p.advance()
	if p.current.Type != lexer.TOKEN_LBRACE {
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil || !p.parseArraySuffix(ref) {
			return false
		}
		p.advance()
	}
	return p.current.Type == lexer.TOKEN_LBRACE
}

func (p *Parser) parseConstGroup() []ast.Declaration {
	p.advance()

	var groupType ast.TypeRef
	if p.current.Type != lexer.TOKEN_LBRACE {
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil || !p.parseArraySuffix(ref) {
			return nil
		}
		p.advance()
		groupType = *ref
	}

	var decls []ast.Declaration
	for p.peek.Type != lexer.TOKEN_RBRACE {
		p.advance()
		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			p.skipConstGroup()
			return nil
		}

		line, col := p.current.Line, p.current.Column
		name := p.current.Literal
		nameLine, nameCol, nameEndLine, nameEndCol := p.currentIdentPos()
		if !p.checkReservedName(name) {
			p.skipConstGroup()
			return nil
		}
		if !p.expect(lexer.TOKEN_ASSIGN) {
			p.skipConstGroup()
			return nil
		}
		p.advance()

		values := p.parseConstValues()
		if values == nil {
			p.skipConstGroup()
			return nil
		}

		decl := &ast.ConstDecl{
			Vars:   []ast.Param{{Name: name, Type: groupType, Line: nameLine, Col: nameCol, EndLine: nameEndLine, EndCol: nameEndCol}},
			Values: values,
		}
		decl.SetPos(line, col)
		decl.SetEndPos(p.currentEnd())
		decls = append(decls, decl)
	}
	p.advance()

	if len(decls) == 0 {
		p.error("a constant group needs at least one member")
		return nil
	}

	return decls
}

func (p *Parser) skipConstGroup() {
	depth := 0
	for p.current.Type != lexer.TOKEN_EOF {
		switch p.current.Type {
		case lexer.TOKEN_LBRACE:
			depth++
		case lexer.TOKEN_RBRACE:
			if depth == 0 {
				return
			}
			depth--
		}
		p.advance()
	}
}
