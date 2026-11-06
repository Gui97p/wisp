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

	layers, ok := p.parseTypeLayers()
	if !ok {
		return nil
	}

	if p.current.Type == lexer.TOKEN_LPAREN {
		ref := p.parseFuncTypePrefix()
		if ref == nil {
			return nil
		}
		ref.Layers = layers
		return ref
	}

	ref := &ast.TypeRef{Layers: layers}
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

func (p *Parser) parseTypeLayers() ([]ast.TypeLayer, bool) {
	var layers []ast.TypeLayer

	for {
		switch p.current.Type {
		case lexer.TOKEN_STAR:
			layers = append(layers, ast.TypeLayer{Pointer: true})
			p.advance()
		case lexer.TOKEN_LBRACKET:
			if p.peek.Type == lexer.TOKEN_RBRACKET {
				p.advance()
				p.advance()
				layers = append(layers, ast.TypeLayer{IsSpan: true})
				continue
			}

			p.advance()
			size := p.parseExpression()
			if size == nil {
				return nil, false
			}
			if !p.expect(lexer.TOKEN_RBRACKET) {
				return nil, false
			}
			p.advance()

			if literal, ok := size.(*ast.IntLiteral); ok {
				layers = append(layers, ast.TypeLayer{Size: literal.Value})
				continue
			}
			layers = append(layers, ast.TypeLayer{Expr: size, Text: ast.ExprText(size)})
		default:
			return layers, true
		}
	}
}

func (p *Parser) looksLikeTypeDeclaration() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	if _, ok := p.parseTypeLayers(); !ok {
		return false
	}

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
		if _, _, ok := p.parseReturnSpec(true, false); !ok {
			return false
		}
	}

	return p.peek.Type == lexer.TOKEN_ARROW
}

func (p *Parser) looksLikeFuncTypeReturn(header, extern bool) bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	ref := p.parseFuncTypePrefix()
	if ref == nil || ref.FuncReturns == nil {
		return false
	}
	if !header {
		return true
	}

	switch p.peek.Type {
	case lexer.TOKEN_LBRACE, lexer.TOKEN_ARROW, lexer.TOKEN_AS:
		return true
	case lexer.TOKEN_SEMICOLON:
		return extern
	}
	return false
}

func (p *Parser) isPrimitiveType() bool {
	return p.current.Type >= lexer.TOKEN_INT && p.current.Type <= lexer.TOKEN_STRING
}

func (p *Parser) isStartType() bool {
	return p.isPrimitiveType() || p.current.Type == lexer.TOKEN_IDENT || p.current.Type == lexer.TOKEN_LPAREN || p.current.Type == lexer.TOKEN_LBRACKET
}

func (p *Parser) isPointerType() bool {
	return p.current.Type == lexer.TOKEN_STAR || p.isStartType()
}
