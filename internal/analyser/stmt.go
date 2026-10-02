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
		if call, ok := s.Expr.(*ast.CallExpr); ok {
			a.checkCallExpr(call)
		} else {
			a.checkExpr(s.Expr)
		}
	case *ast.ReturnStmt:
		a.checkReturnStmt(s)
	case *ast.VarStmt:
		a.checkVarStmt(s)
	case *ast.ConstStmt:
		a.checkConstStmt(s)
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

func (a *Analyser) checkReturnStmt(stmt *ast.ReturnStmt) {
	types := a.checkExprList(stmt.Values)

	if len(types) != len(a.currentReturns) {
		a.errorf(stmt, "expected %d return values, got %d", len(a.currentReturns), len(types))
		return
	}

	for i, t := range types {
		if _, ok := t.(InvalidType); ok {
			continue
		}
		if eu, ok := a.currentReturns[i].(ErrorUnionType); ok {
			if t.Equals(eu.Payload) {
				a.checkConstFits(a.valueAt(stmt.Values, len(types), i), eu.Payload)
				continue
			}
			errSymbol, _ := a.scope.Resolve("Error")
			if t.Equals(errSymbol.Type) {
				continue
			}
			a.errorf(stmt, "return %d: expected %s or Error, got %s", i+1, eu.Payload, t)
			continue
		}
		if !t.Equals(a.currentReturns[i]) {
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
			a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}, Line: line, Col: col, File: a.currentFile})
			if stmt.Var2 != "" {
				a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: t.Element, Line: line, Col: col, File: a.currentFile})
			}
		case SpanType:
			a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}, Line: line, Col: col, File: a.currentFile})
			if stmt.Var2 != "" {
				a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: t.Element, Line: line, Col: col, File: a.currentFile})
			}
		case *MapType:
			a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: t.Key, Line: line, Col: col, File: a.currentFile})
			if stmt.Var2 != "" {
				a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: t.Value, Line: line, Col: col, File: a.currentFile})
			}
		case PrimitiveType:
			if t.Name == "string" {
				a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}, Line: line, Col: col, File: a.currentFile})
				if stmt.Var2 != "" {
					a.scope.Define(&Symbol{Name: stmt.Var2, Kind: VAR, Type: PrimitiveType{Name: "char"}, Line: line, Col: col, File: a.currentFile})
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
			a.requireNumeric(stmt.End, t, "for iteration count")
		}

		a.pushLoop(stmt.Label)
		a.enterScope()
		a.checkBlock(stmt.Body)
		a.exitScope()
		a.popLoop()
		return
	}

	if stmt.Start != nil {
		a.requireNumeric(stmt.Start, a.checkExpr(stmt.Start), "for start")
	}
	if stmt.End != nil {
		a.requireNumeric(stmt.End, a.checkExpr(stmt.End), "for end")
	}
	if stmt.Step != nil {
		a.requireNumeric(stmt.Step, a.checkExpr(stmt.Step), "for step")
	}

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.scope.Define(&Symbol{Name: stmt.Var, Kind: VAR, Type: PrimitiveType{Name: "int"}, Line: line, Col: col, File: a.currentFile})
	a.checkBlock(stmt.Body)
	a.exitScope()
	a.popLoop()
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
	valueTypes := a.checkExprList(s.Values)

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
			if !valueTypes[i].Equals(targetType) {
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
	if baseOp != "<<" && baseOp != ">>" {
		a.checkConstFits(s.Values[0], targetTypes[0])
	}
	switch baseOp {
	case "&", "|", "^", "<<", ">>":
		a.checkBitwise(s, baseOp, targetTypes[0], valueTypes[0])
	default:
		a.checkArithmetic(s, baseOp, targetTypes[0], valueTypes[0])
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
