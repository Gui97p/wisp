package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

func isDerefTarget(expr ast.Expression) bool {
	u, ok := expr.(*ast.UnaryExpr)
	return ok && u.Operator == "*"
}

func (a *Analyser) checkExpr(expr ast.Expression) Type {
	var t Type

	switch e := expr.(type) {
	case *ast.IntLiteral:
		t = UntypedIntType{}
	case *ast.FloatLiteral:
		t = UntypedFloatType{}
	case *ast.StringLiteral:
		t = PrimitiveType{Name: "string"}
	case *ast.CharLiteral:
		t = PrimitiveType{Name: "char"}
	case *ast.BoolLiteral:
		t = PrimitiveType{Name: "bool"}
	case *ast.IdentLiteral:
		symbol, ok := a.scope.Resolve(e.Value)
		if !ok {
			a.errorf(expr, "identifier %s not declared in this scope", e.Value)
			t = InvalidType{}
			break
		}
		a.info.Idents[e] = symbol
		t = symbol.Type
	case *ast.NullLiteral:
		t = NullType{}
	case *ast.ArrayLiteral:
		t = a.checkArrayLiteral(e)
	case *ast.MapLiteral:
		t = a.checkMapLiteral(e)
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
	case *ast.CastExpr:
		t = a.checkCastExpr(e)
	default:
		t = InvalidType{}
	}

	a.info.Types[expr] = t
	return t
}

func (a *Analyser) checkArrayLiteral(expr *ast.ArrayLiteral) Type {
	if len(expr.Elements) == 0 {
		a.error(expr, "impossible to infer type of a empty array")
		return InvalidType{}
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
			a.errorf(expr, "array index %d expected %s, got %s", i+1, first.String(), t.String())
		}
	}

	return ArrayType{Element: first, Size: int64(len(expr.Elements))}
}

func (a *Analyser) checkMapLiteral(expr *ast.MapLiteral) Type {
	if len(expr.Keys) == 0 {
		a.error(expr, "impossible to infer type of an empty map")
		return InvalidType{}
	}

	keyTypes := make([]Type, len(expr.Keys))
	valueTypes := make([]Type, len(expr.Values))

	for i := range expr.Keys {
		keyTypes[i] = a.checkExpr(expr.Keys[i])
		valueTypes[i] = a.checkExpr(expr.Values[i])
	}

	firstKey, firstValue := keyTypes[0], valueTypes[0]
	if _, ok := firstKey.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := firstValue.(InvalidType); ok {
		return InvalidType{}
	}

	for i := 1; i < len(keyTypes); i++ {
		if _, ok := keyTypes[i].(InvalidType); !ok && !keyTypes[i].Equals(firstKey) {
			a.errorf(expr, "map key %d expected %s, got %s", i+1, firstKey.String(), keyTypes[i].String())
		}
		if _, ok := valueTypes[i].(InvalidType); !ok && !valueTypes[i].Equals(firstValue) {
			a.errorf(expr, "map value %d expected %s, got %s", i+1, firstValue.String(), valueTypes[i].String())
		}
	}

	return &MapType{Key: firstKey, Value: firstValue}
}

func (a *Analyser) checkBinaryExpr(expr *ast.BinaryExpr) Type {
	left := a.checkExpr(expr.Left)
	right := a.checkExpr(expr.Right)

	switch expr.Operator {
	case "+", "-", "*", "/", "%":
		return a.checkArithmetic(expr, expr.Operator, left, right)
	case "==", "!=", ">", ">=", "<", "<=":
		return a.checkComparison(expr, expr.Operator, left, right)
	case "&&", "||":
		return a.checkLogical(expr, expr.Operator, left, right)
	default:
		a.errorf(expr, "unknown operator %s", expr.Operator)
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
			a.errorf(expr, "invalid operator %s for %s", expr.Operator, value.String())
			return InvalidType{}
		}
		return value
	case "!":
		boolType := PrimitiveType{Name: "bool"}
		if !value.Equals(boolType) {
			a.errorf(expr, "expected bool for !, got %s", value.String())
		}
		return boolType
	case "&":
		if !isAddressable(expr.Value) {
			a.error(expr, "cannot obtain address of a temporary expression")
			return InvalidType{}
		}
		return PointerType{Element: value}
	case "*":
		ptr, ok := value.(PointerType)
		if !ok {
			a.errorf(expr, "expected pointer for *, got %s", value.String())
			return InvalidType{}
		}
		return ptr.Element
	default:
		a.errorf(expr, "invalid unary operator: %s", expr.Operator)
		return InvalidType{}
	}
}

