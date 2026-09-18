package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/lexer"
)

func (p *Parser) parseSwitchHeader() (*ast.BlockStmt, ast.Expression) {
	block := &ast.BlockStmt{}

	name := fmt.Sprintf("__wisp_switch_%d", p.switchNameCount)
	if p.current.Type == lexer.TOKEN_LET {
		if !p.expect(lexer.TOKEN_IDENT) {
			return nil, nil
		}
		name = p.current.Literal
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

	block.Statements = append(block.Statements, varStmt)

	return block, &ast.IdentLiteral{Value: name}
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

func (p *Parser) parseCaseExpression(value ast.Expression) ast.Expression {
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
		right := &ast.BinaryExpr{
			Left:     value,
			Operator: "<=",
			Right:    end,
		}

		expr = &ast.BinaryExpr{
			Left:     left,
			Operator: "&&",
			Right:    right,
		}
	}

	return expr
}
