package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

type Parser struct {
	l *lexer.Lexer

	current lexer.Token
	peek    lexer.Token

	errors []string
}

func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	p.advance()
	p.advance()
	p.advance()

	return p
}

func (p *Parser) advance() {
	p.current = p.peek
	p.peek = p.l.NextToken()
}

func (p *Parser) expect(t lexer.TokenType) bool {
	if p.current.Type != t {
		p.errors = append(p.errors, "unexpected token")
		return false
	}

	p.advance()
	return true
}

func (p *Parser) isPrimitiveType() bool {
	switch p.current.Type {
	case lexer.TOKEN_INT_LITERAL, lexer.TOKEN_STRING_LITERAL, lexer.TOKEN_FLOAT_LITERAL, lexer.TOKEN_CHAR_LITERAL, lexer.TOKEN_BOOL:
		return true
	}
	return false
}

func (p *Parser) isStartType() bool {
	return p.isPrimitiveType() || p.current.Type == lexer.TOKEN_IDENT
}

func (p *Parser) parseProgram() *ast.Program {
	program := &ast.Program{}

	for p.current.Type != lexer.TOKEN_EOF {
		decl := p.parseDeclaration()

		if decl != nil {
			program.Declarations = append(program.Declarations, decl)
		}
	}

	return program
}
