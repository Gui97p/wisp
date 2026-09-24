package parser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseDeclaration() []ast.Declaration {
	line, col := p.current.Line, p.current.Column

	decls := p.parseDeclarationInner()
	if decls == nil {
		return nil
	}
	for _, decl := range decls {
		decl.SetPos(line, col)
		decl.SetEndPos(p.current.Line, p.current.Column)
	}

	return decls
}

func (p *Parser) parseDeclarationInner() []ast.Declaration {
	decls := []ast.Declaration{}

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		decl := p.parseConstDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_STRUCT:
		decl := p.parseStructDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_TYPE:
		decl := p.parseTypeDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_IMPORT:
		decl := p.parseImportDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_EXPORT:
		return p.parseExportDeclaration()
	case lexer.TOKEN_ENUM:
		return p.parseEnumDeclaration()
	default:
		p.errorf("expected declaration, got %s", p.current.Type.DisplayName())
		p.advance()
		return nil
	}

	return decls
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

func (p *Parser) parseExportDeclaration() []ast.Declaration {
	p.advance()
	decls := []ast.Declaration{}

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		decl := p.parseConstDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_ENUM:
		decls = p.parseEnumDeclaration()
		if decls == nil {
			return nil
		}
		for _, decl := range decls {
			switch d := decl.(type) {
			case *ast.ConstDecl:
				d.Exported = true
			case *ast.TypeDecl:
				d.Exported = true
			default:
				return nil
			}
		}

	case lexer.TOKEN_STRUCT:
		decl := p.parseStructDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_TYPE:
		decl := p.parseTypeDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	default:
		p.error("expected declaration after export")
		return nil
	}

	return decls
}

func (p *Parser) parseConstDeclaration() *ast.ConstDecl {
	decl := &ast.ConstDecl{}

	vars := p.parseConstIdentifiers()
	if vars == nil {
		return nil
	}
	decl.Vars = vars

	p.advance()
	values := p.parseConstValues()
	if values == nil {
		return nil
	}
	decl.Values = values

	return decl
}

func lastPathSegment(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx != -1 {
		path = path[idx+1:]
	}
	return strings.TrimSuffix(path, ".wsp")
}

func (p *Parser) parseTypeDeclaration() *ast.TypeDecl {
	decl := &ast.TypeDecl{}

	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}
	decl.Name = p.current.Literal

	if !p.checkReservedName(decl.Name) {
		return nil
	}

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

func (p *Parser) parseEnumDeclaration() []ast.Declaration {
	decls := []ast.Declaration{}

	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}
	name := p.current.Literal

	if !p.checkReservedName(name) {
		return nil
	}

	decls = append(decls, &ast.TypeDecl{Name: name, Underlying: ast.TypeRef{Name: "int"}})

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	var current int64 = 0
	for p.peek.Type != lexer.TOKEN_EOF && p.peek.Type != lexer.TOKEN_RBRACE {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		decl := &ast.ConstDecl{}

		memberName := p.current.Literal

		if !p.checkReservedName(memberName) {
			return nil
		}

		decl.Vars = append(decl.Vars, ast.Param{
			Name: memberName,
			Type: ast.TypeRef{Name: name},
		})

		if p.peek.Type == lexer.TOKEN_ASSIGN {
			p.advance()
			if !p.expect(lexer.TOKEN_INT_LITERAL) {
				return nil
			}
			num, err := p.parseIntLiteral(p.current.Literal)
			if err != nil {
				p.errorf("error converting int literal %s", err.Error())
				return nil
			}
			if num < current {
				p.errorf("enum expected a value greater than %d, got %d", current-1, num)
			}
			current = num
		}
		decl.Values = append(decl.Values, &ast.IntLiteral{Value: current})
		current++

		decls = append(decls, decl)

		if p.peek.Type != lexer.TOKEN_RBRACE {
			if !p.expect(lexer.TOKEN_COMMA) {
				return nil
			}
		}
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return decls
}
