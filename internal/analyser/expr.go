package analyser

import "github.com/Gui97p/wisp/internal/ast"

func isDerefTarget(expr ast.Expression) bool {
	u, ok := expr.(*ast.UnaryExpr)
	return ok && u.Operator == "*"
}

func (a *Analyser) checkExpr(expr ast.Expression) Type {
	var t Type

	switch e := expr.(type) {
	case *ast.IntLiteral:
		t = PrimitiveType{Name: "int"}
	case *ast.FloatLiteral:
		t = PrimitiveType{Name: "float64"}
	case *ast.StringLiteral:
		t = PrimitiveType{Name: "string"}
	case *ast.CharLiteral:
		t = PrimitiveType{Name: "char"}
	case *ast.BoolLiteral:
		t = PrimitiveType{Name: "bool"}
	case *ast.IdentLiteral:
		symbol, ok := a.scope.Resolve(e.Value)
		if !ok {
			a.errorf("identifier %s not declared in this scope", e.Value)
			t = InvalidType{}
			break
		}
		a.info.Idents[e] = symbol
		t = symbol.Type
	case *ast.BinaryExpr:
		t = a.checkBinaryExpr(e)
	case *ast.UnaryExpr:
		t = a.checkUnaryExpr(e)
	case *ast.CallExpr:
		t = a.checkCallExprValue(e)
	default:
		t = InvalidType{}
	}

	a.info.Types[expr] = t
	return t
}

func (a *Analyser) checkBinaryExpr(expr *ast.BinaryExpr) Type {
	left := a.checkExpr(expr.Left)
	right := a.checkExpr(expr.Right)

	switch expr.Operator {
	case "+", "-", "*", "/", "%":
		return a.checkArithmetic(expr.Operator, left, right)
	case "==", "!=", ">", ">=", "<", "<=":
		return a.checkComparison(expr.Operator, left, right)
	case "&&", "||":
		return a.checkLogical(expr.Operator, left, right)
	default:
		a.errorf("unknown operator %s", expr.Operator)
		return InvalidType{}
	}
}

func (a *Analyser) checkUnaryExpr(expr *ast.UnaryExpr) Type {
	value := a.checkExpr(expr.Value)

	if _, ok := value.(InvalidType); ok {
		return InvalidType{}
	}

	switch expr.Operator {
	case "-":
		if !isNumeric(value) {
			a.errorf("invalid operator %s for %s", expr.Operator, value.String())
			return InvalidType{}
		}
		return value
	case "!":
		boolType := PrimitiveType{Name: "bool"}
		if !value.Equals(boolType) {
			a.errorf("expected bool for !, got %s", value.String())
		}
		return boolType
	case "&":
		if !isAddressable(expr.Value) {
			a.errorf("not possible to obtain address from a temporary expression")
			return InvalidType{}
		}
		return PointerType{Element: value}
	case "*":
		ptr, ok := value.(PointerType)
		if !ok {
			a.errorf("expected pointer for *, got %s", value.String())
			return InvalidType{}
		}
		return ptr.Element
	default:
		a.errorf("invalid unary operator: %s", expr.Operator)
		return InvalidType{}
	}
}

func (a *Analyser) checkCallExprValue(e *ast.CallExpr) Type {
	returns := a.checkCallExpr(e)

	switch len(returns) {
	case 0:
		a.errorf("function with no returns can't be used as value")
		return InvalidType{}
	case 1:
		return returns[0]
	default:
		a.errorf("functions returns %d values, expected 1 in this context", len(returns))
		return InvalidType{}
	}
}

func (a *Analyser) checkCallExpr(e *ast.CallExpr) []Type {
	nameType := a.checkExpr(e.Name)

	ft, ok := nameType.(*FuncType)
	if !ok {
		if _, invalid := nameType.(InvalidType); !invalid {
			a.errorf("%s is not a function", nameType.String())
		}
		a.checkCallArgs(e, nil)
		return nil
	}

	a.checkCallArgs(e, ft)
	return ft.Returns
}

func (a *Analyser) checkCallArgs(e *ast.CallExpr, ft *FuncType) {
	argTypes := make([]Type, len(e.Args))
	for i, arg := range e.Args {
		argTypes[i] = a.checkExpr(arg)
	}

	if ft == nil {
		return
	}

	if len(argTypes) != len(ft.Params) {
		a.errorf("function %s expects %d args, got %d", ft.Name, len(ft.Params), len(argTypes))
		return
	}

	for i, at := range argTypes {
		if _, ok := at.(InvalidType); ok {
			continue
		}
		if !at.Equals(ft.Params[i]) {
			a.errorf("arg %d from %s: expected %s, got %s", i+1, ft.Name, ft.Params[i].String(), at.String())
		}
	}
}

func (a *Analyser) checkExprList(exprs []ast.Expression) []Type {
	var types []Type

	for _, e := range exprs {
		if call, ok := e.(*ast.CallExpr); ok {
			returns := a.checkCallExpr(call)
			if len(returns) == 0 {
				a.errorf("function doesn't expect any return")
				types = append(types, InvalidType{})
				continue
			}
			types = append(types, returns...)
		} else {
			types = append(types, a.checkExpr(e))
		}
	}

	return types
}

func (a *Analyser) checkArithmetic(op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !isNumeric(left) || !isNumeric(right) {
		a.errorf("invalid operator %s for %s and %s", op, left.String(), right.String())
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf("incompatible types: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	return left
}

func (a *Analyser) checkComparison(op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf("tipos incompatíveis: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	return PrimitiveType{Name: "bool"}
}

func (a *Analyser) checkLogical(op string, left, right Type) Type {
	boolType := PrimitiveType{Name: "bool"}

	if _, ok := left.(InvalidType); !ok && !left.Equals(boolType) {
		a.errorf("operador %s espera bool, recebeu %s", op, left.String())
	}
	if _, ok := right.(InvalidType); !ok && !right.Equals(boolType) {
		a.errorf("operador %s espera bool, recebeu %s", op, right.String())
	}

	return boolType
}