func (a *Analyser) checkCallExprValue(expr *ast.CallExpr) Type {
	returns := a.checkCallExpr(expr)

	switch len(returns) {
	case 0:
		a.error(expr, "function with no returns can't be used as value")
		return InvalidType{}
	case 1:
		return returns[0]
	default:
		a.errorf(expr, "functions returns %d values, expected 1 in this context", len(returns))
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
		a.errorf(expr, "%s is not a function or struct", nameType.String())
		a.evalArgTypes(expr)
		return nil
	}
}

func (a *Analyser) checkCallArgs(expr *ast.CallExpr, ft *FuncType) {
	argTypes := a.evalArgTypes(expr)

	if ft.Variadic {
		fixedCount := len(ft.Params) - 1
		if len(argTypes) < fixedCount {
			a.errorf(expr, "function %s expects at least %d arguments, got %d", ft.Name, fixedCount, len(argTypes))
			return
		}
		for i := range fixedCount {
			if _, ok := argTypes[i].(InvalidType); ok {
				continue
			}
			if !argTypes[i].Equals(ft.Params[i]) {
				a.errorf(expr, "argument %d from %s: expected %s, got %s", i+1, ft.Name, ft.Params[i].String(), argTypes[i].String())
			}
		}

		variadicType := ft.Params[len(ft.Params)-1]
		for i := fixedCount; i < len(argTypes); i++ {
			if _, ok := argTypes[i].(InvalidType); ok {
				continue
			}
			if !variadicType.Equals(argTypes[i]) {
				a.errorf(expr, "argument %d from %s: expected %s (variadic), got %s", i+1, ft.Name, variadicType.String(), argTypes[i].String())
			}
		}
		return
	}

	if len(argTypes) != len(ft.Params) {
		a.errorf(expr, "function %s expects %d arguments, got %d", ft.Name, len(ft.Params), len(argTypes))
		return
	}

	for i, at := range argTypes {
		if _, ok := at.(InvalidType); ok {
			continue
		}
		if !at.Equals(ft.Params[i]) {
			a.errorf(expr, "argument %d from %s: expected %s, got %s", i+1, ft.Name, ft.Params[i].String(), at.String())
		}
	}
}

func (a *Analyser) checkExprList(exprs []ast.Expression) []Type {
	var types []Type

	for _, e := range exprs {
		if call, ok := e.(*ast.CallExpr); ok {
			returns := a.checkCallExpr(call)
			if len(returns) == 0 {
				a.error(e, "function doesn't expect any return")
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
		a.errorf(expr, "%s is not a struct", objType.String())
		return InvalidType{}
	}

	fieldType, ok := st.Fields[expr.Field]
	if !ok {
		a.errorf(expr, "invalid field %s in struct %s", expr.Field, st.Name)
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

	switch t := arrType.(type) {
	case ArrayType:
		a.requireNumeric(expr, idxType, "array index")
		return t.Element
	case SpanType:
		a.requireNumeric(expr, idxType, "span index")
		return t.Element
	case *MapType:
		if _, invalid := idxType.(InvalidType); !invalid && !idxType.Equals(t.Key) {
			a.errorf(expr, "map index expected %s, got %s", t.Key.String(), idxType.String())
		}
		return t.Value
	default:
		a.errorf(expr, "%s can't be indexed", arrType.String())
		return InvalidType{}
	}
}

func (a *Analyser) checkTernaryExpr(expr *ast.TernaryExpr) Type {
	condType := a.checkExpr(expr.Condition)
	a.requireBool(expr.Condition, condType, "ternary condition")

	thenType := a.checkExpr(expr.Then)
	elseType := a.checkExpr(expr.Else)

	if _, ok := thenType.(InvalidType); ok {
		return elseType
	}
	if _, ok := elseType.(InvalidType); ok {
		return thenType
	}

	if !thenType.Equals(elseType) {
		a.errorf(expr, "incompatible ternary types (%s & %s)", thenType.String(), elseType.String())
		return InvalidType{}
	}

	return thenType
}

func (a *Analyser) checkCastExpr(expr *ast.CastExpr) Type {
	valueType := a.checkExpr(expr.Value)
	targetType := a.resolveTypeRef(expr, a.scope, expr.Type)
	if targetType == nil {
		return InvalidType{}
	}
	if _, ok := valueType.(InvalidType); ok {
		return targetType
	}

	if isNumeric(valueType) && isNumeric(targetType) {
		return targetType
	}

	if _, srcOk := valueType.(PointerType); srcOk {
		if _, dstOk := targetType.(PointerType); dstOk {
			return targetType
		}
	}

	a.errorf(expr, "cannot cast %s to %s", valueType.String(), targetType.String())
	return InvalidType{}
}
