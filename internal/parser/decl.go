package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseDeclaration() ast.Declaration {
	switch p.current.Type {
	case lexer.TOKEN_FUNC:
		decl := p.parseFuncDeclaration()
		if decl == nil {
			return nil
		}
		return decl
	case lexer.TOKEN_STRUCT:
		decl := p.parseStructDeclaration()
		if decl == nil {
			return nil
		}
		return decl
	default:
		p.error(fmt.Sprintf("expected declaration, got %s", p.current.Type))
		p.advance()
		return nil
	}
}
