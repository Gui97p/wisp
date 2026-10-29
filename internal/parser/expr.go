package parser

import (
	"math"
	"strconv"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseExpression() ast.Expression {
	return p.parseExpressionPratt(LOWEST)
}

func (p *Parser) parseExpressionPratt(precedence Precedence) ast.Expression {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}

	for p.peek.Type != lexer.TOKEN_EOF && precedence < p.peekPrecedence() {
		line, col := left.Position()

		switch p.peek.Type {
		case lexer.TOKEN_LPAREN:
			p.advance()
			left = p.parseCallExpression(left)
		case lexer.TOKEN_LBRACKET:
			p.advance()
			left = p.parseIndexSliceExpression(left)
		case lexer.TOKEN_LBRACE:
			if !p.allowStructLiteral {
				return left
			}
			p.advance()
			left = p.parseStructLiteral(left)
		default:
			p.advance()
			left = p.parseInfix(left)
		}

		if left == nil {
			return nil
		}
		left.SetPos(line, col)
		left.SetEndPos(p.currentEnd())
	}

	left.SetEndPos(p.currentEnd())
	return left
}

func (p *Parser) parsePrefix() ast.Expression {
	line, col := p.current.Line, p.current.Column

	expr := p.parsePrefixInner()
	if expr == nil {
		return nil
	}
	expr.SetPos(line, col)
	expr.SetEndPos(p.currentEnd())
	return expr
}

func (p *Parser) parsePrefixInner() ast.Expression {
	switch p.current.Type {
	case lexer.TOKEN_SWITCH:
		return p.parseSwitchExpression()
	case lexer.TOKEN_INT_LITERAL:
		u, err := p.parseUintLiteral(p.current.Literal)
		if err != nil {
			p.errorf("integer literal %s exceeds 64 bits", p.current.Literal)
			return nil
		}
		return &ast.IntLiteral{Value: int64(u), Unsigned: u > math.MaxInt64}
	case lexer.TOKEN_STRING_LITERAL:
		return &ast.StringLiteral{Value: p.current.Literal}
	case lexer.TOKEN_CHAR_LITERAL:
		return &ast.CharLiteral{Value: p.current.Literal[0]}
	case lexer.TOKEN_FLOAT_LITERAL:
		f, _ := strconv.ParseFloat(p.current.Literal, 64)
		return &ast.FloatLiteral{Value: f}
	case lexer.TOKEN_TRUE, lexer.TOKEN_FALSE:
		return &ast.BoolLiteral{Value: p.current.Type == lexer.TOKEN_TRUE}
	case lexer.TOKEN_IDENT:
		return &ast.IdentLiteral{Value: p.current.Literal}
	case lexer.TOKEN_INT, lexer.TOKEN_UINT,
		lexer.TOKEN_INT8, lexer.TOKEN_UINT8,
		lexer.TOKEN_INT16, lexer.TOKEN_UINT16,
		lexer.TOKEN_INT32, lexer.TOKEN_UINT32,
		lexer.TOKEN_INT64, lexer.TOKEN_UINT64,
		lexer.TOKEN_FLOAT, lexer.TOKEN_FLOAT32, lexer.TOKEN_FLOAT64,
		lexer.TOKEN_BOOL, lexer.TOKEN_CHAR, lexer.TOKEN_STRING:
		if p.peek.Type == lexer.TOKEN_LPAREN || p.peek.Type == lexer.TOKEN_RPAREN {
			return &ast.IdentLiteral{Value: p.current.Literal}
		}
	case lexer.TOKEN_NULL:
		return &ast.NullLiteral{}
	case lexer.TOKEN_DOT:
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		return &ast.DotIdent{Name: p.current.Literal}
	case lexer.TOKEN_LBRACKET:
		return p.parseArrayLiteral()
	case lexer.TOKEN_LBRACE:
		return p.parseMapLiteral()

	case lexer.TOKEN_MINUS,
		lexer.TOKEN_NOT,
		lexer.TOKEN_AMP,
		lexer.TOKEN_NXOR:
		return p.parseUnaryExpression()

	case lexer.TOKEN_STAR:
		p.error("`*` does not dereference: write `p@` to read through a pointer, or `(*T)(x)` to convert to a pointer")
		return nil

	case lexer.TOKEN_LPAREN:
		if p.looksLikePointerConversion() {
			return p.parsePointerConversion()
		}
		if p.looksLikeLambda() {
			return p.parseFuncLiteral()
		}
		return p.parseGroupedExpression()
	}

	p.error("expected expression")
	return nil
}

