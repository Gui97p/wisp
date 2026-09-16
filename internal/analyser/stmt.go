package analyser

import (
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

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
	case *ast.VarStmt:
		a.checkVarStmt(s)
	case *ast.AssignStmt:
		a.checkAssignStmt(s)
	case *ast.IncDecStmt:
		a.checkIncDecStmt(s)
	case *ast.IfStmt:
		a.checkIfStmt(s)
	case *ast.ForStmt:
		a.checkForStmt(s)
	case *ast.LoopStmt:
		a.checkLoopStmt(s)
	case *ast.BreakStmt:
		a.checkBreakContinue(s.Label, "break")
	case *ast.ContinueStmt:
		a.checkBreakContinue(s.Label, "continue")
	case *ast.BlockStmt:
		a.enterScope()
		a.checkBlock(s)
		a.exitScope()
	}
}

func (a *Analyser) checkReturnStmt(stmt *ast.ReturnStmt) {
	types := a.checkExprList(stmt.Values)

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

func (a *Analyser) checkIfStmt(stmt *ast.IfStmt) {
	condType := a.checkExpr(stmt.Condition)
	a.requireBool(condType, "if condition")

	a.enterScope()
	a.checkBlock(stmt.Then)
	a.exitScope()

	if stmt.Else != nil {
		a.checkStmt(stmt.Else)
	}
}

func (a *Analyser) checkForStmt(stmt *ast.ForStmt) {
	if stmt.Range != nil {
		rangeType := a.checkExpr(stmt.Range)

		a.pushLoop(stmt.Label)
		a.enterScope()

		switch t := rangeType.(type) {
		case ArrayType:
			a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}})
			if stmt.Var2 != "" {
				a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: t.Element})
			}
		case SpanType:
			a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}})
			if stmt.Var2 != "" {
				a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: t.Element})
			}
		case *MapType:
			a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: t.Key})
			if stmt.Var2 != "" {
				a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: t.Value})
			}
		case InvalidType:
			// error already reported in stmt.Range check
		default:
			a.errorf("cannot range over %s", rangeType.String())
		}

		a.checkBlock(stmt.Body)
		a.exitScope()
		a.popLoop()
		return
	}

	if stmt.Var == "" {
		if stmt.End != nil {
			t := a.checkExpr(stmt.End)
			a.requireNumeric(t, "for iteration count")
		}

		a.pushLoop(stmt.Label)
		a.enterScope()
		a.checkBlock(stmt.Body)
		a.exitScope()
		a.popLoop()
		return
	}

	if stmt.Start != nil {
		a.requireNumeric(a.checkExpr(stmt.Start), "for start")
	}
	if stmt.End != nil {
		a.requireNumeric(a.checkExpr(stmt.End), "for end")
	}
	if stmt.Step != nil {
		a.requireNumeric(a.checkExpr(stmt.Step), "for step")
	}

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}})
	a.checkBlock(stmt.Body)
	a.exitScope()
	a.popLoop()
}

func (a *Analyser) checkLoopStmt(stmt *ast.LoopStmt) {
	if stmt.Condition != nil {
		t := a.checkExpr(stmt.Condition)
		a.requireBool(t, "loop condition")
	}

	if stmt.UntilCondition != nil {
		t := a.checkExpr(stmt.UntilCondition)
		a.requireBool(t, "loop until condition")
	}

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.checkBlock(stmt.Body)
	a.exitScope()
	a.popLoop()
}

func (a *Analyser) checkVarStmt(stmt *ast.VarStmt) {
	a.checkVarsAndValues(stmt.Vars, stmt.Values, VAR, stmt)
}

func (a *Analyser) checkAssignStmt(s *ast.AssignStmt) {
	if !isAddressable(s.Target) && !isDerefTarget(s.Target) {
		a.errorf("expression not assignable")
		return
	}

	if root := rootIdentifier(s.Target); root != nil {
		if sym, ok := a.scope.Resolve(root.Value); ok && sym.Kind == CONST {
			a.errorf("cannot assign to constant %s", root.Value)
			return
		}
	}

	targetType := a.checkExpr(s.Target)
	valueType := a.checkExpr(s.Value)

	if _, ok := targetType.(InvalidType); ok {
		return
	}
	if _, ok := valueType.(InvalidType); ok {
		return
	}

	if s.Op == "=" {
		if !valueType.Equals(targetType) {
			a.errorf("assign expected %s, got %s", targetType.String(), valueType.String())
		}
		return
	}

	baseOp := s.Op[:len(s.Op)-1]
	a.checkArithmetic(baseOp, targetType, valueType)
}

func (a *Analyser) checkIncDecStmt(s *ast.IncDecStmt) {
	if !isAddressable(s.Target) && !isDerefTarget(s.Target) {
		a.errorf("expression not incrementable")
		return
	}

	if root := rootIdentifier(s.Target); root != nil {
		if sym, ok := a.scope.Resolve(root.Value); ok && sym.Kind == CONST {
			a.errorf("cannot assign to constant %s", root.Value)
			return
		}
	}

	t := a.checkExpr(s.Target)
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !isNumeric(t) {
		a.errorf("operator %s invalid for %s", s.Op, t.String())
	}
}

func (a *Analyser) checkBreakContinue(label, kind string) {
	if len(a.loopLabels) == 0 {
		a.errorf("%s outside a loop", kind)
		return
	}

	if label == "" {
		return
	}

	if slices.Contains(a.loopLabels, label) {
		return
	}

	a.errorf("unknown label: %s", label)
}

func stmtTerminates(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		if s.Else == nil {
			return false
		}
		return blockTerminates(s.Then) && stmtTerminates(s.Else)
	case *ast.LoopStmt:
		return s.Condition == nil && s.UntilCondition == nil
	case *ast.BlockStmt:
		return blockTerminates(s)
	default:
		return false
	}
}

func blockTerminates(block *ast.BlockStmt) bool {
	if len(block.Statements) == 0 {
		return false
	}
	last := block.Statements[len(block.Statements)-1]
	return stmtTerminates(last)
}
