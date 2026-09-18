package parser

import "github.com/Gui97p/wisp/internal/lexer"

type parserCheckpoint struct {
	lexer lexer.Lexer

	current lexer.Token
	peek    lexer.Token

	errorsLen int
}

func (p *Parser) checkpoint() parserCheckpoint {
	return parserCheckpoint{lexer: *p.l, current: p.current, peek: p.peek, errorsLen: len(p.errors)}
}

func (p *Parser) restore(cp parserCheckpoint) {
	*p.l = cp.lexer
	p.current = cp.current
	p.peek = cp.peek
	p.errors = p.errors[:cp.errorsLen]
}
