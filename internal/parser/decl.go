package parser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseTags() ([]ast.Tag, bool) {
	var tags []ast.Tag
	for p.current.Type == lexer.TOKEN_HASH {
		line := p.current.Line
		hash := p.currentPosition()
		if p.peek.Type != lexer.TOKEN_IDENT || p.peek.Line != line {
			p.errorAt(hash, "expected a tag name after #, such as #naked")
			p.skipLine(line)
			p.advance()
			return nil, false
		}
		p.advance()
		tag := ast.Tag{Name: p.current.Literal, At: hash}
		tag.At.EndLine, tag.At.EndCol = p.currentEnd()
		if tag.Name == "naked" {
			p.nakedTag = true
		}

		if p.peek.Type == lexer.TOKEN_LPAREN && p.peek.Line == line {
			p.advance()
			if p.peek.Type != lexer.TOKEN_RPAREN {
				for {
					p.advance()
					arg := p.parseExpression()
					if arg == nil {
						return nil, false
					}
					tag.Args = append(tag.Args, arg)
					if p.peek.Type == lexer.TOKEN_COMMA {
						p.advance()
						continue
					}
					break
				}
			}
			if !p.expect(lexer.TOKEN_RPAREN) {
				return nil, false
			}
			tag.At.EndLine, tag.At.EndCol = p.currentEnd()
		}
		tags = append(tags, tag)
		p.advance()
	}
	return tags, true
}

func (p *Parser) parseDeclaration() []ast.Declaration {
	doc := p.docBefore(p.current.Line)

	tags, ok := p.parseTags()
	if !ok {
		p.nakedTag = false
		return nil
	}
	line, col := p.current.Line, p.current.Column
	if len(tags) > 0 && p.current.Type == lexer.TOKEN_EOF {
		p.errorAt(tags[len(tags)-1].At, "a tag needs a declaration after it")
		return nil
	}
	if p.externBlockAhead() {
		if len(tags) > 0 {
			p.errorAt(tags[0].At, "a tag goes on each declaration inside the extern block, not on the block")
			return nil
		}
		return p.parseExternBlock(false)
	}

	decls := p.parseDeclarationInner()
	p.nakedTag = false
	if decls == nil {
		return nil
	}
	if len(tags) > 0 {
		for _, decl := range decls {
			tagged, can := decl.(ast.Tags)
			if !can {
				p.errorAt(tags[0].At, "this declaration cannot take tags")
				return nil
			}
			tagged.SetTags(tags)
			if fd, isFunc := decl.(*ast.FuncDecl); isFunc {
				applySymbolTag(fd)
			}
		}
	}
	if !p.placed[decls[0]] {
		setDoc(decls[0], doc)
	}
	for _, decl := range decls {
		if p.placed[decl] {
			continue
		}
		if member, isConst := decl.(*ast.ConstDecl); isConst {
			if placed, _ := member.Position(); placed != 0 {
				continue
			}
		}
		decl.SetPos(line, col)
		decl.SetEndPos(p.currentEnd())
	}

	return decls
}

func (p *Parser) parseDeclarationInner() []ast.Declaration {
	decls := []ast.Declaration{}

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		return p.parseConstDeclarations()
	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_EXTERN:
		if p.peek.Type == lexer.TOKEN_TYPE {
			p.advance()
			decl := p.parseExternType()
			if decl == nil {
				return nil
			}
			decls = append(decls, decl)
			break
		}
		decl := p.parseExternDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_INLINE:
		decl := p.parseInlineDeclaration()
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	case lexer.TOKEN_NATIVE:
		decl := p.parseNativeDeclaration()
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
	case lexer.TOKEN_INTERFACE:
		decl := p.parseInterfaceDeclaration()
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
		return p.parseExportDeclaration(ast.ReachProject)
	case lexer.TOKEN_PRIVATE:
		return p.parsePrivateDeclaration()
	case lexer.TOKEN_ENUM:
		return p.parseEnumDeclaration()
	default:
		if p.oldSharedWord(false) {
			return p.parseExportDeclaration(ast.ReachProject)
		}
		if p.current.Type == lexer.TOKEN_IDENT && p.current.Literal == "staticAssert" && p.peek.Type == lexer.TOKEN_LPAREN {
			call, ok := p.parseExpression().(*ast.CallExpr)
			if !ok || !p.expect(lexer.TOKEN_SEMICOLON) {
				return nil
			}
			decls = append(decls, &ast.StaticAssertDecl{Call: call})
			break
		}
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
		if p.current.Type == lexer.TOKEN_HASH {
			p.error("a directive goes at the top of the file, before any declaration")
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
	case lexer.TOKEN_LET, lexer.TOKEN_VAR:
		return true
	case lexer.TOKEN_IDENT, lexer.TOKEN_STAR, lexer.TOKEN_QUESTION_MARK, lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET:
		return p.looksLikeTypeDeclaration()
	}
	return p.isPrimitiveType()
}

