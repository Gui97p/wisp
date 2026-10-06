package analyser

import (
	"github.com/Gui97p/wisp/internal/target"
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) checkBlock(block *ast.BlockStmt) {
	a.info.Scopes[block] = a.scope

	for _, stmt := range block.Statements {
		a.checkStmt(stmt)
	}
}

func (a *Analyser) checkGroup(group *ast.GroupStmt) {
	for _, stmt := range group.Statements {
		a.checkStmt(stmt)
	}
}

func (a *Analyser) checkStmt(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		a.markHandlers([]ast.Expression{s.Expr})
		switch e := s.Expr.(type) {
		case *ast.CallExpr:
			returns := a.checkCallExpr(e)
			a.rejectFallible([]ast.Expression{e}, returns)
		case *ast.PropagateExpr:
			values := a.checkPropagate(e)
			if len(values) > 0 {
				a.info.Types[e] = values[0]
			} else {
				a.info.Types[e] = VoidType{}
			}
		default:
			a.checkExpr(s.Expr)
		}
	case *ast.ReturnStmt:
		a.checkReturnStmt(s)
	case *ast.VarStmt:
		a.markHandlers(s.Values)
		a.checkVarStmt(s)
	case *ast.ConstStmt:
		a.markHandlers(s.Values)
		a.checkConstStmt(s)
	case *ast.AssignStmt:
		a.markHandlers(s.Values)
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
		a.checkBreakContinue(s, s.Label, "break")
	case *ast.ContinueStmt:
		a.checkBreakContinue(s, s.Label, "continue")
	case *ast.NativeStmt:
		a.checkNativeStmt(s)
	case *ast.BlockStmt:
		a.enterScope()
		a.checkBlock(s)
		a.exitScope()
	case *ast.GroupStmt:
		a.checkGroup(s)
	}
}

func (a *Analyser) markHandlers(exprs []ast.Expression) {
	for _, e := range exprs {
		if c, ok := e.(*ast.CoalesceExpr); ok {
			a.handlerOK[c] = true
		}
	}
}

func (a *Analyser) checkReturnStmt(stmt *ast.ReturnStmt) {
	a.markHandlers(stmt.Values)
	types := a.rejectFallible(stmt.Values, a.checkExprList(stmt.Values))

	if a.currentFallible && len(types) == 1 {
		if _, isError := types[0].(ErrorType); isError {
			a.info.ErrorReturns[stmt] = true
			return
		}
	}

	if len(types) != len(a.currentReturns) {
		a.errorf(stmt, "expected %d return values, got %d", len(a.currentReturns), len(types))
		return
	}

	for i, t := range types {
		if _, ok := t.(InvalidType); ok {
			continue
		}
		if !a.coerceAt(stmt.Values, len(types), i, t, a.currentReturns[i]) {
			a.errorf(stmt, "return %d: expected %s, got %s", i+1, a.currentReturns[i], t)
			continue
		}
		a.checkConstFits(a.valueAt(stmt.Values, len(types), i), a.currentReturns[i])
	}
}

func (a *Analyser) checkIfStmt(stmt *ast.IfStmt) {
	if stmt.IsSwitch {
		a.pushSwitch()
		defer a.popLoop()
	}

	condType := a.checkExpr(stmt.Condition)
	a.requireBool(stmt.Condition, condType, "if condition")

	a.enterScope()
	a.checkBlock(stmt.Then)
	a.exitScope()

	if stmt.Else != nil {
		a.checkStmt(stmt.Else)
	}
}