func (p *Parser) looksLikePointerConversion() bool {
	cp := p.checkpoint()
	defer p.restore(cp)

	p.advance()
	if p.current.Type != lexer.TOKEN_STAR {
		return false
	}

	ref := p.parseTypeDefinitionPrefix()
	if ref == nil || ref.IsMap || ref.IsFunc || ref.Name == "" {
		return false
	}
	if !p.expect(lexer.TOKEN_RPAREN) {
		return false
	}

	return p.peek.Type == lexer.TOKEN_LPAREN
}

func (p *Parser) parsePointerConversion() ast.Expression {
	p.advance()

	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nil
	}

	if !p.expect(lexer.TOKEN_RPAREN) || !p.expect(lexer.TOKEN_LPAREN) {
		return nil
	}

	args := p.parseCallArgs()
	p.spread = false
	if args == nil {
		return nil
	}

	return &ast.ConvExpr{Type: *ref, Args: args}
}

func (p *Parser) parseFuncLiteral() ast.Expression {
	lit := &ast.FuncLiteral{}
	lit.Params = p.parseFuncParamList()
	if p.cVariadic {
		p.cVariadic = false
		p.error("a variadic without a name is only for extern functions")
		return nil
	}

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	if p.peek.Type != lexer.TOKEN_ARROW {
		p.advance()
		types, fallible, ok := p.parseReturnSpec()
		if !ok {
			return nil
		}
		lit.ReturnTypes = types
		lit.Fallible = fallible
	}

	if !p.expect(lexer.TOKEN_ARROW) {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_LBRACE {
		p.advance()
		lit.Block = p.parseBlockStatement()
		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
	} else {
		line, col := p.current.Line, p.current.Column

		stmt := &ast.ReturnStmt{}
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
		stmt.SetPos(line, col)
		stmt.SetEndPos(p.currentEnd())
		lit.Block = &ast.BlockStmt{Statements: []ast.Statement{stmt}}
	}

	return lit
}

func (p *Parser) parseUnaryExpression() ast.Expression {
	op := p.current.Literal

	p.advance()

	right := p.parseExpressionPratt(PREFIX)
	if right == nil {
		return nil
	}

	return &ast.UnaryExpr{
		Operator: op,
		Value:    right,
	}
}

func (p *Parser) parseInfix(left ast.Expression) ast.Expression {
	switch p.current.Type {
	case lexer.TOKEN_PLUS,
		lexer.TOKEN_MINUS,
		lexer.TOKEN_STAR,
		lexer.TOKEN_SLASH,
		lexer.TOKEN_PERCENT,
		lexer.TOKEN_EQUAL,
		lexer.TOKEN_NOT_EQUAL,
		lexer.TOKEN_LT,
		lexer.TOKEN_LTE,
		lexer.TOKEN_GT,
		lexer.TOKEN_GTE,
		lexer.TOKEN_AND,
		lexer.TOKEN_OR,
		lexer.TOKEN_XOR,
		lexer.TOKEN_SHIFT_LEFT,
		lexer.TOKEN_SHIFT_RIGHT,
		lexer.TOKEN_AMP,
		lexer.TOKEN_PIPE:
		return p.parseBinaryExpression(left)
	case lexer.TOKEN_DOT:
		return p.parseMemberExpression(left)
	case lexer.TOKEN_QUESTION_MARK:
		return p.parseTernaryExpr(left)
	case lexer.TOKEN_COALESCE:
		return p.parseCoalesceExpr(left)
	case lexer.TOKEN_NOT:
		expr := &ast.PropagateExpr{Value: left}
		if p.peek.Type == lexer.TOKEN_NOT {
			p.advance()
			expr.Assert = true
		}
		return expr
	case lexer.TOKEN_AT:
		return &ast.DerefExpr{Value: left}
	case lexer.TOKEN_IN:
		return p.parseInExpr(left)
	}

	return left
}

func (p *Parser) parseBinaryExpression(left ast.Expression) ast.Expression {
	op := p.current.Literal
	precedece := p.currentPrecedence()

	p.advance()

	right := p.parseExpressionPratt(precedece)
	if right == nil {
		return nil
	}

	return &ast.BinaryExpr{
		Left:     left,
		Operator: op,
		Right:    right,
	}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.advance()

	line, col := p.current.Line, p.current.Column

	prev := p.allowStructLiteral
	p.allowStructLiteral = true
	expr := p.parseExpression()
	if expr != nil && p.peek.Type == lexer.TOKEN_COMMA {
		tuple := &ast.TupleExpr{Elements: []ast.Expression{expr}}
		tuple.SetPos(line, col)
		for p.peek.Type == lexer.TOKEN_COMMA {
			p.advance()
			p.advance()
			element := p.parseExpression()
			if element == nil {
				p.allowStructLiteral = prev
				return nil
			}
			tuple.Elements = append(tuple.Elements, element)
		}
		expr = tuple
	}
	p.allowStructLiteral = prev

	if !p.expect(lexer.TOKEN_RPAREN) {
		p.errorExpected(lexer.TOKEN_RPAREN, p.current.Type)
		return nil
	}

	if tuple, ok := expr.(*ast.TupleExpr); ok {
		tuple.SetEndPos(p.currentEnd())
	}

	return expr
}

func (p *Parser) parseMemberExpression(object ast.Expression) ast.Expression {
	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}

	return &ast.MemberExpr{Object: object, Field: p.current.Literal, FieldAt: p.currentPosition()}
}

