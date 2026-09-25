package lsp

import "github.com/Gui97p/wisp/internal/ast"

func contains(node ast.Node, line, col int) bool {
	sl, sc := node.Position()
	el, ec := node.EndPosition()
	if el == 0 && ec == 0 {
		el, ec = sl, sc
	}

	if line < sl || (line == sl && col < sc) {
		return false
	}
	if line > el || (line == el && col > ec) {
		return false
	}
	return true
}

func paramContains(p *ast.Param, line, col int) bool {
	sl, sc, el, ec := p.Line, p.Col, p.EndLine, p.EndCol
	if sl == 0 && sc == 0 {
		return false
	}
	if el == 0 && ec == 0 {
		el, ec = sl, sc
	}
	if line < sl || (line == sl && col < sc) {
		return false
	}
	if line > el || (line == el && col > ec) {
		return false
	}
	return true
}

func paramAt(program *ast.Program, line, col int) (*ast.Param, ast.Node, int) {
	for _, decl := range program.Declarations {
		if !contains(decl, line, col) {
			continue
		}

		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Receiver != nil && paramContains(d.Receiver, line, col) {
				return d.Receiver, d, -1
			}
			for i := range d.Params {
				if paramContains(&d.Params[i], line, col) {
					return &d.Params[i], d, i
				}
			}
			if d.Body != nil {
				if p, owner, idx := paramInStmt(d.Body, line, col); p != nil {
					return p, owner, idx
				}
			}
		case *ast.ConstDecl:
			for i := range d.Vars {
				if paramContains(&d.Vars[i], line, col) {
					return &d.Vars[i], d, i
				}
			}
		}
	}
	return nil, nil, -1
}

func paramInStmt(stmt ast.Statement, line, col int) (*ast.Param, ast.Node, int) {
	if stmt == nil {
		return nil, nil, -1
	}

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, inner := range s.Statements {
			if p, owner, idx := paramInStmt(inner, line, col); p != nil {
				return p, owner, idx
			}
		}
	case *ast.GroupStmt:
		for _, inner := range s.Statements {
			if p, owner, idx := paramInStmt(inner, line, col); p != nil {
				return p, owner, idx
			}
		}
	case *ast.VarStmt:
		for i := range s.Vars {
			if paramContains(&s.Vars[i], line, col) {
				return &s.Vars[i], s, i
			}
		}
	case *ast.ConstStmt:
		for i := range s.Vars {
			if paramContains(&s.Vars[i], line, col) {
				return &s.Vars[i], s, i
			}
		}
	case *ast.IfStmt:
		if p, owner, idx := paramInStmt(s.Then, line, col); p != nil {
			return p, owner, idx
		}
		return paramInStmt(s.Else, line, col)
	case *ast.ForStmt:
		return paramInStmt(s.Body, line, col)
	case *ast.LoopStmt:
		return paramInStmt(s.Body, line, col)
	}
	return nil, nil, -1
}

func exprInProgram(program *ast.Program, line, col int) ast.Expression {
	for _, decl := range program.Declarations {
		if !contains(decl, line, col) {
			continue
		}

		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body != nil {
				if r := exprInStmt(d.Body, line, col); r != nil {
					return r
				}
			}
		case *ast.ConstDecl:
			if r := firstExpr(line, col, d.Values...); r != nil {
				return r
			}
		}
	}
	return nil
}

func firstExpr(line, col int, exprs ...ast.Expression) ast.Expression {
	for _, e := range exprs {
		if e == nil {
			continue
		}
		if r := exprAt(e, line, col); r != nil {
			return r
		}
	}
	return nil
}

func exprAt(expr ast.Expression, line, col int) ast.Expression {
	if expr == nil || !contains(expr, line, col) {
		return nil
	}

	var child ast.Expression

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		child = firstExpr(line, col, e.Left, e.Right)
	case *ast.UnaryExpr:
		child = exprAt(e.Value, line, col)
	case *ast.CallExpr:
		child = firstExpr(line, col, e.Name)
		if child == nil {
			child = firstExpr(line, col, e.Args...)
		}
	case *ast.MemberExpr:
		child = exprAt(e.Object, line, col)
	case *ast.IndexExpr:
		child = firstExpr(line, col, e.Array, e.Index)
	case *ast.SliceExpr:
		child = firstExpr(line, col, e.Array, e.Start, e.End)
	case *ast.TernaryExpr:
		child = firstExpr(line, col, e.Condition, e.Then, e.Else)
	case *ast.CoalesceExpr:
		child = firstExpr(line, col, e.Left, e.Default)
		if child == nil {
			child = exprInStmt(e.Block, line, col)
		}
	case *ast.CastExpr:
		child = exprAt(e.Value, line, col)
	case *ast.InExpr:
		child = firstExpr(line, col, e.Left, e.Right)
	case *ast.ArrayLiteral:
		child = firstExpr(line, col, e.Elements...)
	case *ast.MapLiteral:
		child = firstExpr(line, col, e.Keys...)
		if child == nil {
			child = firstExpr(line, col, e.Values...)
		}
	case *ast.StructLiteral:
		child = firstExpr(line, col, e.Values...)
	case *ast.FuncLiteral:
		if e.Block != nil {
			child = exprInStmt(e.Block, line, col)
		}
	}

	if child != nil {
		return child
	}
	return expr
}

func exprInStmt(stmt ast.Statement, line, col int) ast.Expression {
	if stmt == nil {
		return nil
	}

	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		return exprAt(s.Expr, line, col)
	case *ast.BlockStmt:
		return exprInStmtList(s.Statements, line, col)
	case *ast.GroupStmt:
		return exprInStmtList(s.Statements, line, col)
	case *ast.VarStmt:
		return firstExpr(line, col, s.Values...)
	case *ast.ConstStmt:
		return firstExpr(line, col, s.Values...)
	case *ast.ReturnStmt:
		return firstExpr(line, col, s.Values...)
	case *ast.IfStmt:
		if r := exprAt(s.Condition, line, col); r != nil {
			return r
		}
		if r := exprInStmt(s.Then, line, col); r != nil {
			return r
		}
		return exprInStmt(s.Else, line, col)
	case *ast.ForStmt:
		if r := firstExpr(line, col, s.Range, s.Start, s.End, s.Step); r != nil {
			return r
		}
		return exprInStmt(s.Body, line, col)
	case *ast.LoopStmt:
		if r := firstExpr(line, col, s.Condition, s.UntilCondition); r != nil {
			return r
		}
		return exprInStmt(s.Body, line, col)
	case *ast.AssignStmt:
		return firstExpr(line, col, s.Target, s.Value)
	case *ast.IncDecStmt:
		return exprAt(s.Target, line, col)
	}

	return nil
}

func exprInStmtList(stmts []ast.Statement, line, col int) ast.Expression {
	for _, s := range stmts {
		if r := exprInStmt(s, line, col); r != nil {
			return r
		}
	}
	return nil
}
