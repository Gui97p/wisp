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
		stmt := p.parseVarStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_IDENT:
		if p.peek.Type == lexer.TOKEN_IDENT {
			stmt := p.parseVarStatement()
			if stmt == nil {
				return nil
			}
			return stmt
		}
		stmt := p.parseSimpleStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_RETURN:
		stmt := p.parseReturnStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_IF:
		stmt := p.parseIfStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_FOR:
		stmt := p.parseForStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_LOOP:
		stmt := p.parseLoopStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_BREAK:
		stmt := p.parseBreakStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_CONTINUE:
		stmt := p.parseContinueStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_COLON:
		stmt := p.parseLabeledStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	default:
		stmt := p.parseSimpleStatement()
		if stmt == nil {
			return nil
		}
		return stmt
	}
}

func (p *Parser) parseSimpleStatement() ast.Statement {
	expr := p.parseExpression()

	switch p.peek.Type {
	case lexer.TOKEN_ASSIGN,
		lexer.TOKEN_PLUS_ASSIGN,
		lexer.TOKEN_MINUS_ASSIGN,
		lexer.TOKEN_STAR_ASSIGN,
		lexer.TOKEN_SLASH_ASSIGN,
		lexer.TOKEN_PERCENT_ASSIGN:
		stmt := p.parseAssignStatement(expr)
		if stmt == nil {
			return nil
		}
		return stmt
	case lexer.TOKEN_INCREMENT, lexer.TOKEN_DECREMENT:
		stmt := p.parseIncDecStatement(expr)
		if stmt == nil {
			return nil
		}
		return stmt
	default:
		stmt := p.finishExpressionStatement(expr)
		if stmt == nil {
			return nil
		}
		return stmt
	}
}

func (p *Parser) finishExpressionStatement(expr ast.Expression) ast.Statement {
	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		p.advance()
	} else if p.peek.Type != lexer.TOKEN_RBRACE {
		p.errorExpected(lexer.TOKEN_SEMICOLON, p.peek.Type)
		return nil
	}

	return &ast.ExpressionStmt{Expr: expr}
}

func (p *Parser) parseVarStatement() ast.Statement {
	stmt := &ast.VarStmt{}
	if p.current.Type != lexer.TOKEN_LET {
		name, depth := p.parseTypePrefix()
		stmt.Type.Name = name
		stmt.Type.PointerDepth = depth
	} else {
		p.advance()
		if p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_COMMA {
			return p.parseMultiVarStatement()
		}
	}

	if p.current.Type != lexer.TOKEN_IDENT {
		return nil
	}
	stmt.Name = p.current.Literal

	isArray, size, ok := p.parseArraySuffix()
	if !ok {
		return nil
	}
	stmt.Type.IsArray = isArray
	stmt.Type.ArraySize = size

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

func (p *Parser) parseMultiVarStatement() ast.Statement {
	stmt := &ast.MultiVarStmt{}
	stmt.Names = append(stmt.Names, p.current.Literal)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil
		}
		stmt.Names = append(stmt.Names, p.current.Literal)
	}

	if !p.expect(lexer.TOKEN_ASSIGN) {
		return nil
	}
	p.advance()
	stmt.Values = append(stmt.Values, p.parseExpression())

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		stmt.Values = append(stmt.Values, p.parseExpression())
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseAssignStatement(target ast.Expression) ast.Statement {
	stmt := &ast.AssignStmt{Target: target}

	p.advance()
	stmt.Op = p.current.Literal

	p.advance()
	stmt.Value = p.parseExpression()

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStmt{}

	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		p.advance()
		return stmt
	}

	for {
		p.advance()

		expr := p.parseExpression()
		if expr != nil {
			stmt.Values = append(stmt.Values, expr)
		}

		if p.peek.Type != lexer.TOKEN_COMMA {
			break
		}
		p.advance()
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseIfStatement() ast.Statement {
	stmt := &ast.IfStmt{}

	p.advance()
	stmt.Condition = p.parseExpression()

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Then = p.parseBlockStatement()

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_ELSE {
		p.advance()

		switch p.peek.Type {
		case lexer.TOKEN_LBRACE:
			p.advance()
			stmt.Else = p.parseBlockStatement()
			if !p.expect(lexer.TOKEN_RBRACE) {
				return nil
			}
		case lexer.TOKEN_IF:
			p.advance()
			stmt.Else = p.parseIfStatement()
		default:
			p.error("expected statement after else")
			return nil
		}
	}

	return stmt
}

func (p *Parser) parseIncDecStatement(target ast.Expression) ast.Statement {
	stmt := &ast.IncDecStmt{Target: target}

	p.advance()
	stmt.Op = p.current.Literal

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStmt {
	stmt := &ast.ForStmt{}
	p.advance()

	if p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_ASSIGN {
		stmt.Var = p.current.Literal
		p.advance()
		p.advance()

		stmt.Start = p.parseExpression()
		if !p.expect(lexer.TOKEN_RANGE) {
			return nil
		}
		p.advance()
		stmt.End = p.parseExpression()

		if p.peek.Type == lexer.TOKEN_COLON {
			p.advance()
			p.advance()
			stmt.Step = p.parseExpression()
		}
	} else {
		stmt.End = p.parseExpression()
	}

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return stmt
}

func (p *Parser) parseLoopStatement() *ast.LoopStmt {
	stmt := &ast.LoopStmt{}

	if p.peek.Type != lexer.TOKEN_LBRACE {
		p.advance()
		stmt.Condition = p.parseExpression()
	}

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_UNTIL {
		p.advance()
		p.advance()
		stmt.UntilCondition = p.parseExpression()

		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}
	}

	return stmt
}

func (p *Parser) parseBreakStatement() ast.Statement {
	stmt := &ast.BreakStmt{}

	if p.peek.Type == lexer.TOKEN_IDENT {
		p.advance()
		stmt.Label = p.current.Literal
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseContinueStatement() ast.Statement {
	stmt := &ast.ContinueStmt{}

	if p.peek.Type == lexer.TOKEN_IDENT {
		p.advance()
		stmt.Label = p.current.Literal
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseLabeledStatement() ast.Statement {
	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}

	label := p.current.Literal
	p.advance()

	switch p.current.Type {
	case lexer.TOKEN_FOR:
		stmt := p.parseForStatement()
		stmt.Label = label
		return stmt
	case lexer.TOKEN_LOOP:
		stmt := p.parseLoopStatement()
		stmt.Label = label
		return stmt
	default:
		p.error("expected for or loop after label")
		return nil
	}
}