func (p *Parser) parseCallExpression(name ast.Expression) ast.Expression {
	args := p.parseCallArgs()
	if args == nil {
		return nil
	}

	call := &ast.CallExpr{Name: name, Args: args, Spread: p.spread}
	p.spread = false
	return call
}

func (p *Parser) parseCallArgs() []ast.Expression {
	args := []ast.Expression{}

	if p.peek.Type == lexer.TOKEN_RPAREN {
		p.advance()
		return args
	}

	p.advance()
	prev := p.allowStructLiteral
	p.allowStructLiteral = true
	arg := p.parseExpression()
	p.allowStructLiteral = prev
	if arg == nil {
		return nil
	}
	args = append(args, arg)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		prev := p.allowStructLiteral
		p.allowStructLiteral = true
		arg := p.parseExpression()
		p.allowStructLiteral = prev
		if arg == nil {
			return nil
		}
		args = append(args, arg)
	}

	if p.peek.Type == lexer.TOKEN_VARIADIC {
		p.advance()
		p.spread = true
	}

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexSliceExpression(array ast.Expression) ast.Expression {
	p.advance()

	var start ast.Expression
	if p.current.Type != lexer.TOKEN_COLON {
		prev := p.allowStructLiteral
		p.allowStructLiteral = true
		start = p.parseExpression()
		p.allowStructLiteral = prev
		if start == nil {
			return nil
		}
		if p.peek.Type == lexer.TOKEN_COLON {
			p.advance()
		}
	}

	if p.current.Type == lexer.TOKEN_COLON {
		slice := &ast.SliceExpr{Array: array, Start: start}
		if p.peek.Type != lexer.TOKEN_RBRACKET {
			p.advance()
			prev := p.allowStructLiteral
			p.allowStructLiteral = true
			slice.End = p.parseExpression()
			p.allowStructLiteral = prev
			if slice.End == nil {
				return nil
			}
		}
		if !p.expect(lexer.TOKEN_RBRACKET) {
			return nil
		}
		return slice
	}

	if start == nil {
		p.error("expected expression")
		return nil
	}
	if !p.expect(lexer.TOKEN_RBRACKET) {
		return nil
	}

	return &ast.IndexExpr{Array: array, Index: start}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{}

	if p.peek.Type == lexer.TOKEN_RBRACKET {
		p.advance()
		return arr
	}

	p.advance()
	prev := p.allowStructLiteral
	p.allowStructLiteral = true
	el := p.parseExpression()
	p.allowStructLiteral = prev
	if el == nil {
		return nil
	}
	arr.Elements = append(arr.Elements, el)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		prev := p.allowStructLiteral
		p.allowStructLiteral = true
		el := p.parseExpression()
		p.allowStructLiteral = prev
		if el == nil {
			return nil
		}
		arr.Elements = append(arr.Elements, el)
	}

	if !p.expect(lexer.TOKEN_RBRACKET) {
		return nil
	}

	return arr
}

