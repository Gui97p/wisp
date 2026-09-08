package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) error(current lexer.Token, e string) {
	p.errors = append(p.errors, fmt.Sprintf("[%d:%d] %s", current.Line, current.Column, e))
}

func (p *Parser) errorExpected(expected, got lexer.TokenType, current lexer.Token) {
	p.error(current, fmt.Sprintf("expected '%s' got '%s'", expected, got))
}

func (p *Parser) errorType(t lexer.TokenType, current lexer.Token) {
	p.error(current, fmt.Sprintf("invalid type: %s", t))
}
