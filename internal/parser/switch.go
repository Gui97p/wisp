package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseSwitchHeader(line, col int) (*ast.BlockStmt, ast.Expression, bool) {
	block := &ast.BlockStmt{}

	if p.current.Type == lexer.TOKEN_LBRACE {
		block.SetPos(line, col)
		block.SetEndPos(p.currentEnd())
		return block, nil, false
	}

	typed := false
	if p.current.Type == lexer.TOKEN_TYPE {
		typed = true
		p.advance()
		if p.current.Type == lexer.TOKEN_LET {
			p.error("a match on types does not name the value, name it in each arm: int v => ...")
			return nil, nil, false
		}
	}

	name := fmt.Sprintf("__wisp_switch_%d", p.switchNameCount)
	var nameLine, nameCol, nameEndLine, nameEndCol int
	if p.current.Type == lexer.TOKEN_LET {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil, nil, false
		}
		name = p.current.Literal
		nameLine, nameCol, nameEndLine, nameEndCol = p.currentIdentPos()

		if !p.checkReservedName(name) {
			return nil, nil, false
		}

		if !p.expect(lexer.TOKEN_ASSIGN) {
			return nil, nil, false
		}
		p.advance()
	} else {
		p.switchNameCount++
	}

	last := p.allowStructLiteral
	p.allowStructLiteral = false
	value := p.parseExpression()
	p.allowStructLiteral = last
	if value == nil {
		return nil, nil, false
	}

	varStmt := &ast.VarStmt{}
	varStmt.Vars = append(varStmt.Vars, ast.Param{Name: name, Line: nameLine, Col: nameCol, EndLine: nameEndLine, EndCol: nameEndCol})
	varStmt.Values = append(varStmt.Values, value)
	varStmt.SetPos(line, col)
	varStmt.SetEndPos(p.currentEnd())

	block.Statements = append(block.Statements, varStmt)
	block.SetPos(line, col)
	block.SetEndPos(p.currentEnd())

	ident := &ast.IdentLiteral{Value: name, Synthetic: true}
	ident.SetPos(line, col)
	ident.SetEndPos(p.currentEnd())

	return block, ident, typed
}

func (p *Parser) startsTypeLabel(typed bool) bool {
	if p.current.Type == lexer.TOKEN_STAR || p.current.Type == lexer.TOKEN_QUESTION_MARK {
		return true
	}
	if typed && p.current.Type == lexer.TOKEN_IDENT && p.peek.Type == lexer.TOKEN_LT {
		return true
	}
	if !p.isPrimitiveType() {
		return false
	}
	if p.peek.Type == lexer.TOKEN_ARROW || p.peek.Type == lexer.TOKEN_COMMA {
		return true
	}
	return typed && p.peek.Type == lexer.TOKEN_IDENT
}

func (p *Parser) parseTypeBind(typed bool) (*ast.TypeBind, bool) {
	if !typed || p.peek.Type != lexer.TOKEN_IDENT {
		return nil, true
	}
	p.advance()
	name := p.current.Literal
	line, col, endLine, endCol := p.currentIdentPos()
	if !p.checkReservedName(name) {
		return nil, false
	}
	return &ast.TypeBind{Name: name, Line: line, Col: col, EndLine: endLine, EndCol: endCol}, true
}

func (p *Parser) atDefaultArm() bool {
	if p.peek.Type != lexer.TOKEN_IDENT || p.peek.Literal != "_" {
		return false
	}
	next := *p.l
	return next.NextToken().Type == lexer.TOKEN_ARROW
}

func (p *Parser) oldArmWord() bool {
	if p.peek.Type != lexer.TOKEN_IDENT {
		return false
	}
	next := *p.l
	following := next.NextToken().Type
	switch p.peek.Literal {
	case "case":
		return following != lexer.TOKEN_ARROW && following != lexer.TOKEN_COMMA
	case "default":
		return following == lexer.TOKEN_ARROW
	}
	return false
}

func (p *Parser) rejectOldArm() {
	p.advance()
	if p.current.Literal == "default" {
		p.errorf("the last arm is written `_ => ...`: `default` is gone")
		return
	}
	p.errorf("an arm is written `value => statement;`: `case` is gone")
}

