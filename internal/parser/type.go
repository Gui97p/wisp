package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseTypeDefinitionPrefix() *ast.TypeRef {
	if p.current.Type == lexer.TOKEN_NOT {
		p.error("`!` marks the whole return list: write `func f() !(int, string)`")
		return nil
	}

	if p.current.Type == lexer.TOKEN_LPAREN {
		return p.parseFuncTypePrefix()
	}

	ref := &ast.TypeRef{}
	ref.PointerDepth = p.parsePointerDepth()
	ref.Name = p.current.Literal
	ref.Line, ref.Col, ref.EndLine, ref.EndCol = p.currentIdentPos()
	if p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_DOT {
		p.advance()
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		ref.Module = ref.Name
		ref.Name = p.current.Literal
		_, _, ref.EndLine, ref.EndCol = p.currentIdentPos()
	}

	return ref
}

func (p *Parser) parseArraySuffix(ref *ast.TypeRef) bool {
	ref.Dims = nil

	for p.peek.Type == lexer.TOKEN_LBRACKET {
		p.advance()
		if p.peek.Type == lexer.TOKEN_RBRACKET {
			p.advance()
			ref.Dims = append(ref.Dims, ast.ArrayDim{IsSpan: true})
			continue
		}

		p.advance()
		size := p.parseExpression()
		if size == nil {
			return false
		}

		if !p.expect(lexer.TOKEN_RBRACKET) {
			return false
		}

		if literal, ok := size.(*ast.IntLiteral); ok {
			ref.Dims = append(ref.Dims, ast.ArrayDim{Size: literal.Value})
			continue
		}
		ref.Dims = append(ref.Dims, ast.ArrayDim{Expr: size, Text: ast.ExprText(size)})
	}

	return true
}

func (p *Parser) looksLikeTypeDeclaration() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	p.parsePointerDepth()

	switch {
	case p.isPrimitiveType():
		return p.peek.Type != lexer.TOKEN_LPAREN
	case p.current.Type == lexer.TOKEN_LPAREN:
		ref := p.parseFuncTypePrefix()
		if ref == nil {
			return false
		}
		return p.peek.Type == lexer.TOKEN_IDENT
	case p.current.Type == lexer.TOKEN_IDENT:
		if p.peek.Type == lexer.TOKEN_DOT {
			p.advance()
			if !p.expect(lexer.TOKEN_IDENT) {
				return false
			}
		}
		if p.peek.Type == lexer.TOKEN_IDENT {
			return true
		}
		if p.peek.Type != lexer.TOKEN_LBRACKET {
			return false
		}
		var ref ast.TypeRef
		if !p.parseArraySuffix(&ref) {
			return false
		}
		return p.peek.Type == lexer.TOKEN_IDENT
	default:
		return false
	}
}

func (p *Parser) looksLikeLambda() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	p.parseFuncParamList()
	if !p.expect(lexer.TOKEN_RPAREN) {
		return false
	}

	if p.peek.Type != lexer.TOKEN_ARROW {
		p.advance()
		if _, _, ok := p.parseReturnSpec(); !ok {
			return false
		}
	}

	return p.peek.Type == lexer.TOKEN_ARROW
}

func (p *Parser) looksLikeFuncTypeReturn() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	ref := p.parseFuncTypePrefix()
	return ref != nil && ref.FuncReturns != nil
}

func (p *Parser) isPrimitiveType() bool {
	return p.current.Type >= lexer.TOKEN_INT && p.current.Type <= lexer.TOKEN_STRING
}

func (p *Parser) isStartType() bool {
	return p.isPrimitiveType() || p.current.Type == lexer.TOKEN_IDENT || p.current.Type == lexer.TOKEN_LPAREN
}

func (p *Parser) isPointerType() bool {
	return p.current.Type == lexer.TOKEN_STAR || p.isStartType()
}

func (p *Parser) parsePointerDepth() int {
	depth := 0
	for p.current.Type == lexer.TOKEN_STAR {
		depth++
		p.advance()
	}
	return depth
}
