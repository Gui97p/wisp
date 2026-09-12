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
	case *ast.ArrayLiteral:
		t = a.checkArrayLiteral(e)
	case *ast.BinaryExpr:
		t = a.checkBinaryExpr(e)
	case *ast.UnaryExpr:
		t = a.checkUnaryExpr(e)
	case *ast.CallExpr:
		t = a.checkCallExprValue(e)
	case *ast.MemberExpr:
		t = a.checkMemberExpr(e)
	case *ast.IndexExpr:
		t = a.checkIndexExpr(e)
	case *ast.TernaryExpr:
		t = a.checkTernaryExpr(e)
	default:
		t = InvalidType{}
	}

	a.info.Types[expr] = t
	return t
}

func (a *Analyser) checkArrayLiteral(expr *ast.ArrayLiteral) Type {
	if len(expr.Elements) == 0 {
		a.errorf("impossible to infer type of a empty array")
	}

	elemTypes := make([]Type, len(expr.Elements))
	for i, el := range expr.Elements {
		elemTypes[i] = a.checkExpr(el)
	}

	first := elemTypes[0]
	if _, ok := first.(InvalidType); ok {
		return InvalidType{}
	}

	for i := 1; i < len(elemTypes); i++ {
		t := elemTypes[i]
		if _, ok := t.(InvalidType); ok {
			continue
		}
		if !t.Equals(first) {
			a.errorf("array index %d expected %s, got %s", i+1, first.String(), t.String())
		}
	}

	return ArrayType{Element: first, Size: int64(len(expr.Elements))}
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

func (a *Analyser) checkCallExpr(expr *ast.CallExpr) []Type {
	nameType := a.checkExpr(expr.Name)

	switch ct := nameType.(type) {
	case *FuncType:
		a.checkCallArgs(expr, ct)
		return ct.Returns
	case *StructType:
		return a.checkStructConstruction(expr, ct)
	case *InvalidType:
		a.evalArgTypes(expr)
		return nil
	default:
		a.errorf("%s is not a function or struct", nameType.String())
		a.evalArgTypes(expr)
		return nil
	}
}

func (a *Analyser) checkCallArgs(expr *ast.CallExpr, ft *FuncType) {
	argTypes := a.evalArgTypes(expr)

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

func (a *Analyser) evalArgTypes(expr *ast.CallExpr) []Type {
	types := make([]Type, len(expr.Args))
	for i, arg := range expr.Args {
		types[i] = a.checkExpr(arg)
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

func (a *Analyser) checkMemberExpr(expr *ast.MemberExpr) Type {
	objType := a.checkExpr(expr.Object)

	if _, ok := objType.(InvalidType); ok {
		return InvalidType{}
	}

	if ptr, ok := objType.(PointerType); ok {
		objType = ptr.Element
	}

	st, ok := objType.(*StructType)
	if !ok {
		a.errorf("%s is not a struct", objType.String())
		return InvalidType{}
	}

	fieldType, ok := st.Fields[expr.Field]
	if !ok {
		a.errorf("invalid field %s in struct %s", expr.Field, st.Name)
		return InvalidType{}
	}

	return fieldType
}

func (a *Analyser) checkIndexExpr(expr *ast.IndexExpr) Type {
	arrType := a.checkExpr(expr.Array)
	idxType := a.checkExpr(expr.Index)

	if _, ok := arrType.(InvalidType); ok {
		return InvalidType{}
	}

	at, ok := arrType.(ArrayType)
	if !ok {
		a.errorf("%s can't be indexed", arrType.String())
		return InvalidType{}
	}

	a.requireNumeric(idxType, "array index")

	return at.Element
}

func (a *Analyser) checkTernaryExpr(expr *ast.TernaryExpr) Type {
	condType := a.checkExpr(expr.Condition)
	a.requireBool(condType, "ternary condition")

	thenType := a.checkExpr(expr.Then)
	elseType := a.checkExpr(expr.Else)

	if _, ok := thenType.(InvalidType); ok {
		return elseType
	}
	if _, ok := elseType.(InvalidType); ok {
		return thenType
	}

	if !thenType.Equals(elseType) {
		a.errorf("incompatible ternary types (%s & %s)", thenType.String(), elseType.String())
		return InvalidType{}
	}

	return thenType
}
