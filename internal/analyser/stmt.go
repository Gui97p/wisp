package analyser

import (
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) checkBlock(block *ast.BlockStmt) {
	if a.currentInline == nil {
		a.info.Scopes[block] = a.scope
	}

	a.checkStatements(block.Statements)
}

func (a *Analyser) checkStatements(stmts []ast.Statement) {
	terminated, warned := false, false
	var frees []freedVariable
	defer func() { a.warnUseAfterFree(stmts, frees) }()
	for i, stmt := range stmts {
		if terminated && !warned {
			a.warnf(stmt, "unreachable code")
			warned = true
		}
		a.checkStmt(stmt)
		if loop, isFor := stmt.(*ast.ForStmt); isFor && loop.Lowered != nil {
			stmts[i] = loop.Lowered
		}
		if sym := a.freedSymbol(stmt); sym != nil {
			frees = append(frees, freedVariable{index: i, sym: sym, stmt: stmt})
		}
		switch s := stmt.(type) {
		case *ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.YieldStmt:
			terminated = true
		case *ast.ExpressionStmt:
			if a.callNeverReturns(s.Expr) {
				terminated = true
			}
		}
	}
}

func (a *Analyser) checkStmt(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		switch e := s.Expr.(type) {
		case *ast.CallExpr:
			returns := a.checkCallExpr(e)
			a.rejectFallible([]ast.Expression{e}, returns)
		case *ast.PropagateExpr:
			a.recordValues(e, a.checkPropagate(e))
		case *ast.CoalesceExpr:
			a.recordValues(e, a.checkCoalesce(e))
		default:
			a.checkExpr(s.Expr)
		}
		a.warnMustUse(s)
	case *ast.ReturnStmt:
		a.checkReturnStmt(s)
	case *ast.YieldStmt:
		a.checkYieldStmt(s)
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
	case *ast.DeferStmt:
		a.checkDeferStmt(s)
	case *ast.BlockStmt:
		if s.Label != "" {
			a.loopFrames = append(a.loopFrames, loopFrame{Label: s.Label, Block: true})
		}
		a.enterScope()
		a.checkBlock(s)
		a.exitScope()
		if s.Label != "" {
			a.popLoop()
		}
	}
}

func (a *Analyser) checkReturnStmt(stmt *ast.ReturnStmt) {
	if a.currentInline != nil {
		a.info.InlineReturns[stmt] = a.currentInline
	}
	a.warnReturnedFree(stmt)
	a.voidCallOK = len(a.currentReturns) == 0 && len(stmt.Values) == 1
	a.hint = a.currentReturns
	types := a.rejectFallible(stmt.Values, a.checkExprList(stmt.Values))
	a.voidCallOK = false

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
			a.helpMismatch(a.errorf(stmt, "return %d: expected %s, got %s", i+1, a.currentReturns[i], t), t, a.currentReturns[i])
			continue
		}
		a.checkConstFits(a.valueAt(stmt.Values, len(types), i), a.currentReturns[i])
	}
}

func (a *Analyser) checkIfStmt(stmt *ast.IfStmt) {
	if stmt.IsSwitch && a.pruneTypeSwitch(stmt) {
		stmt.IsSwitch = false
	}
	if stmt.IsSwitch {
		a.pushSwitch()
		defer a.popLoop()
	}

	if stmt.Capture != "" {
		a.lowerCapture(stmt)
	}

	condType := a.checkExpr(stmt.Condition)
	a.requireBool(stmt.Condition, condType, "if condition")
	a.bindSumCase(stmt)

	a.enterScope()
	a.narrowSwitchVariable(stmt.Condition)
	a.checkBlock(stmt.Then)
	a.exitScope()

	if stmt.Else != nil {
		a.checkStmt(stmt.Else)
	}
	a.checkExhaustive(stmt)
}