func (p *Parser) parseVarDeclaration() *ast.VarDecl {
	var selector string
	var selectorAt ast.Position
	if p.current.Type == lexer.TOKEN_VAR {
		sel, at, ok := p.optionalSelector()
		if !ok {
			return nil
		}
		selector, selectorAt = sel, at
		p.advance()
		if p.current.Type == lexer.TOKEN_LET || p.peek.Type == lexer.TOKEN_ASSIGN {
			p.error("var is followed by a type: write var int count; or var<debug> int count = 1;")
			return nil
		}
	}

	var stmt ast.Statement
	if p.current.Type == lexer.TOKEN_LET && p.peek.Type == lexer.TOKEN_LT {
		sel, at, ok := p.optionalSelector()
		if !ok {
			return nil
		}
		selector, selectorAt = sel, at
		p.advance()
		stmt = p.parseVarNames(&ast.VarStmt{}, ast.TypeRef{})
	} else {
		stmt = p.parseVarStatement()
	}
	if stmt == nil {
		return nil
	}

	vs, ok := stmt.(*ast.VarStmt)
	if !ok {
		p.error("a match expression cannot initialize a global")
		return nil
	}

	return &ast.VarDecl{Vars: vs.Vars, Values: vs.Values, Selector: selector, SelectorAt: selectorAt}
}

func (p *Parser) parseImportDeclaration() *ast.ImportDecl {
	decl := &ast.ImportDecl{}

	selector, at, ok := p.optionalSelector()
	if !ok {
		return nil
	}
	decl.Selector, decl.SelectorAt = selector, at

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

	p.endStatement()

	return decl
}

func (p *Parser) parsePrivateDeclaration() []ast.Declaration {
	decls := p.parseExported()
	for _, d := range decls {
		switch d := d.(type) {
		case *ast.ConstDecl:
			d.Exported, d.Reach = false, ast.ReachFile
		case *ast.VarDecl:
			d.Exported, d.Reach = false, ast.ReachFile
		case *ast.FuncDecl:
			d.Exported, d.Reach = false, ast.ReachFile
		case *ast.StructDecl:
			d.Exported, d.Reach = false, ast.ReachFile
		case *ast.TypeDecl:
			d.Exported, d.Reach = false, ast.ReachFile
		case *ast.InterfaceDecl:
			d.Exported, d.Reach = false, ast.ReachFile
		}
	}
	return decls
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
		case *ast.InterfaceDecl:
			d.Reach = reach
		}
	}
	return decls
}

func (p *Parser) parseExported() []ast.Declaration {
	word := p.current.Literal
	p.advance()
	decls := []ast.Declaration{}

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		decls = p.parseConstDeclarations()
		if decls == nil {
			return nil
		}
		for _, decl := range decls {
			decl.(*ast.ConstDecl).Exported = true
		}

	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_EXTERN:
		if p.externBlockAhead() {
			return p.parseExternBlock(true)
		}
		if p.peek.Type == lexer.TOKEN_TYPE {
			p.advance()
			decl := p.parseExternType()
			if decl == nil {
				return nil
			}
			decl.Exported = true
			decls = append(decls, decl)
			break
		}
		decl := p.parseExternDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_INLINE:
		decl := p.parseInlineDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_NATIVE:
		decl := p.parseNativeDeclaration()
		if decl == nil {
			return nil
		}
		decl.Exported = true
		decls = append(decls, decl)

	case lexer.TOKEN_LET, lexer.TOKEN_VAR, lexer.TOKEN_STAR, lexer.TOKEN_QUESTION_MARK, lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET:
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

	case lexer.TOKEN_INTERFACE:
		decl := p.parseInterfaceDeclaration()
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

		p.errorf("expected a declaration after %s, got %s", word, p.current.Literal)
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
		p.errorf("expected a declaration after %s", word)
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

	selector, at, ok := p.optionalSelector()
	if !ok {
		return nil
	}
	decl.Selector, decl.SelectorAt = selector, at

	if !p.expect(lexer.TOKEN_IDENT) {
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

	p.advance()
	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nil
	}
	decl.Underlying = *ref

	implements, ok := p.optionalImplements()
	if !ok {
		return nil
	}
	decl.Implements = implements

	p.endStatement()

	return decl
}

