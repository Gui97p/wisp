package parser

import (
	"fmt"
	"strconv"

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

func (p *Parser) ShowErrors() {
	for _, e := range p.errors {
		fmt.Println(e)
	}
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
	return p.isPrimitiveType() || p.current.Type == lexer.TOKEN_IDENT
}

func (p *Parser) parsePointerDepth() int {
	depth := 0
	for p.current.Type == lexer.TOKEN_STAR {
		depth++
		p.advance()
	}
	return depth
}

func (p *Parser) parseTypePrefix() (string, int) {
	name := p.current.Literal
	p.advance()
	return name, p.parsePointerDepth()
}

func (p *Parser) parseArraySuffix() (bool, int64, bool) {
	if p.peek.Type != lexer.TOKEN_LBRACKET {
		return false, 0, true
	}

	p.advance()

	if !p.expect(lexer.TOKEN_INT_LITERAL) {
		return false, 0, false
	}

	num, err := strconv.ParseInt(p.current.Literal, 10, 64)
	if err != nil {
		return false, 0, false
	}

	if !p.expect(lexer.TOKEN_RBRACKET) {
		return false, 0, false
	}

	return true, num, true
}
