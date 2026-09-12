package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkBlock(block *ast.BlockStmt) {
	a.info.Scopes[block] = a.scope

	for _, stmt := range block.Statements {
		a.checkStmt(stmt)
	}
}

func (a *Analyser) checkStmt(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		if call, ok := s.Expr.(*ast.CallExpr); ok {
			a.checkCallExpr(call)
		} else {
			a.checkExpr(s.Expr)
		}
	case *ast.ReturnStmt:
		a.checkReturnStmt(s)
	// case *ast.VarStmt:
	// case *ast.IfStmt:
	// case *ast.ForStmt:
	// case *ast.LoopStmt:
	// case *ast.AssignStmt:
	// case *ast.IncDecStmt:
	// case *ast.BreakStmt, *ast.ContinueStmt:
	case *ast.MultiVarStmt:
		a.checkMultiVarStmt(s)
	case *ast.BlockStmt:
		a.enterScope()
		a.checkBlock(s)
		a.exitScope()
	}
}

func (a *Analyser) checkReturnStmt(stmt *ast.ReturnStmt) {
	types := make([]Type, len(stmt.Values))
	for i, v := range stmt.Values {
		types[i] = a.checkExpr(v)
	}

	if len(types) != len(a.currentReturns) {
		a.errorf("expected %d return values, got %d", len(a.currentReturns), len(types))
		return
	}

	for i, t := range types {
		if _, ok := t.(InvalidType); ok {
			continue
		}
		if !t.Equals(a.currentReturns[i]) {
			a.errorf("return %d: expected %s, got %s", i+1, a.currentReturns[i].String(), t.String())
		}
	}
}

func (a *Analyser) checkMultiVarStmt(stmt *ast.MultiVarStmt) {
	var types []Type

	if len(stmt.Values) == 1 {
		if call, ok := stmt.Values[0].(*ast.CallExpr); ok {
			types = a.checkCallExpr(call)
		} else {
			types = []Type{a.checkExpr(stmt.Values[0])}
		}
	} else {
		types = make([]Type, len(stmt.Values))
		for i, v := range stmt.Values {
			types[i] = a.checkExpr(v)
		}
	}

	if len(types) != len(stmt.Names) {
		a.errorf("expected %d values, got %d", len(stmt.Names), len(types))
	}

	for i, name := range stmt.Names {
		var t Type
		if i < len(types) {
			t = types[i]
		} else {
			t = InvalidType{}
		}

		sym := &Symbol{Name: name, Kind: VAR, Type: t}
		if !a.scope.Define(sym) {
			a.errorf("variable already declared: %s", name)
		}
	}
}
