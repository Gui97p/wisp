package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkSwitchExprValue(expr *ast.SwitchExpr) Type {
	values := a.checkSwitchExpr(expr, nil)
	switch len(values) {
	case 1:
		return values[0]
	case 0:
		a.error(expr, "this match gives no value").
			WithHelp("at least one arm has to give a value")
	default:
		a.errorf(expr, "match gives %d values, expected 1 in this context", len(values))
	}
	return InvalidType{}
}

func (a *Analyser) checkSwitchExpr(expr *ast.SwitchExpr, expected []Type) []Type {
	hinted := a.hintedSwitch
	a.hintedSwitch = false
	if failed := a.pruneTypeSwitchExpr(expr); failed != nil {
		return []Type{failed}
	}

	a.enterScope()
	defer a.exitScope()

	for _, stmt := range expr.Prelude {
		a.checkStmt(stmt)
	}

	arms := switchArms(expr)

	armTypes := make([][]Type, len(arms))
	count, valid := -1, true
	for i, arm := range arms {
		a.enterScope()
		if arm.Condition != nil {
			condType := a.checkExpr(arm.Condition)
			a.requireBool(arm.Condition, condType, "arm condition")
			a.narrowSwitchVariable(arm.Condition)
			a.bindSumArm(arm)
		}

		if arm.Exit != nil {
			a.checkStmt(arm.Exit)
		} else {
			if expected != nil {
				a.hint = expected
			}
			armTypes[i] = a.rejectFallible(arm.Values, a.checkExprList(arm.Values))
			a.hint = nil
			switch {
			case count == -1:
				count = len(armTypes[i])
			case len(armTypes[i]) != count:
				a.errorf(arm, "this arm gives %d values, the others give %d", len(armTypes[i]), count)
				valid = false
			}
		}
		a.exitScope()
	}

	if !valid {
		return []Type{InvalidType{}}
	}

	if expr.Default == nil && !a.coversEverything(arms) {
		a.errorRange(switchKeyword(expr), "a match expression needs a `_` arm, or an arm for every member").
			WithHelp("the expression must always give a value")
		return []Type{InvalidType{}}
	}

	if count == -1 {
		if expected == nil {
			return nil
		}
		a.info.SwitchValues[expr] = nil
		return nil
	}

	if hinted && expected != nil && len(expected) != count {
		expected = nil
	}
	if expected != nil && len(expected) != count {
		a.errorf(expr, "this match gives %d values, the call gives %d", count, len(expected))
		return []Type{InvalidType{}}
	}

	result := make([]Type, count)
	for i := range result {
		result[i] = a.unifyArms(arms, armTypes, i, expected, hinted)
	}
	a.info.SwitchValues[expr] = result
	return result
}

func (a *Analyser) unifyArms(arms []*ast.SwitchArm, armTypes [][]Type, index int, expected []Type, hinted bool) Type {
	var target Type
	if expected != nil {
		target = expected[index]
	} else {
		target = a.commonArmType(arms, armTypes, index)
		if target == nil {
			return InvalidType{}
		}
	}

	for i, arm := range arms {
		if arm.Exit != nil {
			continue
		}
		got := armTypes[i][index]
		if _, bad := got.(InvalidType); bad {
			continue
		}
		if !a.coerceAt(arm.Values, len(armTypes[i]), index, got, target) {
			if hinted {
				a.helpMismatch(a.errorf(arm, "this arm gives %s, expected %s", got, target), got, target)
			} else if expected != nil {
				a.errorf(arm, "expected %s for the handler value, got %s", target, got)
			} else {
				a.errorf(arm, "this arm gives %s, the others give %s", got, target)
			}
		}
	}
	return target
}

func (a *Analyser) commonArmType(arms []*ast.SwitchArm, armTypes [][]Type, index int) Type {
	var target, flexible Type
	for i, arm := range arms {
		if arm.Exit != nil {
			continue
		}
		got := armTypes[i][index]
		if _, bad := got.(InvalidType); bad {
			return InvalidType{}
		}
		if isUntyped(got) || isImplicitEnum(got) {
			if flexible == nil {
				flexible = got
			}
			continue
		}
		if target == nil {
			target = got
			continue
		}
		if merged, ok := commonType(target, got); ok {
			target = merged
		}
	}
	if target != nil {
		return target
	}
	return flexible
}

func switchArms(expr *ast.SwitchExpr) []*ast.SwitchArm {
	arms := make([]*ast.SwitchArm, 0, len(expr.Arms)+1)
	for i := range expr.Arms {
		arms = append(arms, &expr.Arms[i])
	}
	if expr.Default != nil {
		arms = append(arms, expr.Default)
	}
	return arms
}
