package analyser

import (
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

type yieldFrame struct {
	values []Type
	bound  bool
	infer  bool
	label  string
}

func (a *Analyser) checkYieldStmt(stmt *ast.YieldStmt) {
	var types []Type
	if len(a.yieldFrames) > 0 && len(stmt.Values) == 1 {
		if sw, ok := stmt.Values[0].(*ast.SwitchExpr); ok {
			frame := a.yieldFrames[len(a.yieldFrames)-1]
			if len(frame.values) > 0 || !frame.infer {
				a.checkYieldSwitch(stmt, sw, frame)
				return
			}
		}
	}
	types = a.rejectFallible(stmt.Values, a.checkExprList(stmt.Values))

	if stmt.Target != "" && !slices.ContainsFunc(a.yieldFrames, func(f yieldFrame) bool { return f.label == stmt.Target }) {
		if slices.ContainsFunc(a.loopFrames, func(f loopFrame) bool { return f.Label == stmt.Target }) {
			a.errorf(stmt, "%s labels a block or loop that gives no value, so the break takes none: write `break :%s;`", stmt.Target, stmt.Target)
		} else {
			a.errorf(stmt, "unknown label: %s", stmt.Target)
		}
		return
	}

	if len(a.yieldFrames) == 0 {
		a.error(stmt, "yield can only be used inside a ?? handler or a block that gives a value").
			WithHelp("it gives the value of the ?? when the handler is a block, or of the block `{ ...; yield v; }`")
		return
	}

	top := len(a.yieldFrames) - 1
	if stmt.Target != "" {
		top = -1
		for i := len(a.yieldFrames) - 1; i >= 0; i-- {
			if a.yieldFrames[i].label == stmt.Target {
				top = i
				break
			}
		}
		if top < 0 {
			a.errorf(stmt, "break :%s gives a value, but %s is not the label of a block that gives one", stmt.Target, stmt.Target)
			return
		}
	}
	if a.yieldFrames[top].infer && len(a.yieldFrames[top].values) == 0 {
		if len(types) != 1 {
			a.errorf(stmt, "a block gives exactly one value, got %d", len(types))
			a.yieldFrames[top].values = []Type{InvalidType{}}
			return
		}
		first := types[0]
		if _, bad := first.(InvalidType); bad {
			a.yieldFrames[top].values = []Type{InvalidType{}}
			return
		}
		first = defaultType(first)
		a.yieldFrames[top].values = []Type{first}
	}

	frame := a.yieldFrames[top]
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
		a.helpMismatch(a.errorf(stmt, "yield %d: expected %s, got %s", i+1, frame.values[i], got), got, frame.values[i])
	case frame.bound:
		a.helpMismatch(a.errorf(stmt, "expected %s for the handler value, got %s", frame.values[i], got), got, frame.values[i])
	default:
		a.helpMismatch(a.errorf(stmt, "expected %s for default value, got %s", frame.values[i], got), got, frame.values[i])
	}
}