func (p *Parser) parseMapLiteral() ast.Expression {
	m := &ast.MapLiteral{}

	if p.peek.Type == lexer.TOKEN_RBRACE {
		p.advance()
		return m
	}

	p.advance()
	prev := p.allowStructLiteral
	p.allowStructLiteral = true
	key := p.parseExpression()
	p.allowStructLiteral = prev
	if key == nil {
		return nil
	}
	if !p.expect(lexer.TOKEN_COLON) {
		return nil
	}
	p.advance()

	prev = p.allowStructLiteral
	p.allowStructLiteral = true
	value := p.parseExpression()
	p.allowStructLiteral = prev
	if value == nil {
		return nil
	}
	m.Keys = append(m.Keys, key)
	m.Values = append(m.Values, value)

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		prev := p.allowStructLiteral
		p.allowStructLiteral = true
		key := p.parseExpression()
		p.allowStructLiteral = prev
		if key == nil {
			return nil
		}
		if !p.expect(lexer.TOKEN_COLON) {
			return nil
		}
		p.advance()
		prev = p.allowStructLiteral
		p.allowStructLiteral = true
		value := p.parseExpression()
		p.allowStructLiteral = prev
		if value == nil {
			return nil
		}
		m.Keys = append(m.Keys, key)
		m.Values = append(m.Values, value)
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return m
}

func (p *Parser) parseStructLiteral(left ast.Expression) ast.Expression {
	var lit *ast.StructLiteral
	switch l := left.(type) {
	case *ast.IdentLiteral:
		lit = &ast.StructLiteral{Name: l.Value}
	case *ast.MemberExpr:
		module, ok := l.Object.(*ast.IdentLiteral)
		if !ok {
			p.error("expected identifier before {")
			return nil
		}
		lit = &ast.StructLiteral{Module: module.Value, Name: l.Field}
	default:
		p.error("expected identifier before {")
		return nil
	}

	if p.peek.Type == lexer.TOKEN_RBRACE {
		p.advance()
		return lit
	}

	for {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		key := p.current.Literal
		keyAt := p.currentPosition()
		if !p.expect(lexer.TOKEN_COLON) {
			return nil
		}
		p.advance()
		value := p.parseExpression()
		if value == nil {
			return nil
		}
		lit.Keys = append(lit.Keys, key)
		lit.KeyAt = append(lit.KeyAt, keyAt)
		lit.Values = append(lit.Values, value)

		if p.peek.Type != lexer.TOKEN_COMMA {
			break
		}
		p.advance()
	}

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}
	return lit
}

func (p *Parser) parseTernaryExpr(left ast.Expression) ast.Expression {
	expr := &ast.TernaryExpr{Condition: left}
	p.advance()
	expr.Then = p.parseExpression()
	if expr.Then == nil {
		return nil
	}

	if !p.expect(lexer.TOKEN_COLON) {
		return nil
	}
	p.advance()
	expr.Else = p.parseExpressionPratt(TERNARY)
	if expr.Else == nil {
		return nil
	}

	return expr
}

