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
			p.error("a switch on types does not name the value, name it in each case: case int v:")
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

func (p *Parser) startsTypeLabel(typed bool) bool {
	if p.current.Type == lexer.TOKEN_STAR {
		return true
	}
	if !p.isPrimitiveType() {
		return false
	}
	if p.peek.Type == lexer.TOKEN_COLON || p.peek.Type == lexer.TOKEN_COMMA {
		return true
	}
	return typed && (p.peek.Type == lexer.TOKEN_IDENT || p.peek.Type == lexer.TOKEN_ARROW)
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

func (p *Parser) parseOneCaseExpression(value ast.Expression, line, col int, typed bool) ast.Expression {
	p.advance()
	p.advance()

	if value == nil {
		expr := p.parseExpression()
		if expr != nil && p.peek.Type == lexer.TOKEN_RANGE {
			p.errorf("a range needs a switch value to compare against")
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

	expr := p.parseExpression()
	if expr == nil {
		return nil
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

	if p.peek.Type == lexer.TOKEN_RANGE {
		p.advance()
		p.advance()

		end := p.parseExpression()
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
			Operator:  "<=",
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

func (p *Parser) parseCaseExpression(value ast.Expression, line, col int, typed bool) ast.Expression {
	expr := p.parseOneCaseExpression(value, line, col, typed)
	if expr == nil {
		return nil
	}
	types, named := 1, false
	if m, ok := expr.(*ast.BinaryExpr); ok && m.Bind != nil {
		named = true
	}

	for p.peek.Type == lexer.TOKEN_COMMA {
		right := p.parseOneCaseExpression(value, line, col, typed)
		if right == nil {
			return nil
		}
		types++
		if m, ok := right.(*ast.BinaryExpr); ok && m.Bind != nil {
			named = true
		}

		expr = &ast.BinaryExpr{
			Synthetic: true,
			Left:      expr,
			Operator:  "||",
			Right:     right,
		}
		expr.(*ast.BinaryExpr).SetPos(line, col)
		expr.(*ast.BinaryExpr).SetEndPos(p.currentEnd())
	}

	if typed && types > 1 && named {
		p.error("a case with several types cannot name the value")
		return nil
	}

	return expr
}