func (p *Parser) parseOneCaseExpression(value ast.Expression, line, col int, typed, first bool) ast.Expression {
	p.advance()
	if !first {
		p.advance()
	}

	if value == nil {
		expr := p.parseExpressionPratt(RANGE)
		if expr != nil && (p.peek.Type == lexer.TOKEN_RANGE || p.peek.Type == lexer.TOKEN_RANGE_OPEN) {
			p.errorf("a range needs a match value to compare against")
			return nil
		}
		return expr
	}

	if p.startsTypeLabel(typed) {
		labelLine, labelCol := p.current.Line, p.current.Column
		ref := p.parseTypeDefinitionPrefix()
		if ref == nil {
			return nil
		}

		label := &ast.TypeLabel{Type: *ref}
		label.SetPos(labelLine, labelCol)
		label.SetEndPos(p.currentEnd())

		bind, ok := p.parseTypeBind(typed)
		if !ok {
			return nil
		}

		match := &ast.BinaryExpr{Left: value, Operator: "==", Right: label, Synthetic: true, TypeTest: typed, Bind: bind}
		match.SetPos(line, col)
		match.SetEndPos(p.currentEnd())
		return match
	}

	expr := p.parseExpressionPratt(RANGE)
	if expr == nil {
		return nil
	}

	if !typed && p.peek.Type == lexer.TOKEN_IDENT && isVariantName(expr) {
		bind, ok := p.parseTypeBind(true)
		if !ok {
			return nil
		}
		match := &ast.BinaryExpr{Left: value, Operator: "==", Right: expr, Synthetic: true, Bind: bind}
		match.SetPos(line, col)
		match.SetEndPos(p.currentEnd())
		return match
	}

	if typed {
		bind, ok := p.parseTypeBind(typed)
		if !ok {
			return nil
		}
		match := &ast.BinaryExpr{Left: value, Operator: "==", Right: expr, Synthetic: true, TypeTest: true, Bind: bind}
		match.SetPos(line, col)
		match.SetEndPos(p.currentEnd())
		return match
	}

	if p.peek.Type == lexer.TOKEN_RANGE || p.peek.Type == lexer.TOKEN_RANGE_OPEN {
		upperOperator := "<="
		if p.peek.Type == lexer.TOKEN_RANGE_OPEN {
			upperOperator = "<"
		}
		p.advance()
		p.advance()

		end := p.parseExpressionPratt(RANGE)
		if end == nil {
			return nil
		}

		left := &ast.BinaryExpr{
			Synthetic: true,
			Left:      value,
			Operator:  ">=",
			Right:     expr,
		}
		left.SetPos(line, col)
		left.SetEndPos(p.currentEnd())
		right := &ast.BinaryExpr{
			Synthetic: true,
			Left:      value,
			Operator:  upperOperator,
			Right:     end,
		}
		right.SetPos(line, col)
		right.SetEndPos(p.currentEnd())

		expr = &ast.BinaryExpr{
			Synthetic: true,
			Left:      left,
			Operator:  "&&",
			Right:     right,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
		expr.(*ast.BinaryExpr).SetEndPos(p.currentEnd())
	} else {
		expr = &ast.BinaryExpr{
			Synthetic: true,
			Left:      value,
			Operator:  "==",
			Right:     expr,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
		expr.(*ast.BinaryExpr).SetEndPos(p.currentEnd())
	}

	return expr
}

func (p *Parser) parseCaseExpression(value ast.Expression, line, col int, typed bool) (ast.Expression, []ast.Expression) {
	expr := p.parseOneCaseExpression(value, line, col, typed, true)
	if expr == nil {
		return nil, nil
	}
	matches := []ast.Expression{expr}
	binds := 0
	if m, ok := expr.(*ast.BinaryExpr); ok && m.Bind != nil {
		binds++
	}

	for p.peek.Type == lexer.TOKEN_COMMA {
		right := p.parseOneCaseExpression(value, line, col, typed, false)
		if right == nil {
			return nil, nil
		}
		matches = append(matches, right)
		if m, ok := right.(*ast.BinaryExpr); ok && m.Bind != nil {
			binds++
		}
	}

	if len(matches) > 1 && binds > 0 {
		last, lastBinds := matches[len(matches)-1].(*ast.BinaryExpr)
		if !typed || binds != 1 || !lastBinds || last.Bind == nil {
			if typed {
				p.error("an arm with several types names the value once, after the last type: *A, *B v => ...")
			} else {
				p.error("an arm with several variants cannot name the data")
			}
			return nil, nil
		}
		for i, m := range matches[:len(matches)-1] {
			alt, ok := m.(*ast.BinaryExpr)
			if !ok {
				p.error("an arm with several types names the value once, after the last type: *A, *B v => ...")
				return nil, nil
			}
			bind := *last.Bind
			alt.Bind = &bind
			matches[i] = alt
		}
		for _, m := range matches {
			m.(*ast.BinaryExpr).Bind.Shared = true
		}
		return matches[0], matches[1:]
	}

	expr = matches[0]
	for _, right := range matches[1:] {
		expr = &ast.BinaryExpr{
			Synthetic: true,
			Left:      expr,
			Operator:  "||",
			Right:     right,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
		expr.(*ast.BinaryExpr).SetEndPos(p.currentEnd())
	}
	return expr, nil
}

func isVariantName(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.DotIdent:
		return true
	case *ast.MemberExpr:
		_, ok := e.Object.(*ast.IdentLiteral)
		return ok
	}
	return false
}

func (p *Parser) parseEachStatement() ast.Statement {
	line, col := p.current.Line, p.current.Column
	p.advance()
	if p.current.Type == lexer.TOKEN_TYPE {
		p.errorf("each runs every arm that holds, and a value has only one type: use match type here")
		return nil
	}
	block, value, _ := p.parseSwitchHeader(line, col)
	if block == nil {
		return nil
	}
	if value != nil && !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	synthetic := func(node ast.Node) {
		node.SetPos(line, col)
		node.SetEndPos(line, col)
	}
	name := fmt.Sprintf("__wisp_each_%d", p.switchNameCount)
	p.switchNameCount++
	flag := func() *ast.IdentLiteral {
		id := &ast.IdentLiteral{Value: name, Synthetic: true}
		synthetic(id)
		return id
	}

	var arms []*ast.IfStmt
	for p.peek.Type != lexer.TOKEN_RBRACE && p.peek.Type != lexer.TOKEN_EOF && !p.atDefaultArm() {
		if p.oldArmWord() {
			p.rejectOldArm()
			return nil
		}
		armLine, armCol := p.peek.Line, p.peek.Column
		condition, _ := p.parseCaseExpression(value, line, col, false)
		if condition == nil {
			return nil
		}
		stmts, ok := p.parseSwitchArmBody()
		if !ok {
			return nil
		}
		arm := &ast.IfStmt{Condition: condition, Then: &ast.BlockStmt{Statements: stmts}}
		arm.SetPos(armLine, armCol)
		arm.SetEndPos(p.currentEnd())
		arm.Then.SetPos(armLine, armCol)
		arm.Then.SetEndPos(p.currentEnd())
		arms = append(arms, arm)
	}

	var fallback *ast.IfStmt
	if p.atDefaultArm() {
		p.advance()
		armLine, armCol := p.current.Line, p.current.Column
		stmts, ok := p.parseSwitchArmBody()
		if !ok {
			return nil
		}
		none := &ast.UnaryExpr{Operator: "!", Value: flag()}
		synthetic(none)
		fallback = &ast.IfStmt{Condition: none, Then: &ast.BlockStmt{Statements: stmts}}
		fallback.SetPos(armLine, armCol)
		fallback.SetEndPos(p.currentEnd())
		fallback.Then.SetPos(armLine, armCol)
		fallback.Then.SetEndPos(p.currentEnd())
		if p.peek.Type != lexer.TOKEN_RBRACE {
			p.errorfNoSync("`_` is the last arm of an each: nothing can come after it")
		}
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	body := &ast.BlockStmt{}
	if fallback != nil {
		unset := &ast.BoolLiteral{Value: false}
		synthetic(unset)
		declare := &ast.VarStmt{Vars: []ast.Param{{Name: name, Line: line, Col: col, EndLine: line, EndCol: col}}, Values: []ast.Expression{unset}}
		synthetic(declare)
		body.Statements = append(body.Statements, declare)
		for _, arm := range arms {
			set := &ast.BoolLiteral{Value: true}
			synthetic(set)
			mark := &ast.AssignStmt{Targets: []ast.Expression{flag()}, Op: "=", Values: []ast.Expression{set}}
			synthetic(mark)
			arm.Then.Statements = append([]ast.Statement{mark}, arm.Then.Statements...)
		}
	}
	for _, arm := range arms {
		body.Statements = append(body.Statements, arm)
	}
	if fallback != nil {
		body.Statements = append(body.Statements, fallback)
	}
	synthetic(body)
	body.SetEndPos(p.currentEnd())

	always := &ast.BoolLiteral{Value: true}
	synthetic(always)
	wrapper := &ast.IfStmt{Condition: always, Then: body, IsSwitch: true, SwitchDefault: true}
	wrapper.SetPos(line, col)
	wrapper.SetEndPos(p.currentEnd())

	block.Statements = append(block.Statements, wrapper)
	return block
}
