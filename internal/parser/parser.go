package parser

import (
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/lexer"
)

type Parser struct {
	l *lexer.Lexer

	current lexer.Token
	peek    lexer.Token

	errors diag.List

	allowStructLiteral bool
	switchNameCount    int
}

func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, allowStructLiteral: true}

	p.advance()
	p.advance()

	return p
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for p.current.Type != lexer.TOKEN_EOF {
		decls := p.parseDeclaration()
		program.Declarations = append(program.Declarations, decls...)
		p.advance()
	}

	return program
}

func (p *Parser) advance() {
	p.current = p.peek
	p.peek = p.l.NextToken()

	if p.peek.Type == lexer.TOKEN_ILLEGAL {
		p.errorf("illegal expression: %s", p.peek.Literal)
	}
}

func (p *Parser) expect(t lexer.TokenType) bool {
	if p.peek.Type != t {
		p.errorExpected(t, p.peek.Type)
		return false
	}

	p.advance()
	return true
}

func (p *Parser) checkReservedName(name string) bool {
	if strings.HasPrefix(name, "__wisp") {
		p.error("declaration cannot start with __wisp prefix")
		return false
	}
	return true
}

func (p *Parser) parseIntLiteral(lit string) (int64, error) {
	charset := "0x0X0b0B0o0O"
	base := 10
	if len(lit) > 2 && strings.Contains(charset, lit[:2]) {
		base = 0
	}
	return strconv.ParseInt(lit, base, 64)
}
