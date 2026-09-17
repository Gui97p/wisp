package parser

import (
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
	var decl ast.Declaration

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		decl = p.parseConstDeclaration()
	case lexer.TOKEN_FUNC:
		decl = p.parseFuncDeclaration()
	case lexer.TOKEN_STRUCT:
		decl = p.parseStructDeclaration()
	case lexer.TOKEN_TYPE:
		decl = p.parseTypeDeclaration()
	case lexer.TOKEN_IMPORT:
		decl = p.parseImportDeclaration()
	case lexer.TOKEN_EXPORT:
		decl = p.parseExportDeclaration()
	default:
		p.errorf("expected declaration, got %s", p.current.Type.DisplayName())
		p.advance()
		return nil
	}

	if decl == nil {
		return nil
	}
	return decl
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

	case lexer.TOKEN_TYPE:
		decl := p.parseTypeDeclaration()
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

	vars, values := p.parseConst()
	if vars == nil || values == nil {
		return nil
	}

	decl.Vars = vars
	decl.Values = values

	return decl
}

func lastPathSegment(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

func (p *Parser) parseTypeDeclaration() *ast.TypeDecl {
	decl := &ast.TypeDecl{}

	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}
	decl.Name = p.current.Literal

	p.advance()
	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nil
	}
	if !p.parseArraySuffix(ref) {
		return nil
	}
	decl.Underlying = *ref

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return decl
}
