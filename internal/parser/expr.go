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
		switch p.peek.Type {
		case lexer.TOKEN_LPAREN:
			p.advance()
			left = p.parseCallExpression(left)
		case lexer.TOKEN_LBRACKET:
			p.advance()
			left = p.parseIndexExpression(left)
		default:
			p.advance()
			left = p.parseInfix(left)
		}
	}

	return left
}

func (p *Parser) parsePrefix() ast.Expression {
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
	case lexer.TOKEN_LBRACKET:
		return p.parseArrayLiteral()

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
	}

	return left
}

func (p *Parser) parseBinaryExpression(left ast.Expression) ast.Expression {
	op := p.current.Literal
	precedece := p.currentPrecedence()

	p.advance()

	right := p.parseExpressionPratt(precedece)

	return &ast.BinaryExpr{
		Left:     left,
		Operator: op,
		Right:    right,
	}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.advance()

	expr := p.parseExpression()

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
	return &ast.CallExpr{Name: name, Args: p.parseCallArgs()}
}

func (p *Parser) parseCallArgs() []ast.Expression {
	args := []ast.Expression{}

	if p.peek.Type == lexer.TOKEN_RPAREN {
		p.advance()
		return args
	}

	p.advance()
	args = append(args, p.parseExpression())

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		args = append(args, p.parseExpression())
	}

	if !p.expect(lexer.TOKEN_RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexExpression(array ast.Expression) ast.Expression {
	p.advance()
	index := p.parseExpression()

	if !p.expect(lexer.TOKEN_RBRACKET) {
		return nil
	}

	return &ast.IndexExpr{Array: array, Index: index}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{}

	if p.peek.Type == lexer.TOKEN_RBRACKET {
		p.advance()
		return arr
	}

	p.advance()
	arr.Elements = append(arr.Elements, p.parseExpression())

	for p.peek.Type == lexer.TOKEN_COMMA {
		p.advance()
		p.advance()
		arr.Elements = append(arr.Elements, p.parseExpression())
	}

	if !p.expect(lexer.TOKEN_RBRACKET) {
		return nil
	}

	return arr
}
