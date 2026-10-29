package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkSwitchExprValue(expr *ast.SwitchExpr) Type {
	values := a.checkSwitchExpr(expr, nil)
	switch len(values) {
	case 1:
		return values[0]
	case 0:
		a.error(expr, "this switch gives no value").
			WithHelp("at least one case has to give a value")
	default:
		a.errorf(expr, "switch gives %d values, expected 1 in this context", len(values))
	}
	return InvalidType{}
}

func (a *Analyser) checkSwitchExpr(expr *ast.SwitchExpr, expected []Type) []Type {
	a.enterScope()
	defer a.exitScope()

	for _, stmt := range expr.Prelude {
		a.checkStmt(stmt)
	}

	if expr.Default == nil {
		a.error(expr, "a switch expression needs a default")
		return []Type{InvalidType{}}
	}

	arms := make([]*ast.SwitchArm, 0, len(expr.Arms)+1)
	for i := range expr.Arms {
		arms = append(arms, &expr.Arms[i])
	}
	arms = append(arms, expr.Default)

	armTypes := make([][]Type, len(arms))
	count, valid := -1, true
	for i, arm := range arms {
		a.enterScope()
		if arm.Condition != nil {
			condType := a.checkExpr(arm.Condition)
			a.requireBool(arm.Condition, condType, "case condition")
			a.narrowSwitchVariable(arm.Condition)
		}

		if arm.Exit != nil {
			a.checkStmt(arm.Exit)
		} else {
			armTypes[i] = a.rejectFallible(arm.Values, a.checkExprList(arm.Values))
			switch {
			case count == -1:
				count = len(armTypes[i])
			case len(armTypes[i]) != count:
				a.errorf(arm, "this case gives %d values, the others give %d", len(armTypes[i]), count)
				valid = false
			}
		}
		a.exitScope()
	}

	if !valid {
		return []Type{InvalidType{}}
	}

	if count == -1 {
		if expected == nil {
			return nil
		}
		a.info.SwitchValues[expr] = nil
		return nil
	}

	if expected != nil && len(expected) != count {
		a.errorf(expr, "this switch gives %d values, the call gives %d", count, len(expected))
		return []Type{InvalidType{}}
	}

	result := make([]Type, count)
	for i := range result {
		result[i] = a.unifyArms(arms, armTypes, i, expected)
	}
	a.info.SwitchValues[expr] = result
	return result
}

func (a *Analyser) unifyArms(arms []*ast.SwitchArm, armTypes [][]Type, index int, expected []Type) Type {
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
			if expected != nil {
				a.errorf(arm, "expected %s for the handler value, got %s", target, got)
			} else {
				a.errorf(arm, "this case gives %s, the others give %s", got, target)
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
