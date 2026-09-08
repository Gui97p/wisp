package parser

import (
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
		p.errors = append(p.errors, "expected declaration")
		p.advance()
		return nil
	}
}
