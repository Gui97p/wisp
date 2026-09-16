package parser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseDeclaration() ast.Declaration {
	line, col := p.current.Line, p.current.Column

	decl := p.parseDeclarationInner()
	if decl == nil {
		return nil
	}
	decl.SetPos(line, col)
	return decl
}

func (p *Parser) parseDeclarationInner() ast.Declaration {
	switch p.current.Type {
	case lexer.TOKEN_CONST:
		decl := p.parseConstDeclaration()
		if decl == nil {
			return nil
		}
		return decl
	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		return decl
	case lexer.TOKEN_STRUCT:
		decl := p.parseStructDeclaration()
		if decl == nil {
			return nil
		}
		return decl
	case lexer.TOKEN_IMPORT:
		decl := p.parseImportDeclaration()
		if decl == nil {
			return nil
		}
		return decl
	case lexer.TOKEN_EXPORT:
		return p.parseExportDeclaration()
	default:
		p.error(fmt.Sprintf("expected declaration, got %s", p.current.Type))
		p.advance()
		return nil
	}
}

func (p *Parser) parseImportDeclaration() *ast.ImportDecl {
	decl := &ast.ImportDecl{}

	if !p.expect(lexer.TOKEN_STRING_LITERAL) {
		return nil
	}
	decl.Path = p.current.Literal
	decl.Alias = lastPathSegment(decl.Path)

	if p.peek.Type == lexer.TOKEN_AS {
		p.advance()
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		decl.Alias = p.current.Literal
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return decl
}

func (p *Parser) parseExportDeclaration() ast.Declaration {
	p.advance()

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		decl := p.parseConstDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		return decl

	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		return decl

	case lexer.TOKEN_STRUCT:
		decl := p.parseStructDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		return decl

	default:
		p.error("expected declaration after export")
		return nil
	}
}

func (p *Parser) parseConstDeclaration() *ast.ConstDecl {
	decl := &ast.ConstDecl{}
	p.advance()

	isTypedDecl := p.isPrimitiveType() || p.current.Type == lexer.TOKEN_MAP || (p.current.Type == lexer.TOKEN_IDENT && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_STAR))

	var currentType ast.TypeRef
	if isTypedDecl {
		ref := p.parseTypePrefix()
		if ref == nil {
			return nil
		}
		currentType = *ref
	}

	if p.current.Type != lexer.TOKEN_IDENT {
		p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
		return nil
	}
	name := p.current.Literal

	if !p.parseArraySuffix(&currentType) {
		return nil
	}
	decl.Vars = append(decl.Vars, ast.Param{Name: name, Type: currentType})

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()

		currentType.PointerDepth = p.parsePointerDepth()

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil
		}
		name := p.current.Literal

		if !p.parseArraySuffix(&currentType) {
			return nil
		}
		decl.Vars = append(decl.Vars, ast.Param{Name: name, Type: currentType})
	}

	if !p.expect(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.advance()
	decl.Values = append(decl.Values, p.parseExpression())

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		decl.Values = append(decl.Values, p.parseExpression())
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return decl
}

func lastPathSegment(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}
