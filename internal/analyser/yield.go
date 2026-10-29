package analyser

import "github.com/Gui97p/wisp/internal/ast"

type yieldFrame struct {
	values []Type
	bound  bool
}

func (a *Analyser) checkYieldStmt(stmt *ast.YieldStmt) {
	var types []Type
	if len(a.yieldFrames) > 0 && len(stmt.Values) == 1 {
		if sw, ok := stmt.Values[0].(*ast.SwitchExpr); ok {
			frame := a.yieldFrames[len(a.yieldFrames)-1]
			a.checkYieldSwitch(stmt, sw, frame)
			return
		}
	}
	types = a.rejectFallible(stmt.Values, a.checkExprList(stmt.Values))

	if len(a.yieldFrames) == 0 {
		a.error(stmt, "yield can only be used inside a ?? handler").
			WithHelp("it gives the value of the ?? when the handler is a block")
		return
	}

	frame := a.yieldFrames[len(a.yieldFrames)-1]
	if len(frame.values) == 0 {
		a.error(stmt, "this call has no value to yield")
		return
	}

	if len(types) != len(frame.values) {
		a.countMismatch(stmt, frame, len(types))
		return
	}

	for i, t := range types {
		if _, ok := t.(InvalidType); ok {
			continue
		}
		if !a.coerceAt(stmt.Values, len(types), i, t, frame.values[i]) {
			a.yieldMismatch(stmt, frame, i, t)
			continue
		}
		a.checkConstFits(a.valueAt(stmt.Values, len(types), i), frame.values[i])
	}
}

func (a *Analyser) checkYieldSwitch(stmt *ast.YieldStmt, sw *ast.SwitchExpr, frame yieldFrame) {
	if len(frame.values) == 0 {
		a.error(stmt, "this call has no value to yield")
		return
	}
	got := a.checkSwitchExpr(sw, frame.values)
	a.recordValues(sw, got)
}

func (a *Analyser) countMismatch(stmt *ast.YieldStmt, frame yieldFrame, got int) {
	if !stmt.Implicit {
		a.errorf(stmt, "expected %d yield values, got %d", len(frame.values), got)
		return
	}
	if len(frame.values) > 1 {
		a.errorf(stmt, "?? needs %d values for this call, written as a tuple (a, b), got %d", len(frame.values), got)
		return
	}
	a.errorf(stmt, "?? needs 1 value for this call, got %d", got)
}

func (a *Analyser) yieldMismatch(stmt *ast.YieldStmt, frame yieldFrame, i int, got Type) {
	switch {
	case !stmt.Implicit:
		a.errorf(stmt, "yield %d: expected %s, got %s", i+1, frame.values[i], got)
	case frame.bound:
		a.errorf(stmt, "expected %s for the handler value, got %s", frame.values[i], got)
	default:
		a.errorf(stmt, "expected %s for default value, got %s", frame.values[i], got)
	}
}
