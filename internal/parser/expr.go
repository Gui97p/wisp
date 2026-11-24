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

	for p.peek.Type != lexer.TOKEN_EOF && (precedence < p.peekPrecedence() || (p.peek.Type == lexer.TOKEN_LT && isNameExpr(left))) {
		if p.captureAllowed && p.peek.Type == lexer.TOKEN_PIPE && p.atCapture() {
			break
		}
		line, col := left.Position()

		if p.peek.Type == lexer.TOKEN_LT && isNameExpr(left) {
			if generic := p.tryGenericRef(left); generic != nil {
				if g, isGeneric := generic.(*ast.GenericExpr); isGeneric {
					g.SetPos(line, col)
					g.SetEndPos(p.currentEnd())
				} else {
					id := generic.(*ast.IdentLiteral)
					id.SetEndPos(p.currentEnd())
				}
				left = generic
				continue
			}
			if precedence >= p.peekPrecedence() {
				break
			}
		}

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
		f, err := strconv.ParseFloat(p.current.Literal, 64)
		if err != nil {
			p.error("float literal out of range")
			return nil
		}
		return &ast.FloatLiteral{Value: f}
	case lexer.TOKEN_TRUE, lexer.TOKEN_FALSE:
		return &ast.BoolLiteral{Value: p.current.Type == lexer.TOKEN_TRUE}
	case lexer.TOKEN_STATIC:
		if !p.expect(lexer.TOKEN_LBRACE) {
			return nil
		}
		block := p.parseBlockStatement()
		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
		return &ast.BlockExpr{Block: block, Static: true}
	case lexer.TOKEN_IDENT:
		return &ast.IdentLiteral{Value: p.current.Literal}
	case lexer.TOKEN_INT, lexer.TOKEN_UINT,
		lexer.TOKEN_INT8, lexer.TOKEN_UINT8,
		lexer.TOKEN_INT16, lexer.TOKEN_UINT16,
		lexer.TOKEN_INT32, lexer.TOKEN_UINT32,
		lexer.TOKEN_INT64, lexer.TOKEN_UINT64,
		lexer.TOKEN_FLOAT, lexer.TOKEN_FLOAT32, lexer.TOKEN_FLOAT64,
		lexer.TOKEN_BOOL, lexer.TOKEN_BYTE, lexer.TOKEN_STRING:
		if p.peek.Type == lexer.TOKEN_LPAREN || p.peek.Type == lexer.TOKEN_RPAREN || p.peek.Type == lexer.TOKEN_COMMA {
			return &ast.IdentLiteral{Value: p.typeWord()}
		}
	case lexer.TOKEN_NULL:
		return &ast.NullLiteral{}
	case lexer.TOKEN_DOT:
		if p.peek.Type == lexer.TOKEN_LBRACE {
			p.advance()
			return p.parseAnonLiteral()
		}
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		return &ast.DotIdent{Name: p.current.Literal}
	case lexer.TOKEN_LBRACKET:
		return p.parseArrayLiteral()
	case lexer.TOKEN_LBRACE:
		block := p.parseBlockStatement()
		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
		return &ast.BlockExpr{Block: block}
	case lexer.TOKEN_COLON:
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		label := p.current.Literal
		if !p.checkReservedName(label) || !p.expect(lexer.TOKEN_LBRACE) {
			return nil
		}
		block := p.parseBlockStatement()
		if !p.expect(lexer.TOKEN_RBRACE) {
			return nil
		}
		return &ast.BlockExpr{Block: block, Label: label}

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
	if p.current.Type != lexer.TOKEN_STAR && p.current.Type != lexer.TOKEN_QUESTION_MARK {
		return false
	}

	ref := p.parseTypeDefinitionPrefix()
	if ref == nil || ref.IsFunc || ref.Name == "" {
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
		types, fallible, ok := p.parseReturnSpec(true, false)
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

		want := max(1, len(lit.ReturnTypes))
		stmt := &ast.ReturnStmt{}
		for {
			p.advance()

			expr := p.parseExpression()
			if expr == nil {
				return nil
			}
			stmt.Values = append(stmt.Values, expr)

			if len(stmt.Values) >= want || p.peek.Type != lexer.TOKEN_COMMA {
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
	case lexer.TOKEN_RANGE, lexer.TOKEN_RANGE_OPEN:
		return p.parseRangeExpr(left)
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
	savedOperands := p.typeOperands
	if id, ok := name.(*ast.IdentLiteral); ok {
		switch id.Value {
		case "sizeof", "alignof", "offsetof":
			p.typeOperands = true
		}
	}
	args := p.parseCallArgs()
	p.typeOperands = savedOperands
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

func isRangeToken(t lexer.TokenType) bool {
	return t == lexer.TOKEN_RANGE || t == lexer.TOKEN_RANGE_OPEN
}

func (p *Parser) parseSliceBound() ast.Expression {
	prev := p.allowStructLiteral
	p.allowStructLiteral = true
	bound := p.parseExpressionPratt(RANGE)
	p.allowStructLiteral = prev
	return bound
}

func (p *Parser) parseIndexSliceExpression(array ast.Expression) ast.Expression {
	p.advance()

	var start ast.Expression
	if !isRangeToken(p.current.Type) {
		start = p.parseSliceBound()
		if start == nil {
			return nil
		}
		if !isRangeToken(p.peek.Type) {
			if !p.expect(lexer.TOKEN_RBRACKET) {
				return nil
			}
			return &ast.IndexExpr{Array: array, Index: start}
		}
		p.advance()
	}

	inclusive := p.current.Type == lexer.TOKEN_RANGE
	slice := &ast.SliceExpr{Array: array, Start: start}
	if p.peek.Type != lexer.TOKEN_RBRACKET {
		p.advance()
		slice.End = p.parseSliceBound()
		if slice.End == nil {
			return nil
		}
		if inclusive {
			slice.End = p.afterLast(slice.End)
		}
	} else if inclusive {
		p.error("a slice with ..= needs an end: write a..=b, or a.. to go to the end")
		return nil
	}
	if !p.expect(lexer.TOKEN_RBRACKET) {
		return nil
	}
	return slice
}

func (p *Parser) afterLast(end ast.Expression) ast.Expression {
	if value, ok := literalInt(end); ok {
		return p.numberAt(value+1, end)
	}
	next := &ast.BinaryExpr{Left: end, Operator: "+", Right: p.numberAt(1, end), Synthetic: true}
	line, col := end.Position()
	endLine, endCol := end.EndPosition()
	next.SetPos(line, col)
	next.SetEndPos(endLine, endCol)
	return next
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nil
	}
	if len(ref.Layers) == 0 || (!ref.Layers[0].IsSpan && ref.Layers[0].Pointer) {
		p.error("a literal that starts with [ needs an array or slice type, such as [3]int{1, 2, 3}")
		return nil
	}
	if p.peek.Type != lexer.TOKEN_LBRACE {
		p.errorf("expected { after the type %s: write %s{...}", ref.String(), ref.String())
		return nil
	}
	p.advance()

	arr := &ast.ArrayLiteral{Type: ref}
	if p.peek.Type == lexer.TOKEN_RBRACE {
		p.advance()
		return arr
	}
	elements, ok := p.parseLiteralElements()
	if !ok {
		return nil
	}
	arr.Elements = elements
	return arr
}

func (p *Parser) parseLiteralElements() ([]ast.Expression, bool) {
	var elements []ast.Expression
	for {
		p.advance()
		prev := p.allowStructLiteral
		p.allowStructLiteral = true
		el := p.parseExpression()
		p.allowStructLiteral = prev
		if el == nil {
			return nil, false
		}
		elements = append(elements, el)
		if p.peek.Type != lexer.TOKEN_COMMA {
			break
		}
		p.advance()
	}
	if !p.expect(lexer.TOKEN_RBRACE) {
		return nil, false
	}
	return elements, true
}

func (p *Parser) parseAnonLiteral() ast.Expression {
	lit := &ast.AnonLiteral{}
	if p.peek.Type == lexer.TOKEN_RBRACE {
		p.advance()
		return lit
	}

	for {
		p.advance()
		prev := p.allowStructLiteral
		p.allowStructLiteral = true
		first := p.parseExpression()
		p.allowStructLiteral = prev
		if first == nil {
			return nil
		}

		key, isKey := first.(*ast.IdentLiteral)
		if isKey && p.peek.Type == lexer.TOKEN_COLON {
			if len(lit.Elements) > 0 {
				p.error("a literal is either values or name: value pairs, not both")
				return nil
			}
			line, col := key.Position()
			p.advance()
			p.advance()
			value := p.parseExpression()
			if value == nil {
				return nil
			}
			lit.Keys = append(lit.Keys, key.Value)
			lit.KeyAt = append(lit.KeyAt, ast.Position{Line: line, Col: col, EndLine: line, EndCol: col + len(key.Value) - 1})
			lit.Values = append(lit.Values, value)
		} else {
			if len(lit.Keys) > 0 {
				p.error("a literal is either values or name: value pairs, not both")
				return nil
			}
			lit.Elements = append(lit.Elements, first)
		}

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

func (p *Parser) parseStructLiteral(left ast.Expression) ast.Expression {
	var lit *ast.StructLiteral
	switch l := left.(type) {
	case *ast.IdentLiteral:
		lit = &ast.StructLiteral{Name: l.Value}
	case *ast.GenericExpr:
		switch base := l.Base.(type) {
		case *ast.IdentLiteral:
			lit = &ast.StructLiteral{Name: base.Value, TypeArgs: l.Args}
		case *ast.MemberExpr:
			module, ok := base.Object.(*ast.IdentLiteral)
			if !ok {
				p.error("expected identifier before {")
				return nil
			}
			lit = &ast.StructLiteral{Module: module.Value, Name: base.Field, TypeArgs: l.Args}
		default:
			p.error("expected a type name before {")
			return nil
		}
	case *ast.DotIdent:
		lit = &ast.StructLiteral{Name: l.Name, Dot: true}
	case *ast.MemberExpr:
		if generic, isGeneric := l.Object.(*ast.GenericExpr); isGeneric {
			switch base := generic.Base.(type) {
			case *ast.IdentLiteral:
				lit = &ast.StructLiteral{Name: base.Value, TypeArgs: generic.Args, Variant: l.Field}
			case *ast.MemberExpr:
				module, ok := base.Object.(*ast.IdentLiteral)
				if !ok {
					p.error("expected identifier before {")
					return nil
				}
				lit = &ast.StructLiteral{Module: module.Value, Name: base.Field, TypeArgs: generic.Args, Variant: l.Field}
			default:
				p.error("expected a type name before {")
				return nil
			}
			break
		}
		if inner, nested := l.Object.(*ast.MemberExpr); nested {
			module, ok := inner.Object.(*ast.IdentLiteral)
			if !ok {
				p.error("expected identifier before {")
				return nil
			}
			lit = &ast.StructLiteral{Module: module.Value, Name: inner.Field, Variant: l.Field}
			break
		}
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

	if isPanicCall(value) {
		stmt := &ast.ExpressionStmt{Expr: value}
		line, col := value.Position()
		endLine, endCol := value.EndPosition()
		stmt.SetPos(line, col)
		stmt.SetEndPos(endLine, endCol)
		expr.Block = &ast.BlockStmt{Statements: []ast.Statement{stmt}}
		expr.Block.(*ast.BlockStmt).SetPos(line, col)
		expr.Block.(*ast.BlockStmt).SetEndPos(endLine, endCol)
		return expr
	}

	expr.Block = valueHandler(value)
	return expr
}

func isPanicCall(value ast.Expression) bool {
	call, ok := value.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Name.(*ast.IdentLiteral)
	return ok && id.Value == "panic"
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

	keyword := "continue"
	if isBreak {
		keyword = "break"
	}
	label, ok := p.parseJumpLabel(keyword)
	if !ok {
		return nil
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

	if p.peek.Type == lexer.TOKEN_RANGE || p.peek.Type == lexer.TOKEN_RANGE_OPEN {
		upperOperator := "<="
		if p.peek.Type == lexer.TOKEN_RANGE_OPEN {
			upperOperator = "<"
		}
		p.advance()
		p.advance()
		end := p.parseExpressionPratt(COMPARISON)
		if end == nil {
			return nil
		}

		lower := &ast.BinaryExpr{Left: left, Operator: ">=", Right: start}
		lower.SetPos(line, col)
		lower.SetEndPos(p.currentEnd())
		upper := &ast.BinaryExpr{Left: left, Operator: upperOperator, Right: end}
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
	block, value, typed := p.parseSwitchHeader(line, col)
	if block == nil {
		return nil
	}

	if value != nil && !p.expect(lexer.TOKEN_LBRACE) {
		return nil
	}

	expr := &ast.SwitchExpr{Prelude: block.Statements}

	for p.peek.Type == lexer.TOKEN_CASE {
		armLine, armCol := p.peek.Line, p.peek.Column
		condition := p.parseCaseExpression(value, line, col, typed)
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

	if p.peek.Type == lexer.TOKEN_DEFAULT {
		armLine, armCol := p.peek.Line, p.peek.Column
		p.advance()

		if !p.expect(lexer.TOKEN_ARROW) {
			return nil
		}

		arm, ok := p.parseSwitchArm()
		if !ok {
			return nil
		}
		arm.SetPos(armLine, armCol)
		expr.Default = &arm
	}

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
	if len(arm.Values) == 1 && p.peek.Type != lexer.TOKEN_SEMICOLON {
		if _, isBlock := arm.Values[0].(*ast.BlockExpr); isBlock {
			return arm, true
		}
	}
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

	keyword := "continue"
	if isBreak {
		keyword = "break"
	}
	label, ok := p.parseJumpLabel(keyword)
	if !ok {
		return nil
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

func (p *Parser) parseRangeExpr(start ast.Expression) ast.Expression {
	line, col := p.current.Line, p.current.Column
	open := p.current.Type == lexer.TOKEN_RANGE_OPEN
	p.advance()
	end := p.parseExpressionPratt(RANGE)
	if end == nil {
		return nil
	}
	callee := "__wisp_range"
	if open {
		callee = "__wisp_range_open"
	}
	name := &ast.IdentLiteral{Value: callee, Synthetic: true}
	name.SetPos(line, col)
	name.SetEndPos(line, col+1)
	call := &ast.CallExpr{Name: name, Args: []ast.Expression{start, end}, Range: true, RangeOpen: open}
	call.SetPos(line, col)
	call.SetEndPos(p.currentEnd())
	return call
}

func rangeOperands(expr ast.Expression) (start, end, step ast.Expression, open, ok bool) {
	call, isCall := expr.(*ast.CallExpr)
	if !isCall {
		return nil, nil, nil, false, false
	}
	if call.Range {
		return call.Args[0], call.Args[1], nil, call.RangeOpen, true
	}
	member, isMember := call.Name.(*ast.MemberExpr)
	if !isMember || member.Field != "step" || len(call.Args) != 1 {
		return nil, nil, nil, false, false
	}
	inner, isRange := member.Object.(*ast.CallExpr)
	if !isRange || !inner.Range {
		return nil, nil, nil, false, false
	}
	return inner.Args[0], inner.Args[1], call.Args[0], inner.RangeOpen, true
}
