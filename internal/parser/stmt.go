package parser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseStatement() ast.Statement {
	line, col := p.current.Line, p.current.Column

	stmt := p.parseStatementInner()
	if stmt == nil {
		return nil
	}

	stmt.SetPos(line, col)
	return stmt
}

func (p *Parser) parseStatementInner() ast.Statement {
	var stmt ast.Statement

	switch p.current.Type {
	case lexer.TOKEN_CONST:
		stmt = p.parseConstStatement()
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
		stmt = p.parseVarStatement()
	case lexer.TOKEN_IDENT, lexer.TOKEN_STAR:
		if p.looksLikeTypeDeclaration() {
			stmt = p.parseVarStatement()
		} else {
			stmt = p.parseSimpleStatement()
		}
	case lexer.TOKEN_MAP:
		stmt = p.parseVarStatement()
	case lexer.TOKEN_RETURN:
		stmt = p.parseReturnStatement()
	case lexer.TOKEN_SWITCH:
		stmt = p.parseSwitchStatement()
	case lexer.TOKEN_IF:
		stmt = p.parseIfStatement()
	case lexer.TOKEN_FOR:
		forStmt := p.parseForStatement()
		if forStmt == nil {
			return nil
		}
		stmt = forStmt
	case lexer.TOKEN_LOOP:
		loopStmt := p.parseLoopStatement()
		if loopStmt == nil {
			return nil
		}
		stmt = loopStmt
	case lexer.TOKEN_BREAK:
		stmt = p.parseBreakStatement()
	case lexer.TOKEN_CONTINUE:
		stmt = p.parseContinueStatement()
	case lexer.TOKEN_COLON:
		stmt = p.parseLabeledStatement()
	default:
		stmt = p.parseSimpleStatement()
	}

	if stmt == nil {
		return nil
	}
	return stmt
}

func (p *Parser) parseConstStatement() ast.Statement {
	stmt := &ast.ConstStmt{}

	vars := p.parseConstIdentifiers()
	if vars == nil {
		return nil
	}
	stmt.Vars = vars

	p.advance()

	if p.current.Type == lexer.TOKEN_SWITCH {
		group, value := p.parseSwitchExpr()
		if value == nil {
			return nil
		}
		stmt.Values = append(stmt.Values, value)

		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}

		group.Statements = append(group.Statements, stmt)
		return group
	}

	values := p.parseConstValues()
	if values == nil {
		return nil
	}
	stmt.Values = values

	return stmt
}

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