func (p *Parser) parseEnumDeclaration() []ast.Declaration {
	decls := []ast.Declaration{}

	selector, selectorAt, ok := p.optionalSelector()
	if !ok {
		return nil
	}

	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}
	name := p.current.Literal

	if !p.checkReservedName(name) {
		return nil
	}

	typeParams, ok := p.optionalTypeParams()
	if !ok {
		return nil
	}

	implements, ok := p.optionalImplements()
	if !ok {
		return nil
	}

	enum := &ast.TypeDecl{Name: name, TypeParams: typeParams, Implements: implements, Underlying: ast.TypeRef{Name: "int"}, IsEnum: true, Selector: selector, SelectorAt: selectorAt}
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

		var fields []ast.Param
		payload, numbered := false, false
		if p.peek.Type == lexer.TOKEN_LBRACE {
			p.advance()
			fields = p.parseStructParamList()
			if !p.expect(lexer.TOKEN_RBRACE) {
				return nil
			}
			payload = true
			enum.Data = true
		}

		if p.peek.Type == lexer.TOKEN_ASSIGN {
			numbered = true
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
			Name:     memberName,
			Fields:   fields,
			Payload:  payload,
			Numbered: numbered,
			Value:    current,
			Doc:      memberDoc,
			Line:     memberLine,
			Col:      memberCol,
			EndLine:  memberEndLine,
			EndCol:   memberEndCol,
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

	if len(enum.TypeParams) > 0 && !enum.Data {
		p.errorf("%s takes type parameters, so some variant has to carry data that uses them", name)
		return nil
	}

	if enum.Data {
		for _, m := range enum.Members {
			if m.Numbered {
				p.errorf("%s has variants with data, so its members cannot be numbered", name)
				return nil
			}
		}
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
	case *ast.InterfaceDecl:
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
		if strings.HasPrefix(c.Text, "///") {
			block = append(block, c)
		}
		want = c.Line - 1
	}

	lines := make([]string, 0, len(block))
	for i := len(block) - 1; i >= 0; i-- {
		lines = append(lines, commentText(block[i].Text))
	}
	return strings.Join(lines, "\n")
}

func commentText(raw string) string {
	text := strings.TrimPrefix(raw, "///")
	text = strings.TrimPrefix(text, " ")
	return strings.TrimRight(text, " \r")
}

func (p *Parser) oldSharedWord(field bool) bool {
	if p.current.Type != lexer.TOKEN_IDENT || p.current.Literal != "shared" {
		return false
	}
	next := p.peek.Type
	declaration := next == lexer.TOKEN_FUNC || next == lexer.TOKEN_STRUCT || next == lexer.TOKEN_ENUM || next == lexer.TOKEN_TYPE ||
		next == lexer.TOKEN_INTERFACE || next == lexer.TOKEN_CONST || next == lexer.TOKEN_INLINE || next == lexer.TOKEN_NATIVE ||
		next == lexer.TOKEN_EXTERN || next == lexer.TOKEN_VAR
	typed := next >= lexer.TOKEN_INT && next <= lexer.TOKEN_STRING || next == lexer.TOKEN_STAR || next == lexer.TOKEN_QUESTION_MARK || next == lexer.TOKEN_LBRACKET
	if !declaration && !(field && typed) {
		return false
	}
	endLine, endCol := p.currentEnd()
	p.errors.Add("", p.current.Line, p.current.Column, endLine, endCol, "shared was replaced by internal, which now means the whole project").WithHelp("write internal where you wrote shared")
	return true
}

func applySymbolTag(fd *ast.FuncDecl) {
	tag := fd.TagNamed("symbol")
	if tag == nil || len(tag.Args) != 1 {
		return
	}
	text, ok := tag.Args[0].(*ast.StringLiteral)
	if !ok || text.Value == "" {
		return
	}
	if fd.Extern {
		fd.ExternName = text.Value
	} else {
		fd.CExport = text.Value
	}
}
