package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseInterfaceDeclaration() *ast.InterfaceDecl {
	decl := &ast.InterfaceDecl{}

	selector, at, ok := p.optionalSelector()
	if !ok {
		return nil
	}
	decl.Selector, decl.SelectorAt = selector, at

	if !p.expect(lexer.TOKEN_IDENT) {
		p.error("expected interface name in declaration")
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

	if p.peek.Type == lexer.TOKEN_IMPLEMENTS {
		p.error("an interface does not implement another: compose interfaces with &, as in interface ReadWriter { Reader & Writer }")
		return nil
	}

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	for p.peek.Type != lexer.TOKEN_RBRACE && p.peek.Type != lexer.TOKEN_EOF {
		p.advance()
		doc := p.docBefore(p.current.Line)

		if p.current.Type == lexer.TOKEN_FUNC {
			method := p.parseInterfaceMethod(doc)
			if method == nil {
				return nil
			}
			decl.Methods = append(decl.Methods, *method)
			continue
		}

		if decl.Constraint != nil {
			p.error("an interface has one type expression: join the parts with & or |")
			return nil
		}
		constraint := p.parseConstraint()
		if constraint == nil {
			return nil
		}
		decl.Constraint = constraint
		if p.peek.Type == lexer.TOKEN_SEMICOLON {
			p.advance()
		}
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return decl
}

func (p *Parser) parseInterfaceMethod(doc string) *ast.InterfaceMethod {
	method := &ast.InterfaceMethod{Doc: doc}
	start := p.currentPosition()

	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}
	method.Name = p.current.Literal
	method.NameAt = p.currentPosition()
	if !p.checkReservedName(method.Name) {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_LT {
		p.error("an interface method cannot take type parameters: put them on the interface")
		return nil
	}

	if !p.expect(lexer.TOKEN_LPAREN) {
		return nil
	}
	method.Params = p.parseFuncParamList()
	if p.cVariadic {
		p.cVariadic = false
		p.error("a variadic without a name is only for extern functions")
		return nil
	}
	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_SEMICOLON || p.peek.Type == lexer.TOKEN_RBRACE {
		method.ReturnTypes = []ast.TypeRef{}
	} else {
		p.advance()
		types, fallible, ok := p.parseReturnSpec(true, false)
		if !ok {
			return nil
		}
		method.ReturnTypes = types
		method.Fallible = fallible
	}

	endLine, endCol := p.currentEnd()
	method.At = ast.Position{Line: start.Line, Col: start.Col, EndLine: endLine, EndCol: endCol}

	if p.peek.Type == lexer.TOKEN_RBRACE {
		p.errorExpected(lexer.TOKEN_SEMICOLON, lexer.TOKEN_RBRACE)
		return method
	}
	p.endStatement()
	return method
}

func (p *Parser) parseConstraint() *ast.Constraint {
	left := p.constraintAnd()
	for left != nil && p.peek.Type == lexer.TOKEN_PIPE {
		p.advance()
		p.advance()
		right := p.constraintAnd()
		if right == nil {
			return nil
		}
		left = &ast.Constraint{Kind: ast.ConstraintOr, Left: left, Right: right, At: joinAt(left.At, right.At)}
	}
	return left
}

func (p *Parser) constraintAnd() *ast.Constraint {
	left := p.constraintNot()
	for left != nil && p.peek.Type == lexer.TOKEN_AMP {
		p.advance()
		p.advance()
		right := p.constraintNot()
		if right == nil {
			return nil
		}
		left = &ast.Constraint{Kind: ast.ConstraintAnd, Left: left, Right: right, At: joinAt(left.At, right.At)}
	}
	return left
}

func (p *Parser) constraintNot() *ast.Constraint {
	start := p.currentPosition()

	switch p.current.Type {
	case lexer.TOKEN_NOT:
		p.advance()
		inner := p.constraintNot()
		if inner == nil {
			return nil
		}
		return &ast.Constraint{Kind: ast.ConstraintNot, Left: inner, At: joinAt(start, inner.At)}
	case lexer.TOKEN_LPAREN:
		p.advance()
		inner := p.parseConstraint()
		if inner == nil || !p.expect(lexer.TOKEN_RPAREN) {
			return nil
		}
		return inner
	}

	if !p.isPointerType() {
		p.errorType(p.current.Type)
		return nil
	}
	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nil
	}
	endLine, endCol := p.currentEnd()
	return &ast.Constraint{Kind: ast.ConstraintType, Type: ref, At: ast.Position{Line: start.Line, Col: start.Col, EndLine: endLine, EndCol: endCol}}
}

func joinAt(from, to ast.Position) ast.Position {
	return ast.Position{Line: from.Line, Col: from.Col, EndLine: to.EndLine, EndCol: to.EndCol}
}

func (p *Parser) optionalImplements() ([]ast.TypeRef, bool) {
	if p.peek.Type != lexer.TOKEN_IMPLEMENTS {
		return nil, true
	}
	p.advance()

	var refs []ast.TypeRef
	for {
		p.advance()
		if !p.isPointerType() {
			p.errorType(p.current.Type)
			return nil, false
		}
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil, false
		}
		refs = append(refs, *ref)
		if p.peek.Type != lexer.TOKEN_COMMA {
			return refs, true
		}
		p.advance()
	}
}