func (p *Parser) parseSimpleStatement() ast.Statement {
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

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

	var currentType ast.TypeRef
	if p.current.Type != lexer.TOKEN_LET {
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil
		}
		if !p.parseArraySuffix(ref) {
			return nil
		}
		p.advance()
		currentType = *ref
	} else {
		p.advance()
	}

	if p.current.Type != lexer.TOKEN_IDENT {
		p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
		return nil
	}
	name := p.current.Literal

	if !p.checkReservedName(name) {
		return nil
	}

	stmt.Vars = append(stmt.Vars, ast.Param{Name: name, Type: currentType})

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil
		}
		name := p.current.Literal

		if !p.checkReservedName(name) {
			return nil
		}

		stmt.Vars = append(stmt.Vars, ast.Param{Name: name, Type: currentType})
	}

	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		if stmt.Vars[0].Type.Name == "" {
			p.error("variable declaration needs a type or a value")
			return nil
		}
		p.advance()
		return stmt
	}

	if !p.expect(lexer.TOKEN_ASSIGN) {
		return nil
	}

	p.advance()
	if p.current.Type == lexer.TOKEN_SWITCH {
		group, value := p.parseSwitchExpr()
		if value == nil {
			return nil
		}
		stmt.Values = append(stmt.Values, value)

		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}

		group.Statements = append(group.Statements, stmt)
		return group
	}

	value := p.parseExpression()
	if value == nil {
		return nil
	}
	stmt.Values = append(stmt.Values, value)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		value := p.parseExpression()
		if value == nil {
			return nil
		}
		stmt.Values = append(stmt.Values, value)
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

	if p.current.Type == lexer.TOKEN_SWITCH {
		group, value := p.parseSwitchExpr()
		if value == nil {
			return nil
		}
		stmt.Value = value

		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}

		group.Statements = append(group.Statements, stmt)
		return group
	}

	stmt.Value = p.parseExpression()
	if stmt.Value == nil {
		return nil
	}

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

	if p.peek.Type == lexer.TOKEN_SWITCH {
		p.advance()
		group, value := p.parseSwitchExpr()
		if value == nil {
			return nil
		}
		stmt.Values = append(stmt.Values, value)

		if !p.expect(lexer.TOKEN_SEMICOLON) {
			return nil
		}

		group.Statements = append(group.Statements, stmt)
		return group
	}

	for {
		p.advance()

		expr := p.parseExpression()
		if expr == nil {
			return nil
		}
		stmt.Values = append(stmt.Values, expr)

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

func (p *Parser) parseSwitchStatement() ast.Statement {
	line, col := p.current.Line, p.current.Column
	p.advance()
	block, value := p.parseSwitchHeader(line, col)
	if block == nil || value == nil {
		return nil
	}

	if !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	first := true
	stmt := &ast.IfStmt{Then: &ast.BlockStmt{}}
	stmt.SetPos(line, col)
	currentIf := stmt

	for p.peek.Type == lexer.TOKEN_CASE {
		if !first {
			newIf := &ast.IfStmt{Then: &ast.BlockStmt{}}
			newIf.SetPos(line, col)
			currentIf.Else = &ast.BlockStmt{Statements: []ast.Statement{newIf}}
			currentIf = newIf
		} else {
			first = false
		}

		expr := p.parseCaseExpression(value, line, col)

		currentIf.Condition = expr

		if !p.expect(lexer.TOKEN_COLON) {
			return nil
		}

		stmts := p.parseSwitchBlock()
		if stmts != nil {
			currentIf.Then.Statements = stmts
		}
	}

	if p.peek.Type == lexer.TOKEN_DEFAULT {
		p.advance()
		if !p.expect(lexer.TOKEN_COLON) {
			return nil
		}

		stmts := p.parseSwitchBlock()
		if stmts != nil {
			currentIf.Else = &ast.BlockStmt{Statements: stmts}
		}
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	block.Statements = append(block.Statements, stmt)

	return block
}

func (p *Parser) parseIfStatement() ast.Statement {
	stmt := &ast.IfStmt{}

	p.advance()
	prev := p.allowStructLiteral
	p.allowStructLiteral = false
	stmt.Condition = p.parseExpression()
	p.allowStructLiteral = prev

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
			line, col := p.current.Line, p.current.Column
			stmt.Else = p.parseIfStatement()
			if stmt.Else != nil {
				stmt.Else.SetPos(line, col)
			}
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

	switch {
	case p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_COMMA:
		stmt.Var = p.current.Literal

		if !p.checkReservedName(stmt.Var) {
			return nil
		}

		p.advance()

		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		stmt.Var2 = p.current.Literal

		if !p.checkReservedName(stmt.Var2) {
			return nil
		}

		if !p.expect(lexer.TOKEN_IN) {
			return nil
		}
		p.advance()
		prev := p.allowStructLiteral
		p.allowStructLiteral = false
		stmt.Range = p.parseExpression()
		p.allowStructLiteral = prev

	case p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_IN:
		stmt.Var = p.current.Literal

		if !p.checkReservedName(stmt.Var) {
			return nil
		}

		p.advance()
		p.advance()

		prev := p.allowStructLiteral
		p.allowStructLiteral = false
		expr := p.parseExpression()
		p.allowStructLiteral = prev

		if p.peek.Type == lexer.TOKEN_RANGE {
			stmt.Start = expr
			p.advance()
			p.advance()
			prev := p.allowStructLiteral
			p.allowStructLiteral = false
			stmt.End = p.parseExpression()
			p.allowStructLiteral = prev

			if p.peek.Type == lexer.TOKEN_COLON {
				p.advance()
				p.advance()
				prev := p.allowStructLiteral
				p.allowStructLiteral = false
				stmt.Step = p.parseExpression()
				p.allowStructLiteral = prev
			}
		} else {
			stmt.Range = expr
		}

	default:
		prev := p.allowStructLiteral
		p.allowStructLiteral = false
		stmt.End = p.parseExpression()
		p.allowStructLiteral = prev
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
		prev := p.allowStructLiteral
		p.allowStructLiteral = false
		stmt.Condition = p.parseExpression()
		p.allowStructLiteral = prev
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
		prev := p.allowStructLiteral
		p.allowStructLiteral = false
		stmt.UntilCondition = p.parseExpression()
		p.allowStructLiteral = prev
		if stmt.UntilCondition == nil {
			return nil
		}

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

	if !p.checkReservedName(label) {
		return nil
	}

	p.advance()

	switch p.current.Type {
	case lexer.TOKEN_FOR:
		stmt := p.parseForStatement()
		if stmt == nil {
			return nil
		}
		stmt.Label = label
		return stmt
	case lexer.TOKEN_LOOP:
		stmt := p.parseLoopStatement()
		if stmt == nil {
			return nil
		}
		stmt.Label = label
		return stmt
	default:
		p.error("expected for or loop after label")
		return nil
	}
}
