package parser

import (
	"strconv"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseExpression() ast.Expression {
	return p.parseExpressionPratt(LOWEST)
}

func (p *Parser) parseExpressionPratt(precedence Precedence) ast.Expression {
	left := p.parsePrefix()

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
	}

	return left
}

func (p *Parser) parsePrefix() ast.Expression {
	line, col := p.current.Line, p.current.Column

	expr := p.parsePrefixInner()
	if expr == nil {
		return nil
	}
	expr.SetPos(line, col)
	return expr
}

func (p *Parser) parsePrefixInner() ast.Expression {
	switch p.current.Type {
	case lexer.TOKEN_INT_LITERAL:
		i, _ := strconv.ParseInt(p.current.Literal, 10, 64)
		return &ast.IntLiteral{Value: i}
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
	case lexer.TOKEN_NULL:
		return &ast.NullLiteral{}
	case lexer.TOKEN_LBRACKET:
		return p.parseArrayLiteral()
	case lexer.TOKEN_LBRACE:
		return p.parseMapLiteral()

	case lexer.TOKEN_MINUS,
		lexer.TOKEN_NOT,
		lexer.TOKEN_AMP,
		lexer.TOKEN_STAR:
		return p.parseUnaryExpression()

	case lexer.TOKEN_LPAREN:
		return p.parseGroupedExpression()
	}

	p.error("expected expression")
	return nil
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
		lexer.TOKEN_OR:
		return p.parseBinaryExpression(left)
	case lexer.TOKEN_DOT:
		return p.parseMemberExpression(left)
	case lexer.TOKEN_QUESTION_MARK:
		return p.parseTernaryExpr(left)
	case lexer.TOKEN_AS:
		return p.parseCastExpr(left)
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

	prev := p.allowStructLiteral
	p.allowStructLiteral = true
	expr := p.parseExpression()
	p.allowStructLiteral = prev

	if !p.expect(lexer.TOKEN_RPAREN) {
		p.errorExpected(lexer.TOKEN_RPAREN, p.current.Type)
		return nil
	}

	return expr
}

func (p *Parser) parseMemberExpression(object ast.Expression) ast.Expression {
	if !p.expect(lexer.TOKEN_IDENT) {
		return nil
	}

	return &ast.MemberExpr{Object: object, Field: p.current.Literal}
}

func (p *Parser) parseCallExpression(name ast.Expression) ast.Expression {
	args := p.parseCallArgs()
	if args == nil {
		return nil
	}

	return &ast.CallExpr{Name: name, Args: args}
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
	ident, ok := left.(*ast.IdentLiteral)
	if !ok {
		p.error("expected identifier before {")
		return nil
	}
	lit := &ast.StructLiteral{Name: ident.Value}

	if p.peek.Type == lexer.TOKEN_RBRACE {
		p.advance()
		return lit
	}

	for {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil
		}
		key := p.current.Literal
		if !p.expect(lexer.TOKEN_COLON) {
			return nil
		}
		p.advance()
		value := p.parseExpression()
		if value == nil {
			return nil
		}
		lit.Keys = append(lit.Keys, key)
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

func (p *Parser) parseCastExpr(left ast.Expression) ast.Expression {
	p.advance()
	ref := p.parseTypeDefinitionPrefix()
	if ref == nil {
		return nil
	}

	return &ast.CastExpr{Value: left, Type: *ref}
}
