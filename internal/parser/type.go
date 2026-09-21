package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseMapPrefix() *ast.TypeRef {
	ref := &ast.TypeRef{}
	if p.current.Type == lexer.TOKEN_MAP {
		ref.Name = "map"
		ref.IsMap = true
		if !p.expect(lexer.TOKEN_LBRACKET) {
			return nil
		}
		p.advance()

		if !p.isPointerType() {
			p.errorType(p.current.Type)
			return nil
		}
		ref.MapKey = p.parseTypeDefinitionPrefix()

		if !p.expect(lexer.TOKEN_RBRACKET) {
			return nil
		}
		p.advance()

		if !p.isPointerType() {
			p.errorType(p.current.Type)
			return nil
		}
		ref.MapValue = p.parseTypeDefinitionPrefix()
	}

	return ref
}

func (p *Parser) parseTypeDefinitionPrefix() *ast.TypeRef {
	fallible := p.current.Type == lexer.TOKEN_NOT
	if fallible {
		p.advance()
	}

	ref := p.parseMapPrefix()
	ref.Fallible = fallible

	if !ref.IsMap {
		ref.PointerDepth = p.parsePointerDepth()
		ref.Name = p.current.Literal
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

		if !p.expect(lexer.TOKEN_INT_LITERAL) {
			return false
		}

		num, err := p.parseIntLiteral(p.current.Literal)
		if err != nil {
			return false
		}

		if !p.expect(lexer.TOKEN_RBRACKET) {
			return false
		}

		ref.Dims = append(ref.Dims, ast.ArrayDim{Size: num})
	}

	return true
}

func (p *Parser) looksLikeTypeDeclaration() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	p.parsePointerDepth()

	switch {
	case p.current.Type == lexer.TOKEN_MAP, p.isPrimitiveType():
		return true
	case p.current.Type == lexer.TOKEN_IDENT:
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
