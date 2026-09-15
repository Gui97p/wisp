package parser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseDeclaration() ast.Declaration {
	switch p.current.Type {
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
		decl := p.parseImportDecl()
		if decl == nil {
			return nil
		}
		return decl
	case lexer.TOKEN_EXPORT:
		return p.parseExportDecl()
	default:
		p.error(fmt.Sprintf("expected declaration, got %s", p.current.Type))
		p.advance()
		return nil
	}
}

func (p *Parser) parseImportDecl() *ast.ImportDecl {
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

func (p *Parser) parseExportDecl() ast.Declaration {
	p.advance()

	switch p.current.Type {
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

func lastPathSegment(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}