func (p *Parser) parseCoalesceExpr(left ast.Expression) ast.Expression {
	line, col := p.current.Line, p.current.Column
	p.advance()

	expr := &ast.CoalesceExpr{Left: left}
	expr.SetPos(line, col)

	if p.current.Type == lexer.TOKEN_LBRACE {
		expr.Block = p.parseBlockStatement()

		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
		return expr
	}

	switch p.current.Type {
	case lexer.TOKEN_RETURN:
		return p.parseReturnHandler(expr)
	case lexer.TOKEN_BREAK, lexer.TOKEN_CONTINUE:
		return p.parseJumpHandler(expr)
	}

	if p.current.Type == lexer.TOKEN_PIPE {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		expr.ErrorBind = p.current.Literal
		expr.ErrorBindAt = p.currentPosition()

		if !p.checkReservedName(expr.ErrorBind) {
			return nil
		}

		if !p.expect(lexer.TOKEN_PIPE) {
			return nil
		}

		switch p.peek.Type {
		case lexer.TOKEN_LBRACE:
			p.advance()
			expr.Block = p.parseBlockStatement()

			if !p.expect(lexer.TOKEN_RBRACE) {
				return nil
			}
			return expr
		case lexer.TOKEN_RETURN:
			p.advance()
			return p.parseReturnHandler(expr)
		case lexer.TOKEN_BREAK, lexer.TOKEN_CONTINUE:
			p.advance()
			return p.parseJumpHandler(expr)
		}

		p.advance()
	}

	value := p.parseExpression()
	if value == nil {
		return nil
	}

	expr.Block = valueHandler(value)
	return expr
}

func valueHandler(value ast.Expression) *ast.BlockStmt {
	values := []ast.Expression{value}
	if tuple, ok := value.(*ast.TupleExpr); ok {
		values = tuple.Elements
	}

	line, col := value.Position()
	endLine, endCol := value.EndPosition()

	yield := &ast.YieldStmt{Values: values, Implicit: true}
	yield.SetPos(line, col)
	yield.SetEndPos(endLine, endCol)

	block := &ast.BlockStmt{Statements: []ast.Statement{yield}}
	block.SetPos(line, col)
	block.SetEndPos(endLine, endCol)
	return block
}

func (p *Parser) parseJumpHandler(expr *ast.CoalesceExpr) ast.Expression {
	line, col := p.current.Line, p.current.Column
	isBreak := p.current.Type == lexer.TOKEN_BREAK

	label := ""
	if p.peek.Type == lexer.TOKEN_IDENT {
		p.advance()
		label = p.current.Literal
	}

	var stmt ast.Statement
	if isBreak {
		jump := &ast.BreakStmt{Label: label}
		jump.SetPos(line, col)
		jump.SetEndPos(p.currentEnd())
		stmt = jump
	} else {
		jump := &ast.ContinueStmt{Label: label}
		jump.SetPos(line, col)
		jump.SetEndPos(p.currentEnd())
		stmt = jump
	}
	expr.Block = &ast.BlockStmt{Statements: []ast.Statement{stmt}}
	return expr
}

func (p *Parser) parseReturnHandler(expr *ast.CoalesceExpr) ast.Expression {
	retLine, retCol := p.current.Line, p.current.Column

	ret := &ast.ReturnStmt{}
	if p.peek.Type == lexer.TOKEN_SEMICOLON || p.peek.Type == lexer.TOKEN_RBRACE || p.peek.Type == lexer.TOKEN_RPAREN || p.peek.Type == lexer.TOKEN_COMMA {
		ret.SetPos(retLine, retCol)
		ret.SetEndPos(p.currentEnd())
		expr.Block = &ast.BlockStmt{Statements: []ast.Statement{ret}}
		return expr
	}
	for {
		p.advance()
		value := p.parseExpression()
		if value == nil {
			return nil
		}
		ret.Values = append(ret.Values, value)

		if p.peek.Type != lexer.TOKEN_COMMA {
			break
		}
		p.advance()
	}
	ret.SetPos(retLine, retCol)
	ret.SetEndPos(p.currentEnd())
	expr.Block = &ast.BlockStmt{Statements: []ast.Statement{ret}}
	return expr
}

