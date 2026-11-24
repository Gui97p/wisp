package parser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

type forGuard struct {
	temps []ast.Statement
	cond  ast.Expression
}

func literalInt(expr ast.Expression) (int64, bool) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		if e.Unsigned {
			return 0, false
		}
		return e.Value, true
	case *ast.UnaryExpr:
		if e.Operator == "-" {
			if inner, ok := e.Value.(*ast.IntLiteral); ok && !inner.Unsigned {
				return -inner.Value, true
			}
		}
	}
	return 0, false
}

func simpleOperand(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IdentLiteral, *ast.IntLiteral, *ast.FloatLiteral, *ast.CharLiteral:
		return true
	case *ast.MemberExpr:
		return simpleOperand(e.Object)
	case *ast.UnaryExpr:
		return e.Operator == "-" && simpleOperand(e.Value)
	}
	return false
}

func (p *Parser) numberAt(value int64, at ast.Expression) ast.Expression {
	literal := &ast.IntLiteral{Value: value}
	line, col := at.Position()
	literal.SetPos(line, col)
	endLine, endCol := at.EndPosition()
	literal.SetEndPos(endLine, endCol)
	return literal
}

func (p *Parser) makeExclusive(stmt *ast.ForStmt, whole ast.Expression) {
	step := int64(1)
	if stmt.Step != nil {
		value, ok := literalInt(stmt.Step)
		if !ok || value == 0 {
			if whole != nil {
				stmt.Range = whole
				stmt.Start, stmt.End, stmt.Step = nil, nil, nil
			}
			return
		}
		step = value
	}
	up := step > 0
	delta := int64(-1)
	cmp := "<"
	if !up {
		delta, cmp = 1, ">"
	}

	if lo, ok := literalInt(stmt.Start); ok {
		if hi, ok := literalInt(stmt.End); ok {
			_ = lo
			stmt.End = p.numberAt(hi+delta, stmt.End)
			return
		}
	}

	guard := &forGuard{}
	bound := func(expr ast.Expression) ast.Expression {
		if simpleOperand(expr) {
			return expr
		}
		name := fmt.Sprintf("__wisp_bound_%d", p.captureCount)
		p.captureCount++
		line, col := expr.Position()
		endLine, endCol := expr.EndPosition()
		decl := &ast.VarStmt{}
		decl.Vars = append(decl.Vars, ast.Param{Name: name})
		decl.Values = append(decl.Values, expr)
		decl.SetPos(line, col)
		decl.SetEndPos(endLine, endCol)
		guard.temps = append(guard.temps, decl)
		ident := &ast.IdentLiteral{Value: name, Synthetic: true}
		ident.SetPos(line, col)
		ident.SetEndPos(endLine, endCol)
		return ident
	}
	stmt.Start = bound(stmt.Start)
	stmt.End = bound(stmt.End)

	copyOf := func(expr ast.Expression) ast.Expression { return ast.Clone(expr, ast.CloneOptions{}) }
	line, col := stmt.End.Position()
	endLine, endCol := stmt.End.EndPosition()

	cond := &ast.BinaryExpr{Left: copyOf(stmt.Start), Operator: cmp, Right: copyOf(stmt.End), Synthetic: true}
	cond.SetPos(line, col)
	cond.SetEndPos(endLine, endCol)
	guard.cond = cond

	op := "-"
	if !up {
		op = "+"
	}
	last := &ast.BinaryExpr{Left: copyOf(stmt.End), Operator: op, Right: p.numberAt(1, stmt.End), Synthetic: true}
	last.SetPos(line, col)
	last.SetEndPos(endLine, endCol)
	stmt.End = last

	if p.guards == nil {
		p.guards = map[*ast.ForStmt]*forGuard{}
	}
	p.guards[stmt] = guard
}

func (p *Parser) wrapFor(stmt *ast.ForStmt) ast.Statement {
	guard, ok := p.guards[stmt]
	if !ok {
		return stmt
	}
	delete(p.guards, stmt)

	line, col := stmt.Position()
	endLine, endCol := stmt.EndPosition()
	inner := &ast.BlockStmt{Statements: []ast.Statement{stmt}}
	inner.SetPos(line, col)
	inner.SetEndPos(endLine, endCol)
	check := &ast.IfStmt{Condition: guard.cond, Then: inner}
	check.SetPos(line, col)
	check.SetEndPos(endLine, endCol)

	block := &ast.BlockStmt{Statements: append(append([]ast.Statement{}, guard.temps...), check)}
	block.SetPos(line, col)
	block.SetEndPos(endLine, endCol)
	return block
}
