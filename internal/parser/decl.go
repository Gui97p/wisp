package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseDeclaration() ast.Declaration {
	switch p.current.Type {
	case lexer.TOKEN_FUNC:
		return p.parseFuncDeclaration()
	case lexer.TOKEN_STRUCT:
		return p.parseStructDeclaration()
	default:
		p.error(p.current, fmt.Sprintf("expected declaration, got %s", p.current.Type))
		p.advance()
		return nil
	}
}