func (p *Parser) parseInExpr(left ast.Expression) ast.Expression {
	line, col := p.current.Line, p.current.Column
	p.advance()

	start := p.parseExpressionPratt(COMPARISON)
	if start == nil {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_RANGE {
		p.advance()
		p.advance()
		end := p.parseExpressionPratt(COMPARISON)
		if end == nil {
			return nil
		}

		lower := &ast.BinaryExpr{Left: left, Operator: ">=", Right: start}
		lower.SetPos(line, col)
		lower.SetEndPos(p.currentEnd())
		upper := &ast.BinaryExpr{Left: left, Operator: "<=", Right: end}
		upper.SetPos(line, col)
		upper.SetEndPos(p.currentEnd())

		result := &ast.BinaryExpr{Left: lower, Operator: "&&", Right: upper}
		result.SetPos(line, col)
		result.SetEndPos(p.currentEnd())

		return result
	}

	expr := &ast.InExpr{Left: left, Right: start}
	expr.SetPos(line, col)
	return expr
}

func (p *Parser) parseSwitchExpression() ast.Expression {
	line, col := p.current.Line, p.current.Column
	p.advance()
	block, value := p.parseSwitchHeader(line, col)
	if block == nil {
		return nil
	}

	if value != nil && !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	expr := &ast.SwitchExpr{Prelude: block.Statements}

	for p.peek.Type == lexer.TOKEN_CASE {
		armLine, armCol := p.peek.Line, p.peek.Column
		condition := p.parseCaseExpression(value, line, col)
		if condition == nil {
			return nil
		}

		if !p.expect(lexer.TOKEN_ARROW) {
			return nil
		}

		arm, ok := p.parseSwitchArm()
		if !ok {
			return nil
		}
		arm.Condition = condition
		arm.SetPos(armLine, armCol)
		expr.Arms = append(expr.Arms, arm)
	}

	armLine, armCol := p.peek.Line, p.peek.Column
	if !p.expect(lexer.TOKEN_DEFAULT) {
		return nil
	}

	if !p.expect(lexer.TOKEN_ARROW) {
		return nil
	}

	arm, ok := p.parseSwitchArm()
	if !ok {
		return nil
	}
	arm.SetPos(armLine, armCol)
	expr.Default = &arm

	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil
	}

	return expr
}

func (p *Parser) parseSwitchArm() (ast.SwitchArm, bool) {
	var arm ast.SwitchArm
	p.advance()

	switch p.current.Type {
	case lexer.TOKEN_RETURN:
		stmt := p.parseReturnArm()
		if stmt == nil {
			return arm, false
		}
		arm.Exit = stmt
	case lexer.TOKEN_BREAK, lexer.TOKEN_CONTINUE:
		arm.Exit = p.parseJumpArm()
	default:
		for {
			value := p.parseExpression()
			if value == nil {
				return arm, false
			}
			arm.Values = append(arm.Values, value)

			if p.peek.Type != lexer.TOKEN_COMMA {
				break
			}
			p.advance()
			p.advance()
		}
	}

	arm.SetEndPos(p.currentEnd())
	if !p.expect(lexer.TOKEN_SEMICOLON) {
		return arm, false
	}
	return arm, true
}

func (p *Parser) parseReturnArm() ast.Statement {
	line, col := p.current.Line, p.current.Column
	ret := &ast.ReturnStmt{}

	if p.peek.Type != lexer.TOKEN_SEMICOLON {
		for {
			p.advance()
			value := p.parseExpression()
			if value == nil {
				return nil
			}
			ret.Values = append(ret.Values, value)

			if p.peek.Type != lexer.TOKEN_COMMA {
				break
			}
			p.advance()
		}
	}

	ret.SetPos(line, col)
	ret.SetEndPos(p.currentEnd())
	return ret
}

func (p *Parser) parseJumpArm() ast.Statement {
	line, col := p.current.Line, p.current.Column
	isBreak := p.current.Type == lexer.TOKEN_BREAK

	label := ""
	if p.peek.Type == lexer.TOKEN_IDENT {
		p.advance()
		label = p.current.Literal
	}

	var stmt ast.Statement
	if isBreak {
		jump := &ast.BreakStmt{Label: label}
		jump.SetPos(line, col)
		jump.SetEndPos(p.currentEnd())
		stmt = jump
	} else {
		jump := &ast.ContinueStmt{Label: label}
		jump.SetPos(line, col)
		jump.SetEndPos(p.currentEnd())
		stmt = jump
	}
	return stmt
}
