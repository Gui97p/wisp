package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) error(e string) {
	p.errors = append(p.errors, fmt.Sprintf("[%d:%d] %s", p.current.Line, p.current.Column, e))
}

func (p *Parser) errorExpected(expected, got lexer.TokenType) {
	p.error(fmt.Sprintf("expected '%s' got '%s'", expected, got))
}

func (p *Parser) errorType(t lexer.TokenType) {
	p.error(fmt.Sprintf("invalid type: %s", t))
}
