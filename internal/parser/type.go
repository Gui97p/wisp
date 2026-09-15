package parser

import (
	"strconv"

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
	ref := p.parseMapPrefix()

	if !ref.IsMap {
		ref.PointerDepth = p.parsePointerDepth()
		ref.Name = p.current.Literal
	}

	return ref
}

func (p *Parser) parseTypePrefix() *ast.TypeRef {
	ref := p.parseMapPrefix()

	if !ref.IsMap {
		ref.Name = p.current.Literal
		p.advance()
		ref.PointerDepth = p.parsePointerDepth()
	} else {
		p.advance()
	}

	return ref
}

func (p *Parser) parseArraySuffix(ref *ast.TypeRef) bool {
	ref.IsArray = false
	ref.IsSpan = false
	ref.ArraySize = 0

	if p.peek.Type != lexer.TOKEN_LBRACKET {
		return true
	}

	p.advance()

	if p.peek.Type == lexer.TOKEN_RBRACKET {
		p.advance()
		ref.IsSpan = true
		return true
	}

	if !p.expect(lexer.TOKEN_INT_LITERAL) {
		return false
	}

	num, err := strconv.ParseInt(p.current.Literal, 10, 64)
	if err != nil {
		return false
	}

	if !p.expect(lexer.TOKEN_RBRACKET) {
		return false
	}

	ref.IsArray = true
	ref.ArraySize = num
	return true
}
