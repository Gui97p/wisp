package analyser

import "github.com/Gui97p/wisp/internal/ast"

type yieldFrame struct {
	values []Type
}

func (a *Analyser) checkYieldStmt(stmt *ast.YieldStmt) {
	a.markHandlers(stmt.Values)
	types := a.rejectFallible(stmt.Values, a.checkExprList(stmt.Values))

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
		a.errorf(stmt, "expected %d yield values, got %d", len(frame.values), len(types))
		return
	}

	for i, t := range types {
		if _, ok := t.(InvalidType); ok {
			continue
		}
		if !a.coerceAt(stmt.Values, len(types), i, t, frame.values[i]) {
			a.errorf(stmt, "yield %d: expected %s, got %s", i+1, frame.values[i], t)
			continue
		}
		a.checkConstFits(a.valueAt(stmt.Values, len(types), i), frame.values[i])
	}
}
