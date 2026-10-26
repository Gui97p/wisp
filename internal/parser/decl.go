package parser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseDeclaration() []ast.Declaration {
	line, col := p.current.Line, p.current.Column
	doc := p.docBefore(line)

	decls := p.parseDeclarationInner()
	if decls == nil {
		return nil
	}
	setDoc(decls[0], doc)
	for _, decl := range decls {
		decl.SetPos(line, col)
		decl.SetEndPos(p.currentEnd())
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
	case lexer.TOKEN_EXTERN:
		decl := p.parseExternDeclaration()
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
	case lexer.TOKEN_PUBLIC:
		return p.parseExportDeclaration(ast.ReachNone)
	case lexer.TOKEN_INTERNAL:
		return p.parseExportDeclaration(ast.ReachInternal)
	case lexer.TOKEN_SHARED:
		return p.parseExportDeclaration(ast.ReachShared)
	case lexer.TOKEN_ENUM:
		return p.parseEnumDeclaration()
	default:
		if p.startsVarDeclaration() {
			decl := p.parseVarDeclaration()
			if decl == nil {
				return nil
			}
			decls = append(decls, decl)
			break
		}
		if p.current.Type == lexer.TOKEN_IDENT && p.current.Literal == "export" {
			p.error("export was renamed to public")
			p.advance()
			return nil
		}
		p.errorf("expected declaration, got %s", p.current.Type.DisplayName())
		p.advance()
		return nil
	}

	return decls
}

func (p *Parser) startsVarDeclaration() bool {
	switch p.current.Type {
	case lexer.TOKEN_LET, lexer.TOKEN_MAP:
		return true
	case lexer.TOKEN_IDENT, lexer.TOKEN_STAR, lexer.TOKEN_LPAREN:
		return p.looksLikeTypeDeclaration()
	}
	return p.isPrimitiveType()
}

func (p *Parser) parseVarDeclaration() *ast.VarDecl {
	stmt := p.parseVarStatement()
	if stmt == nil {
		return nil
	}

	vs, ok := stmt.(*ast.VarStmt)
	if !ok {
		p.error("a switch expression cannot initialize a global")
		return nil
	}

	return &ast.VarDecl{Vars: vs.Vars, Values: vs.Values}
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

func (p *Parser) parseExportDeclaration(reach int) []ast.Declaration {
	decls := p.parseExported()
	if reach == ast.ReachNone {
		return decls
	}
	for _, d := range decls {
		switch d := d.(type) {
		case *ast.ConstDecl:
			d.Reach = reach
		case *ast.VarDecl:
			d.Reach = reach
		case *ast.FuncDecl:
			d.Reach = reach
		case *ast.StructDecl:
			d.Reach = reach
		case *ast.TypeDecl:
			d.Reach = reach
		}
	}
	return decls
}

func (p *Parser) parseExported() []ast.Declaration {
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

	case lexer.TOKEN_EXTERN:
		decl := p.parseExternDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_LET, lexer.TOKEN_MAP, lexer.TOKEN_STAR, lexer.TOKEN_LPAREN:
		decl := p.parseVarDeclaration()
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

	case lexer.TOKEN_IDENT:
		if p.looksLikeTypeDeclaration() {
			decl := p.parseVarDeclaration()
			if decl == nil {
				return nil
			}
			decl.Exported = true
			decls = append(decls, decl)
			break
		}

		p.errorf("expected declaration after public, got %s", p.current.Literal)
		return nil

	default:
		if p.isPrimitiveType() {
			decl := p.parseVarDeclaration()
			if decl == nil {
				return nil
			}
			decl.Exported = true
			decls = append(decls, decl)
			break
		}
		p.error("expected declaration after public")
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

	enum := &ast.TypeDecl{Name: name, Underlying: ast.TypeRef{Name: "int"}, IsEnum: true}
	decls = append(decls, enum)

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	var current int64 = 0
	for p.peek.Type != lexer.TOKEN_EOF && p.peek.Type != lexer.TOKEN_RBRACE {
		memberDoc := p.docBefore(p.peek.Line)
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		memberName := p.current.Literal
		memberLine, memberCol, memberEndLine, memberEndCol := p.currentIdentPos()

		if !p.checkReservedName(memberName) {
			return nil
		}

		for _, existing := range enum.Members {
			if existing.Name == memberName {
				p.errorf("duplicated enum member %s", memberName)
				return nil
			}
		}

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
		enum.Members = append(enum.Members, ast.EnumMember{
			Name:    memberName,
			Value:   current,
			Doc:     memberDoc,
			Line:    memberLine,
			Col:     memberCol,
			EndLine: memberEndLine,
			EndCol:  memberEndCol,
		})
		current++

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

func setDoc(decl ast.Declaration, doc string) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		d.Doc = doc
	case *ast.StructDecl:
		d.Doc = doc
	case *ast.TypeDecl:
		d.Doc = doc
	case *ast.ConstDecl:
		d.Doc = doc
	case *ast.VarDecl:
		d.Doc = doc
	}
}

func (p *Parser) docBefore(line int) string {
	comments := p.l.Comments()

	var block []lexer.Comment
	want := line - 1
	for i := len(comments) - 1; i >= 0; i-- {
		c := comments[i]
		if c.Line >= line {
			continue
		}
		if !c.Standalone || c.EndLine != want {
			break
		}
		block = append(block, c)
		want = c.Line - 1
	}

	lines := make([]string, 0, len(block))
	for i := len(block) - 1; i >= 0; i-- {
		lines = append(lines, commentText(block[i].Text))
	}
	return strings.Join(lines, "\n")
}

func commentText(raw string) string {
	if text, ok := strings.CutPrefix(raw, "//"); ok {
		text = strings.TrimPrefix(text, " ")
		return strings.TrimRight(text, " \r")
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(raw, "/*"), "*/")
	rows := strings.Split(inner, "\n")
	for i, row := range rows {
		row = strings.TrimSpace(row)
		row = strings.TrimPrefix(row, "*")
		rows[i] = strings.TrimPrefix(row, " ")
	}
	return strings.Trim(strings.Join(rows, "\n"), "\n")
}