func (a *Analyser) checkForStmt(stmt *ast.ForStmt) {
	line, col := stmt.Position()

	if stmt.Lowered != nil {
		a.checkStmt(stmt.Lowered)
		return
	}

	if stmt.Range != nil {
		probe := ast.Clone(stmt.Range, ast.CloneOptions{})
		probeType := a.viewAsBase(probe, a.checkExpr(probe))
		if a.lowerPointerFor(stmt, probeType) || a.lowerIteratorFor(stmt, probeType) {
			a.checkStmt(stmt.Lowered)
			return
		}
		rangeType := a.viewAsBase(stmt.Range, a.checkExpr(stmt.Range))

		a.pushLoop(stmt.Label)
		a.enterScope()

		switch rangeType := rangeType.(type) {
		case ArrayType, SpanType:
			a.elementOnly(stmt)
		case PrimitiveType:
			if rangeType.Name == "string" {
				a.elementOnly(stmt)
			}
		}

		switch t := rangeType.(type) {
		case ArrayType:
			a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, stmt.VarAt, line, col)
			if stmt.Var2 != "" {
				a.defineLoopVar(stmt, stmt.Var2, t.Element, stmt.Var2At, line, col)
			}
		case SpanType:
			a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, stmt.VarAt, line, col)
			if stmt.Var2 != "" {
				a.defineLoopVar(stmt, stmt.Var2, t.Element, stmt.Var2At, line, col)
			}
		case PrimitiveType:
			if t.Name == "string" {
				a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, stmt.VarAt, line, col)
				if stmt.Var2 != "" {
					a.defineLoopVar(stmt, stmt.Var2, PrimitiveType{Name: "uint8"}, stmt.Var2At, line, col)
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
			if (step.Kind == ConstInt && step.Int.Sign() == 0) || (step.Kind == ConstFloat && step.Float == 0) {
				a.error(stmt.Step, "range step cannot be zero")
			}
		}
	}
	a.warnEmptyRange(stmt)

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.defineLoopVar(stmt, stmt.Var, PrimitiveType{Name: "int"}, stmt.VarAt, line, col)
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

func (a *Analyser) defineLoopVar(stmt *ast.ForStmt, name string, tp Type, at ast.Position, line, col int) {
	if at.Line != 0 {
		line, col = at.Line, at.Col
	}
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

	a.pushLoop(stmt.Label)
	a.enterScope()
	a.checkBlock(stmt.Body)
	if stmt.UntilCondition != nil {
		a.untilScope = a.scope
		a.untilFrame = len(a.loopFrames) - 1
		a.untilSawBody = false
		t := a.checkExpr(stmt.UntilCondition)
		a.requireBool(stmt.UntilCondition, t, "loop until condition")
		a.info.UntilSeesBody[stmt] = a.untilSawBody
		a.untilScope = nil
	}
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
		if a.assignsSwitchVariable(target) {
			valid = false
		}
		if name, ok := a.importedConst(target); ok {
			a.errorConstAssign(target, name)
			valid = false
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
		if a.isStringElement(target) {
			return
		}
	}
	if s.Op == "=" {
		a.hint = targetTypes
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
				a.helpMismatch(a.errorf(s.Targets[i], "assign expected %s, got %s", targetType.String(), valueTypes[i].String()), valueTypes[i], targetType)
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
	if ptr, isPtr := targetTypes[0].(PointerType); isPtr {
		a.checkPointerStep(s, baseOp, ptr, valueType, s.Values[0])
		return
	}
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
	case "+%", "-%", "*%", "+|", "-|", "*|":
		a.checkOverflowArithmetic(s, baseOp, targetTypes[0], valueType)
	default:
		a.checkArithmetic(s, baseOp, targetTypes[0], valueType)
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
	if a.assignsSwitchVariable(s.Target) {
		return
	}
	if name, ok := a.importedConst(s.Target); ok {
		a.errorConstAssign(s.Target, name)
		return
	}

	t := a.checkExpr(s.Target)
	if a.isErrorField(s.Target) {
		a.error(s.Target, "fields of Error are read-only")
		return
	}
	if a.isStringElement(s.Target) {
		return
	}
	if _, ok := t.(InvalidType); ok {
		return
	}
	if ptr, isPtr := t.(PointerType); isPtr {
		if isVoidPointer(ptr) {
			a.errorf(s.Target, "arithmetic on %s: convert it to a typed pointer first", ptr.String())
		}
		return
	}
	if !isNumeric(t) {
		a.errorf(s.Target, "operator %s invalid for %s", s.Op, t.String())
	}
}

func (a *Analyser) checkPointerStep(node ast.Node, op string, ptr PointerType, value Type, expr ast.Expression) {
	if op != "+" && op != "-" {
		a.errorf(node, "operator %s= invalid for %s", op, ptr.String())
		return
	}
	if isVoidPointer(ptr) {
		a.errorf(node, "arithmetic on %s: convert it to a typed pointer first", ptr.String())
		return
	}
	if !isInteger(value) {
		a.errorf(node, "pointer offset must be an integer, got %s", value.String())
		return
	}
	a.fixUntyped(expr, PrimitiveType{Name: "int"})
}

func (a *Analyser) checkBreakContinue(node ast.Node, label, kind string) {
	if kind == "continue" {
		if !a.hasContinuableLoop() {
			a.errorf(node, "%s outside a loop", kind)
			return
		}
		if label != "" {
			for _, f := range a.loopFrames {
				if f.Label == label && f.Block {
					a.errorf(node, "%s is the label of a block, which has no next turn: use break :%s", label, label)
					return
				}
			}
		}
	} else if label == "" {
		if len(a.loopFrames) == 0 || !slices.ContainsFunc(a.loopFrames, func(f loopFrame) bool { return !f.Block }) {
			a.errorf(node, "%s outside a loop", kind)
			return
		}
		if inner := a.loopFrames[len(a.loopFrames)-1]; inner.Block {
			a.errorf(node, "this break is inside the block :%s, so it is not clear which one it leaves", inner.Label).
				WithHelp("write `break :%s` to leave the block, or give the loop a label and break it by name", inner.Label)
			return
		}
	}

	if kind == "continue" {
		a.markContinued(label)
	}

	if label == "" {
		return
	}

	if slices.ContainsFunc(a.loopFrames, func(f loopFrame) bool { return f.Label == label }) {
		return
	}

	if slices.ContainsFunc(a.yieldFrames, func(f yieldFrame) bool { return f.label == label }) {
		a.errorf(node, "the block :%s gives a value, so the break needs one: write `break :%s <value>;`", label, label)
		return
	}

	a.errorf(node, "unknown label: %s", label)
}

func (a *Analyser) stmtTerminates(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.ExpressionStmt:
		return a.callNeverReturns(s.Expr)
	case *ast.IfStmt:
		if lit, isLit := s.Condition.(*ast.BoolLiteral); isLit && lit.Value && s.Capture == "" {
			return a.blockTerminates(s.Then)
		}
		if s.Else == nil {
			return false
		}
		return a.blockTerminates(s.Then) && a.stmtTerminates(s.Else)
	case *ast.LoopStmt:
		return s.Condition == nil && s.UntilCondition == nil
	case *ast.BlockStmt:
		return a.blockTerminates(s)
	default:
		return false
	}
}

