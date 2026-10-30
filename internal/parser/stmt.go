package parser

import (
	"slices"

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
	stmt.SetEndPos(p.currentEnd())
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
		lexer.TOKEN_FLOAT,
		lexer.TOKEN_FLOAT32,
		lexer.TOKEN_FLOAT64,
		lexer.TOKEN_STRING,
		lexer.TOKEN_BOOL,
		lexer.TOKEN_CHAR:
		stmt = p.parseVarStatement()
	case lexer.TOKEN_IDENT, lexer.TOKEN_STAR, lexer.TOKEN_LPAREN:
		if p.looksLikeTypeDeclaration() {
			stmt = p.parseVarStatement()
		} else {
			stmt = p.parseSimpleStatement()
		}
	case lexer.TOKEN_RETURN:
		stmt = p.parseReturnStatement()
	case lexer.TOKEN_YIELD:
		stmt = p.parseYieldStatement()
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
	case lexer.TOKEN_NATIVE:
		stmt = p.parseNativeStatement()
	case lexer.TOKEN_DEFER:
		stmt = p.parseDeferStatement()
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

	values := p.parseConstValues()
	if values == nil {
		return nil
	}
	stmt.Values = values

	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStmt {
	block := &ast.BlockStmt{}
	block.SetPos(p.current.Line, p.current.Column)

	for p.peek.Type != lexer.TOKEN_RBRACE && p.current.Type != lexer.TOKEN_EOF {
		p.advance()

		start := p.checkpoint()
		errorsBefore := len(p.errors)
		stmt := p.parseStatement()

		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		} else if len(p.errors) > errorsBefore {
			p.skipBrokenStatement(start)
		}
	}

	block.SetEndPos(p.peek.Line, p.peek.Column)

	return block
}

func (p *Parser) skipBrokenStatement(start parserCheckpoint) {
	errors := slices.Clone(p.errors)
	p.restore(start)
	p.errors = errors

	startLine := p.current.Line
	depth := 0
	for {
		switch p.current.Type {
		case lexer.TOKEN_LBRACE:
			depth++
		case lexer.TOKEN_RBRACE:
			depth--
		}

		if depth <= 0 {
			if p.current.Type == lexer.TOKEN_SEMICOLON {
				return
			}
			if p.peek.Type == lexer.TOKEN_RBRACE || p.peek.Type == lexer.TOKEN_EOF {
				return
			}
			if p.peek.Line > startLine && startsStatement(p.peek.Type) {
				return
			}
		}
		if p.peek.Type == lexer.TOKEN_EOF {
			return
		}
		p.advance()
	}
}

func startsStatement(t lexer.TokenType) bool {
	if t >= lexer.TOKEN_INT && t <= lexer.TOKEN_STRING {
		return true
	}
	switch t {
	case lexer.TOKEN_LET, lexer.TOKEN_CONST, lexer.TOKEN_IF, lexer.TOKEN_FOR, lexer.TOKEN_LOOP,
		lexer.TOKEN_RETURN, lexer.TOKEN_BREAK, lexer.TOKEN_CONTINUE, lexer.TOKEN_NATIVE,
		lexer.TOKEN_SWITCH, lexer.TOKEN_DEFER, lexer.TOKEN_YIELD:
		return true
	}
	return false
}

