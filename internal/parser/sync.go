package parser

import "github.com/Gui97p/wisp/internal/lexer"

var safeTokens = map[lexer.TokenType]bool{
	lexer.TOKEN_SEMICOLON: true,
	lexer.TOKEN_RBRACE:    true,

	lexer.TOKEN_FUNC:   true,
	lexer.TOKEN_STRUCT: true,

	lexer.TOKEN_LET:    true,
	lexer.TOKEN_IF:     true,
	lexer.TOKEN_FOR:    true,
	lexer.TOKEN_LOOP:   true,
	lexer.TOKEN_RETURN: true,
}

func (p *Parser) synchronize() {
	for p.current.Type != lexer.TOKEN_EOF {
		if _, ok := safeTokens[p.current.Type]; ok {
			return
		}
		p.advance()
	}
}
