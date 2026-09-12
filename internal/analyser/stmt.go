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
	case *ast.MultiVarStmt:
		a.checkMultiVarStmt(s)
	case *ast.AssignStmt:
		a.checkAssignStmt(s)
	case *ast.IncDecStmt:
		a.checkIncDecStmt(s)
	// case *ast.IfStmt:
	// a.checkIfStmt(s)
	// case *ast.ForStmt:
	// a.checkForStmt(s)
	// case *ast.LoopStmt:
	// a.checkLoopStmt(s)
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

func (a *Analyser) checkMultiVarStmt(stmt *ast.MultiVarStmt) {
	types := a.checkExprList(stmt.Values)

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

func (a *Analyser) checkVarStmt(stmt *ast.VarStmt) {
	var declaredType Type
	hasExplicitType := stmt.Type.Name != ""

	if hasExplicitType {
		declaredType = a.resolveTypeRef(a.scope, stmt.Type)
	}

	var valueType Type
	if stmt.Value != nil {
		valueType = a.checkExpr(stmt.Value)
	}

	var finalType Type
	switch {
	case hasExplicitType && stmt.Value == nil:
		finalType = declaredType
	case hasExplicitType && stmt.Value != nil:
		if declaredType != nil {
			if _, invalid := valueType.(InvalidType); !invalid && !valueType.Equals(declaredType) {
				a.errorf("variable %s declared as %s, got %s", stmt.Name, declaredType.String(), valueType.String())
			}
		}
		finalType = declaredType
	default:
		a.errorf("variable %s doesn't have a type or value", stmt.Name)
		finalType = InvalidType{}
	}

	if finalType == nil {
		finalType = InvalidType{}
	}

	symbol := &Symbol{Name: stmt.Name, Kind: VAR, Type: finalType}
	if !a.scope.Define(symbol) {
		a.errorf("variable %s already declared in this scope", stmt.Name)
	}
}

func (a *Analyser) checkAssignStmt(s *ast.AssignStmt) {
	if !isAddressable(s.Target) && !isDerefTarget(s.Target) {
		a.errorf("expression not assignable")
		return
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