func (a *Analyser) checkForStmt(stmt *ast.ForStmt) {
	line, col := stmt.Position()

	if stmt.Range != nil {
		rangeType := a.checkExpr(stmt.Range)

		a.pushLoop(stmt.Label)
		a.enterScope()

		switch t := rangeType.(type) {
		case ArrayType:
			a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, line, col)
			if stmt.Var2 != "" {
				a.defineLoopVar(stmt, stmt.Var2, t.Element, line, col)
			}
		case SpanType:
			a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, line, col)
			if stmt.Var2 != "" {
				a.defineLoopVar(stmt, stmt.Var2, t.Element, line, col)
			}
		case *MapType:
			a.defineLoopVar(stmt, stmt.Var, t.Key, line, col)
			if stmt.Var2 != "" {
				a.defineLoopVar(stmt, stmt.Var2, t.Value, line, col)
			}
		case PrimitiveType:
			if t.Name == "string" {
				a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, line, col)
				if stmt.Var2 != "" {
					a.defineLoopVar(stmt, stmt.Var2, PrimitiveType{Name: "char"}, line, col)
				}
			} else {
				a.errorf(stmt.Range, "cannot range over %s", rangeType)
			}
		case InvalidType:
			// error already reported in stmt.Range check
		default:
			a.errorf(stmt.Range, "cannot range over %s", rangeType)
		}

		a.checkBlock(stmt.Body)
		a.exitScope()
		a.popLoop()
		return
	}

	if stmt.Var == "" {
		if stmt.End != nil {
			t := a.checkExpr(stmt.End)
			if _, bad := t.(InvalidType); !bad && !isInteger(underlying(t)) {
				a.errorf(stmt.End, "for iteration count must be an integer, got %s", t)
			}
		}

		a.pushLoop(stmt.Label)
		a.enterScope()
		a.checkBlock(stmt.Body)
		a.exitScope()
		a.popLoop()
		return
	}

	if stmt.Start != nil {
		a.coerceRangeValue(&stmt.Start, "for start")
	}
	if stmt.End != nil {
		a.coerceRangeValue(&stmt.End, "for end")
	}
	if stmt.Step != nil {
		a.coerceRangeValue(&stmt.Step, "for step")
		if step, ok := a.evalConst(stmt.Step); ok {
			if (step.Kind == ConstInt && step.Int.Sign() <= 0) || (step.Kind == ConstFloat && step.Float <= 0) {
				a.error(stmt.Step, "range step must be positive: the direction comes from the bounds, as in 10..1:2")
			}
		}
	}

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, line, col)
	a.checkBlock(stmt.Body)
	a.exitScope()
	a.popLoop()
}

func (a *Analyser) coerceRangeValue(slot *ast.Expression, what string) {
	t := a.checkExpr(*slot)
	if _, bad := t.(InvalidType); bad {
		return
	}
	if !isInteger(underlying(t)) {
		a.errorf(*slot, "%s must be an integer, got %s", what, t)
		return
	}
	intType := PrimitiveType{Name: "int"}
	if !a.coerce(slot, t, intType) {
		a.errorf(*slot, "%s must be an int, got %s: convert it explicitly", what, t)
		return
	}
	a.checkConstFits(*slot, intType)
}

func (a *Analyser) defineLoopVar(stmt *ast.ForStmt, name string, tp Type, line, col int) {
	sym := &Symbol{Name: name, Kind: VAR, Type: tp, Line: line, Col: col, File: a.currentFile}
	a.scope.Define(sym)
	a.info.VarSymbols[stmt] = append(a.info.VarSymbols[stmt], sym)
	a.info.VarTypes[stmt] = append(a.info.VarTypes[stmt], tp)
}

func (a *Analyser) checkLoopStmt(stmt *ast.LoopStmt) {
	if stmt.Condition != nil {
		t := a.checkExpr(stmt.Condition)
		a.requireBool(stmt.Condition, t, "loop condition")
	}

	if stmt.UntilCondition != nil {
		t := a.checkExpr(stmt.UntilCondition)
		a.requireBool(stmt.UntilCondition, t, "loop until condition")
	}

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.checkBlock(stmt.Body)
	a.exitScope()
	a.popLoop()
}

func (a *Analyser) checkVarStmt(stmt *ast.VarStmt) {
	a.checkVarsAndValues(stmt, stmt.Vars, stmt.Values, VAR)
}

func (a *Analyser) checkConstStmt(stmt *ast.ConstStmt) {
	a.checkVarsAndValues(stmt, stmt.Vars, stmt.Values, CONST)
}

