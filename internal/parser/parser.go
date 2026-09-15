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

	return p
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for p.current.Type != lexer.TOKEN_EOF {
		decl := p.parseDeclaration()

		if decl != nil {
			program.Declarations = append(program.Declarations, decl)
		}
		p.advance()
	}

	return program
}

func (p *Parser) advance() {
	p.current = p.peek
	p.peek = p.l.NextToken()
}

func (p *Parser) expect(t lexer.TokenType) bool {
	if p.peek.Type != t {
		p.errorExpected(t, p.peek.Type)
		return false
	}

	p.advance()
	return true
}

func (p *Parser) isPrimitiveType() bool {
	return p.current.Type >= lexer.TOKEN_INT && p.current.Type <= lexer.TOKEN_STRING
}

func (p *Parser) isStartType() bool {
	return p.isPrimitiveType() || p.current.Type == lexer.TOKEN_IDENT || p.current.Type == lexer.TOKEN_MAP
}

func (p *Parser) isPointerType() bool {
	return p.current.Type == lexer.TOKEN_STAR || p.isStartType()
}

func (p *Parser) parsePointerDepth() int {
	depth := 0
	for p.current.Type == lexer.TOKEN_STAR {
		depth++
		p.advance()
	}
	return depth
}