func (p *Parser) parseSimpleStatement() ast.Statement {
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	targets := []ast.Expression{expr}
	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		next := p.parseExpression()
		if next == nil {
			return nil
		}
		targets = append(targets, next)
	}

	if len(targets) > 1 && p.peek.Type != lexer.TOKEN_ASSIGN {
		p.errorExpected(lexer.TOKEN_ASSIGN, p.peek.Type)
		return nil
	}

	switch p.peek.Type {
	case lexer.TOKEN_ASSIGN,
		lexer.TOKEN_PLUS_ASSIGN,
		lexer.TOKEN_MINUS_ASSIGN,
		lexer.TOKEN_STAR_ASSIGN,
		lexer.TOKEN_SLASH_ASSIGN,
		lexer.TOKEN_PERCENT_ASSIGN,
		lexer.TOKEN_AMP_ASSIGN,
		lexer.TOKEN_PIPE_ASSIGN,
		lexer.TOKEN_XOR_ASSIGN,
		lexer.TOKEN_SHIFT_LEFT_ASSIGN,
		lexer.TOKEN_SHIFT_RIGHT_ASSIGN:
		stmt := p.parseAssignStatement(targets)
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
	if !p.expect(lexer.TOKEN_SEMICOLON) {
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
	line, col, endLine, endCol := p.currentIdentPos()

	if !p.checkReservedName(name) {
		return nil
	}

	stmt.Vars = append(stmt.Vars, ast.Param{Name: name, Type: currentType, Line: line, Col: col, EndLine: endLine, EndCol: endCol})

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil
		}
		name := p.current.Literal
		line, col, endLine, endCol := p.currentIdentPos()

		if !p.checkReservedName(name) {
			return nil
		}

		stmt.Vars = append(stmt.Vars, ast.Param{Name: name, Type: currentType, Line: line, Col: col, EndLine: endLine, EndCol: endCol})
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

func (p *Parser) parseAssignStatement(targets []ast.Expression) ast.Statement {
	stmt := &ast.AssignStmt{Targets: targets}

	p.advance()
	stmt.Op = p.current.Literal

	p.advance()

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

	if stmt.Op != "=" && len(stmt.Values) > 1 {
		p.errorf("operator %s accepts a single value", stmt.Op)
		return nil
	}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseYieldStatement() ast.Statement {
	stmt := &ast.YieldStmt{}

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

func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStmt{}

	if p.peek.Type == lexer.TOKEN_SEMICOLON {
		p.advance()
		return stmt
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
	if block == nil {
		return nil
	}

	if value != nil && !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	first := true
	stmt := &ast.IfStmt{Then: &ast.BlockStmt{}, IsSwitch: true}
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
		currentIf.SetEndPos(p.currentEnd())
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
		currentIf.SetEndPos(p.currentEnd())
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	stmt.SetEndPos(p.currentEnd())

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
				stmt.Else.SetEndPos(p.currentEnd())
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
		stmt.VarAt = p.currentPosition()

		if !p.checkReservedName(stmt.Var) {
			return nil
		}

		p.advance()

		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		stmt.Var2 = p.current.Literal
		stmt.Var2At = p.currentPosition()

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
		stmt.VarAt = p.currentPosition()

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
		if stmt.Condition != nil {
			p.error("a loop takes either a condition or an until, not both")
			return nil
		}
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

func (p *Parser) parseDeferStatement() ast.Statement {
	stmt := &ast.DeferStmt{}

	if p.peek.Type == lexer.TOKEN_NOT {
		p.advance()
		stmt.OnError = true

		if p.peek.Type == lexer.TOKEN_PIPE {
			p.advance()
			if !p.expect(lexer.TOKEN_IDENT) {
				return nil
			}
			stmt.ErrorBind = p.current.Literal
			stmt.ErrorBindAt = p.currentPosition()
			if !p.checkReservedName(stmt.ErrorBind) {
				return nil
			}
			if !p.expect(lexer.TOKEN_PIPE) {
				return nil
			}
		}
	}

	if p.peek.Type == lexer.TOKEN_LBRACE {
		p.advance()
		block := p.parseBlockStatement()
		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
		stmt.Body = block
		return stmt
	}

	p.advance()
	body := p.parseStatement()
	if body == nil {
		return nil
	}
	stmt.Body = body
	return stmt
}

func (p *Parser) parseNativeStatement() ast.Statement {
	stmt := &ast.NativeStmt{}

	if p.peek.Type == lexer.TOKEN_IDENT {
		p.advance()
		stmt.Backend = p.current.Literal
		endLine, endCol := p.currentEnd()
		stmt.BackendAt = ast.Position{Line: p.current.Line, Col: p.current.Column, EndLine: endLine, EndCol: endCol}
	}

	if p.peek.Type == lexer.TOKEN_LPAREN {
		p.advance()
		bindings, ok := p.parseNativeBindings()
		if !ok {
			return nil
		}
		stmt.Bindings = bindings
	}

	if !p.expect(lexer.TOKEN_STRING_LITERAL) {
		return nil
	}
	stmt.Code = p.current.Literal
	stmt.Raw = p.current.Raw
	codeEndLine, codeEndCol := p.currentEnd()
	stmt.CodeAt = ast.Position{Line: p.current.Line, Col: p.current.Column, EndLine: codeEndLine, EndCol: codeEndCol}

	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return nil
	}

	return stmt
}

func (p *Parser) parseNativeBindings() ([]ast.NativeBinding, bool) {
	var bindings []ast.NativeBinding

	if p.peek.Type == lexer.TOKEN_RPAREN {
		p.advance()
		return bindings, true
	}

	var in, out, haveMode bool
	for {
		p.advance()

		if mIn, mOut, ok := p.nativeMode(); ok {
			in, out, haveMode = mIn, mOut, true
			p.advance()
		} else if !haveMode {
			p.error("expected in, out or inout")
			return nil, false
		}

		if p.current.Type != lexer.TOKEN_IDENT {
			p.errorExpected(lexer.TOKEN_IDENT, p.current.Type)
			return nil, false
		}
		ident := &ast.IdentLiteral{Value: p.current.Literal}
		ident.SetPos(p.current.Line, p.current.Column)
		ident.SetEndPos(p.currentEnd())
		bindings = append(bindings, ast.NativeBinding{In: in, Out: out, Var: ident})

		if p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			continue
		}
		if !p.expect(lexer.TOKEN_RPAREN) {
			return nil, false
		}
		return bindings, true
	}
}

func (p *Parser) nativeMode() (in, out, ok bool) {
	switch {
	case p.current.Type == lexer.TOKEN_IN:
		return true, false, true
	case p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_IDENT:
		switch p.current.Literal {
		case "out":
			return false, true, true
		case "inout":
			return true, true, true
		}
	}
	return false, false, false
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
