package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseBlockStatement() *ast.BlockStmt {
	block := &ast.BlockStmt{}

	for p.peek.Type != lexer.TOKEN_RBRACE && p.current.Type != lexer.TOKEN_EOF {
		p.advance()

		stmt := p.parseStatement()

		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
	}

	return block
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.current.Type {
	case lexer.TOKEN_LET,
		lexer.TOKEN_INT,
		lexer.TOKEN_INT8,
		lexer.TOKEN_INT16,
		lexer.TOKEN_INT32,
		lexer.TOKEN_INT64,
		lexer.TOKEN_UINT,
		lexer.TOKEN_UINT8,
		lexer.TOKEN_UINT16,
		lexer.TOKEN_UINT32,
		lexer.TOKEN_UINT64,
		lexer.TOKEN_FLOAT32,
		lexer.TOKEN_FLOAT64,
		lexer.TOKEN_STRING,
		lexer.TOKEN_BOOL,
		lexer.TOKEN_CHAR:
		return p.parseVarStatement()
	case lexer.TOKEN_IDENT:
		switch p.peek.Type {
		case lexer.TOKEN_IDENT:
			return p.parseVarStatement()
		case lexer.TOKEN_ASSIGN,
			lexer.TOKEN_PLUS_ASSIGN,
			lexer.TOKEN_MINUS_ASSIGN,
			lexer.TOKEN_STAR_ASSIGN,
			lexer.TOKEN_SLASH_ASSIGN,
			lexer.TOKEN_PERCENT_ASSIGN:
			return p.parseAssignStatement()
		default:
			return p.parseExpressionStatement()
		}
	// case lexer.TOKEN_RETURN:
	// 	return p.parseReturnStatement()
	// case lexer.TOKEN_IF:
	// 	return p.parseIfStatement()
	// case lexer.TOKEN_FOR:
	// 	return p.parseForStatement()
	// case lexer.TOKEN_LOOP:
	// 	return p.parseLoopStatement()
	// case lexer.TOKEN_BREAK:
	// 	return p.parseBreakStatement()
	// case lexer.TOKEN_CONTINUE:
	// 	return p.parseContinueStatement()
	// case lexer.TOKEN_COLON:
	// 	return p.parseLabeledStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	return &ast.ExpressionStmt{
		Expr: p.parseExpression(),
	}
}

func (p *Parser) parseVarStatement() ast.Statement {
	stmt := &ast.VarStmt{IsPointer: false}
	if p.current.Type != lexer.TOKEN_LET {
		stmt.Type = p.current.Literal
	}

	if p.peek.Type == lexer.TOKEN_STAR {
		p.advance()
		stmt.IsPointer = true
	}

	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}
	stmt.Name = p.current.Literal

	if !p.expect(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.advance()
	stmt.Value = p.parseExpression()

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseAssignStatement() ast.Statement {
	stmt := &ast.AssignStmt{}
	stmt.Name = p.current.Literal

	p.advance()
	stmt.Op = p.current.Literal

	p.advance()
	stmt.Value = p.parseExpression()

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}
