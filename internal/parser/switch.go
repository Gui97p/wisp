package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseSwitchHeader(line, col int) (*ast.BlockStmt, ast.Expression) {
	block := &ast.BlockStmt{}

	name := fmt.Sprintf("__wisp_switch_%d", p.switchNameCount)
	if p.current.Type == lexer.TOKEN_LET {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil, nil
		}
		name = p.current.Literal

		if !p.checkReservedName(name) {
			return nil, nil
		}

		if !p.expect(lexer.TOKEN_ASSIGN) {
			return nil, nil
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
		return nil, nil
	}

	varStmt := &ast.VarStmt{}
	varStmt.Vars = append(varStmt.Vars, ast.Param{Name: name})
	varStmt.Values = append(varStmt.Values, value)
	varStmt.SetPos(line, col)

	block.Statements = append(block.Statements, varStmt)
	block.SetPos(line, col)

	ident := &ast.IdentLiteral{Value: name}
	ident.SetPos(line, col)

	return block, ident
}

func (p *Parser) parseSwitchBlock() []ast.Statement {
	stmts := []ast.Statement{}

	for {
		if p.peek.Type == lexer.TOKEN_CASE || p.peek.Type == lexer.TOKEN_DEFAULT {
			break
		}
		if p.peek.Type == lexer.TOKEN_RBRACE || p.peek.Type == lexer.TOKEN_EOF {
			break
		}

		p.advance()

		stmt := p.parseStatement()

		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	return stmts
}

func (p *Parser) parseOneCaseExpression(value ast.Expression, line, col int) ast.Expression {
	p.advance()
	p.advance()

	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if p.peek.Type == lexer.TOKEN_RANGE {
		p.advance()
		p.advance()

		end := p.parseExpression()
		if end == nil {
			return nil
		}

		left := &ast.BinaryExpr{
			Left:     value,
			Operator: ">=",
			Right:    expr,
		}
		left.SetPos(line, col)
		right := &ast.BinaryExpr{
			Left:     value,
			Operator: "<=",
			Right:    end,
		}
		right.SetPos(line, col)

		expr = &ast.BinaryExpr{
			Left:     left,
			Operator: "&&",
			Right:    right,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
	} else {
		expr = &ast.BinaryExpr{
			Left:     value,
			Operator: "==",
			Right:    expr,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
	}

	return expr
}

func (p *Parser) parseCaseExpression(value ast.Expression, line, col int) ast.Expression {
	expr := p.parseOneCaseExpression(value, line, col)

	for p.peek.Type == lexer.TOKEN_COMMA {
		right := p.parseOneCaseExpression(value, line, col)
		if right == nil {
			return nil
		}

		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: "||",
			Right:    right,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
	}

	return expr
}
