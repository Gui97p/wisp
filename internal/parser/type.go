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

func (p *Parser) parseArraySuffix() (bool, int64, bool) {
	if p.peek.Type != lexer.TOKEN_LBRACKET {
		return false, 0, true
	}

	p.advance()

	if !p.expect(lexer.TOKEN_INT_LITERAL) {
		return false, 0, false
	}

	num, err := strconv.ParseInt(p.current.Literal, 10, 64)
	if err != nil {
		return false, 0, false
	}

	if !p.expect(lexer.TOKEN_RBRACKET) {
		return false, 0, false
	}

	return true, num, true
}
