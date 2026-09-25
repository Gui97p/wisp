package parser

import (
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) Errors() diag.List {
	return p.errors
}

func (p *Parser) HasErrors() bool {
	return len(p.errors) > 0
}

func (p *Parser) errorfNoSync(format string, args ...any) {
	endLine, endCol := p.currentEnd()
	p.errors.Add("", p.current.Line, p.current.Column, endLine, endCol, format, args...)
}

func (p *Parser) errorf(format string, args ...any) {
	p.errorfNoSync(format, args...)
	p.synchronize()
}

func (p *Parser) error(msg string) {
	p.errorf("%s", msg)
}

func (p *Parser) errorExpected(expected, got lexer.TokenType) {
	if expected == lexer.TOKEN_SEMICOLON {
		p.errorfNoSync("expected '%s' got '%s'", expected.DisplayName(), got.DisplayName())
		return
	}
	p.errorf("expected '%s' got '%s'", expected.DisplayName(), got.DisplayName())
}

func (p *Parser) errorType(t lexer.TokenType) {
	p.errorf("invalid type: %s", t.DisplayName())
}