func (a *Analyser) checkAssignStmt(s *ast.AssignStmt) {
	valid := true
	for _, target := range s.Targets {
		if !isAddressable(target) && !isDerefTarget(target) {
			a.error(target, "expression not assignable")
			valid = false
			continue
		}

		if root := rootIdentifier(target); root != nil {
			if sym, ok := a.scope.Resolve(root.Value); ok && sym.Kind == CONST {
				a.errorConstAssign(target, root.Value)
				valid = false
			}
		}
	}
	if !valid {
		return
	}

	targetTypes := make([]Type, len(s.Targets))
	for i, target := range s.Targets {
		targetTypes[i] = a.checkExpr(target)
	}
	for _, target := range s.Targets {
		if a.isErrorField(target) {
			a.error(target, "fields of Error are read-only")
			return
		}
	}
	valueTypes := a.checkExprList(s.Values)
	valueTypes, _ = a.expandFallible(s.Values, valueTypes, len(targetTypes))
	valueTypes = a.rejectFallible(s.Values, valueTypes)

	if len(valueTypes) != len(targetTypes) {
		a.errorf(s, "expected %d values, got %d", len(targetTypes), len(valueTypes))
		return
	}

	if s.Op == "=" {
		for i, targetType := range targetTypes {
			if _, ok := targetType.(InvalidType); ok {
				continue
			}
			if _, ok := valueTypes[i].(InvalidType); ok {
				continue
			}
			if !a.coerceAt(s.Values, len(s.Targets), i, valueTypes[i], targetType) {
				a.errorf(s.Targets[i], "assign expected %s, got %s", targetType.String(), valueTypes[i].String())
				continue
			}
			a.checkConstFits(a.valueAt(s.Values, len(s.Targets), i), targetType)
		}
		return
	}

	if _, ok := targetTypes[0].(InvalidType); ok {
		return
	}
	if _, ok := valueTypes[0].(InvalidType); ok {
		return
	}

	baseOp := s.Op[:len(s.Op)-1]
	valueType := valueTypes[0]
	if baseOp == "/" || baseOp == "%" {
		a.checkDivisor(s.Values[0], valueType)
	}
	if baseOp != "<<" && baseOp != ">>" {
		a.checkConstFits(s.Values[0], targetTypes[0])
		if numericLike(targetTypes[0]) && numericLike(valueType) && a.coerce(&s.Values[0], valueType, targetTypes[0]) {
			valueType = targetTypes[0]
		}
	} else {
		a.fixUntyped(s.Values[0], PrimitiveType{Name: "int"})
	}
	switch baseOp {
	case "&", "|", "^", "<<", ">>":
		a.checkBitwise(s, baseOp, targetTypes[0], valueType)
	default:
		a.checkArithmetic(s, baseOp, targetTypes[0], valueType)
	}
}

func (a *Analyser) checkNativeStmt(s *ast.NativeStmt) {
	if s.Backend != "" {
		if err := target.ValidateSelector(s.Backend); err != nil {
			a.errorf(s, "native: %s", err)
		}
	}

	for _, bd := range s.Bindings {
		if _, ok := a.checkExpr(bd.Var).(InvalidType); ok {
			continue
		}
		if !bd.Out {
			continue
		}
		if sym, ok := a.scope.Resolve(bd.Var.Value); ok && sym.Kind == CONST {
			a.errorConstAssign(bd.Var, bd.Var.Value)
		}
	}
}

func (a *Analyser) checkIncDecStmt(s *ast.IncDecStmt) {
	if !isAddressable(s.Target) && !isDerefTarget(s.Target) {
		a.error(s.Target, "expression not incrementable")
		return
	}

	if root := rootIdentifier(s.Target); root != nil {
		if sym, ok := a.scope.Resolve(root.Value); ok && sym.Kind == CONST {
			a.errorConstAssign(s.Target, root.Value)
			return
		}
	}

	t := a.checkExpr(s.Target)
	if a.isErrorField(s.Target) {
		a.error(s.Target, "fields of Error are read-only")
		return
	}
	if _, ok := t.(InvalidType); ok {
		return
	}
	if !isNumeric(t) {
		a.errorf(s.Target, "operator %s invalid for %s", s.Op, t.String())
	}
}

func (a *Analyser) checkBreakContinue(node ast.Node, label, kind string) {
	if kind == "continue" {
		if !a.hasContinuableLoop() {
			a.errorf(node, "%s outside a loop", kind)
			return
		}
	} else if len(a.loopFrames) == 0 {
		a.errorf(node, "%s outside a loop", kind)
		return
	}

	if label == "" {
		return
	}

	if slices.ContainsFunc(a.loopFrames, func(f loopFrame) bool { return f.Label == label }) {
		return
	}

	a.errorf(node, "unknown label: %s", label)
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
	case *ast.GroupStmt:
		return blockTerminates((*ast.BlockStmt)(s))
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