func (a *Analyser) blockTerminates(block *ast.BlockStmt) bool {
	if len(block.Statements) == 0 {
		return false
	}
	last := block.Statements[len(block.Statements)-1]
	return a.stmtTerminates(last)
}

func (a *Analyser) importedConst(expr ast.Expression) (string, bool) {
	for {
		switch e := expr.(type) {
		case *ast.IndexExpr:
			expr = e.Array
		case *ast.MemberExpr:
			if id, ok := e.Object.(*ast.IdentLiteral); ok {
				if sym, found := a.scope.Resolve(id.Value); found {
					if mod, isModule := sym.Type.(*ModuleType); isModule {
						if exp, exported := mod.Exports[e.Field]; exported && exp.Kind == CONST {
							return id.Value + "." + e.Field, true
						}
						return "", false
					}
				}
			}
			expr = e.Object
		default:
			return "", false
		}
	}
}

func (a *Analyser) markContinued(label string) {
	for i := len(a.loopFrames) - 1; i >= 0; i-- {
		f := &a.loopFrames[i]
		if label == "" && !f.SupportsContinue {
			continue
		}
		if label == "" || f.Label == label {
			f.Continued = true
			return
		}
	}
}

func (a *Analyser) isStringElement(target ast.Expression) bool {
	idx, ok := target.(*ast.IndexExpr)
	if !ok || !isString(underlyingOf(a.info.Types[idx.Array])) {
		return false
	}
	a.error(target, "a string is read-only").
		WithHelp("write into the []byte it was made from, or build a new string")
	return true
}

func (a *Analyser) warnEmptyRange(stmt *ast.ForStmt) {
	if a.currentInline != nil || stmt.Start == nil || stmt.End == nil {
		return
	}
	a.quiet = true
	defer func() { a.quiet = false }()
	start, okStart := a.evalConst(stmt.Start)
	end, okEnd := a.evalConst(stmt.End)
	if !okStart || !okEnd || start.Kind != ConstInt || end.Kind != ConstInt {
		return
	}
	down := false
	if stmt.Step != nil {
		step, ok := a.evalConst(stmt.Step)
		if !ok || step.Kind != ConstInt {
			return
		}
		down = step.Int.Sign() < 0
	}
	switch {
	case !down && start.Int.Cmp(end.Int) > 0:
		a.quiet = false
		a.warnf(stmt.End, "this range is empty: it starts above its end").
			WithHelp("to count down, give a negative step, as in %s..%s:-1", start.Int, end.Int)
	case down && start.Int.Cmp(end.Int) < 0:
		a.quiet = false
		a.warnf(stmt.End, "this range is empty: with a negative step it must start above its end")
	}
}
