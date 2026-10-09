package analyser

import (
	"math/big"
	"sort"
	"strings"

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
			a.suggest(a.errorf(expr, "identifier %s not declared in this scope", e.Value), e.Value, a.scope.names(func(sym *Symbol) bool {
				return sym.Kind == VAR || sym.Kind == CONST || sym.Kind == PARAM || sym.Kind == FUNC || sym.Kind == MODULE
			}))
			t = InvalidType{}
			break
		}
		a.info.Idents[e] = symbol
		symbol.Used = true
		t = symbol.Type
	case *ast.NullLiteral:
		t = NullType{}
	case *ast.FuncLiteral:
		t = a.checkFuncLiteral(e)
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
	case *ast.SliceExpr:
		t = a.checkSliceExpr(e)
	case *ast.TernaryExpr:
		t = a.checkTernaryExpr(e)
	case *ast.CoalesceExpr:
		t = a.checkCoalesceExpr(e)
	case *ast.PropagateExpr:
		t = a.checkPropagateExpr(e)
	case *ast.TupleExpr:
		a.error(e, "a tuple is only valid as the value of ??")
		t = InvalidType{}
	case *ast.CastExpr:
		t = a.info.Types[e]
	case *ast.InExpr:
		t = a.checkInExpr(e)
	case *ast.StructLiteral:
		t = a.checkStructLiteral(e)
	case *ast.DotIdent:
		t = a.checkDotIdent(e)
	default:
		t = InvalidType{}
	}

	a.info.Types[expr] = t
	return t
}

func (a *Analyser) checkFuncLiteral(expr *ast.FuncLiteral) Type {
	params := make([]Type, len(expr.Params))
	for i, p := range expr.Params {
		params[i] = a.resolveTypeRef(expr, a.scope, p.Type)
	}
	returns := make([]Type, len(expr.ReturnTypes))
	for i, r := range expr.ReturnTypes {
		returns[i] = a.resolveTypeRef(expr, a.scope, r)
	}

	line, col := expr.Position()
	a.enterScope()
	for i, p := range expr.Params {
		a.scope.Define(&Symbol{Name: p.Name, Kind: PARAM, Type: params[i], Line: line, Col: col, File: a.currentFile})
	}

	if len(returns) == 0 {
		if ret, ok := singleReturn(expr.Block); ok {
			returns = a.checkExprList(ret.Values)
		}
	}

	prevReturns, prevFallible := a.currentReturns, a.currentFallible
	a.currentReturns, a.currentFallible = returns, expr.Fallible

	a.checkBlock(expr.Block)

	a.currentReturns, a.currentFallible = prevReturns, prevFallible
	a.exitScope()

	if len(returns) > 0 && !blockTerminates(expr.Block) {
		a.error(expr, "missing return at end of lambda")
	}

	return &FuncType{Params: params, Returns: returns, Fallible: expr.Fallible}
}

func singleReturn(block *ast.BlockStmt) (*ast.ReturnStmt, bool) {
	if len(block.Statements) != 1 {
		return nil, false
	}
	ret, ok := block.Statements[0].(*ast.ReturnStmt)
	if !ok || len(ret.Values) == 0 {
		return nil, false
	}
	return ret, true
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

	common := first
	for i := 1; i < len(elemTypes); i++ {
		t := elemTypes[i]
		if _, ok := t.(InvalidType); ok {
			continue
		}
		next, ok := a.commonElement(common, t)
		if !ok {
			a.errorf(expr, "array index %d expected %s, got %s", i+1, common.String(), t.String())
			continue
		}
		common = next
	}

	for i := range expr.Elements {
		if _, ok := elemTypes[i].(InvalidType); !ok {
			a.coerce(&expr.Elements[i], elemTypes[i], common)
		}
	}

	return ArrayType{Element: common, Size: int64(len(expr.Elements))}
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

	commonKey, commonValue := firstKey, firstValue
	for i := 1; i < len(keyTypes); i++ {
		if _, ok := keyTypes[i].(InvalidType); !ok {
			if next, ok := a.commonElement(commonKey, keyTypes[i]); ok {
				commonKey = next
			} else {
				a.errorf(expr, "map key %d expected %s, got %s", i+1, commonKey.String(), keyTypes[i].String())
			}
		}
		if _, ok := valueTypes[i].(InvalidType); !ok {
			if next, ok := a.commonElement(commonValue, valueTypes[i]); ok {
				commonValue = next
			} else {
				a.errorf(expr, "map value %d expected %s, got %s", i+1, commonValue.String(), valueTypes[i].String())
			}
		}
	}

	for i := range expr.Keys {
		if _, ok := keyTypes[i].(InvalidType); !ok {
			a.coerce(&expr.Keys[i], keyTypes[i], commonKey)
		}
		if _, ok := valueTypes[i].(InvalidType); !ok {
			a.coerce(&expr.Values[i], valueTypes[i], commonValue)
		}
	}

	return &MapType{Key: commonKey, Value: commonValue}
}

func (a *Analyser) checkBinaryExpr(expr *ast.BinaryExpr) Type {
	left := a.checkExpr(expr.Left)
	right := a.checkExpr(expr.Right)

	if expr.Operator == "==" || expr.Operator == "!=" {
		if t, handled := a.checkErrorComparison(expr, left, right); handled {
			return t
		}
	}

	switch expr.Operator {
	case "+", "-", "*", "/", "%", "==", "!=", ">", ">=", "<", "<=", "&", "|", "^":
		a.checkBinaryConst(expr.Operator, expr.Left, expr.Right, left, right)
		left, right = a.unifySlots(&expr.Left, &expr.Right, left, right)
		if expr.Operator == "/" || expr.Operator == "%" {
			a.checkDivisor(expr.Right, right)
		}
	case "<<", ">>":
		if neg, mag, ok := intLiteralValue(expr.Right); ok && neg && mag != 0 {
			a.error(expr.Right, "negative shift count")
		}
		a.fixUntyped(expr.Right, PrimitiveType{Name: "int"})
		if isUntyped(right) {
			right = PrimitiveType{Name: "int"}
		}
	}

	switch expr.Operator {
	case "+", "-", "*", "/", "%":
		return a.checkArithmetic(expr, expr.Operator, left, right)
	case "==", "!=", ">", ">=", "<", "<=":
		return a.checkComparison(expr, expr.Operator, left, right)
	case "&", "|", "^", "<<", ">>":
		return a.checkBitwise(expr, expr.Operator, left, right)
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
			a.errorf(expr, "invalid operator %s for %s", expr.Operator, value)
			return InvalidType{}
		}
		return value
	case "~":
		if !isInteger(value) {
			a.errorf(expr, "invalid operator %s for %s", expr.Operator, value)
			return InvalidType{}
		}
		return value
	case "!":
		boolType := PrimitiveType{Name: "bool"}

		if value.Equals(boolType) {
			return boolType
		}

		if _, ok := value.(FallibleType); ok {
			a.error(expr, "a call that can fail is propagated with a postfix !, as in f()!")
			return InvalidType{}
		}

		a.errorf(expr, "expected bool for !, got %s", value.String())
		return InvalidType{}
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
	if a.isErrorConstructor(expr) {
		returns := []Type{a.checkErrorConstructor(expr)}
		a.info.CallReturns[expr] = returns
		return returns
	}

	if target, ok := a.conversionTarget(expr); ok {
		returns := []Type{a.checkConversion(expr, target)}
		a.info.CallReturns[expr] = returns
		return returns
	}

	returns := a.checkCallTypes(expr)
	a.info.CallReturns[expr] = machineReturns(returns)
	return returns
}

func machineReturns(returns []Type) []Type {
	if len(returns) == 1 {
		if ft, ok := returns[0].(FallibleType); ok {
			return append(append([]Type{}, ft.Values...), ErrorType{})
		}
	}
	return returns
}

func callResults(ft *FuncType) []Type {
	if ft.Fallible {
		return []Type{FallibleType{Values: ft.Returns}}
	}
	return ft.Returns
}

func (a *Analyser) checkCallTypes(expr *ast.CallExpr) []Type {
	if member, ok := expr.Name.(*ast.MemberExpr); ok {
		objType := a.checkExpr(member.Object)
		if _, ok := objType.(InvalidType); ok {
			a.evalArgTypes(expr)
			return []Type{InvalidType{}}
		}
		if methods := MethodsOf(objType); methods != nil {
			if ft, ok := methods[member.Field]; ok {
				_, pointerRecv := ft.Receiver.(PointerType)
				_, objectPtr := objType.(PointerType)
				if pointerRecv && !objectPtr && !isAddressable(member.Object) {
					a.errorf(expr, "method %s needs a pointer receiver, but the value is a temporary", member.Field)
				}
				owner, ownerModule, ownerExported := methodOwner(objType)
				if ownerModule != a.module && !(ft.Exported && ownerExported) {
					a.errorf(expr, "method %s of %s is not exported", member.Field, owner)
				}
				a.info.MethodCalls[expr] = &MethodCall{
					Owner:           owner,
					Module:          ownerModule,
					Func:            ft,
					PointerReceiver: pointerRecv,
					ObjectIsPointer: objectPtr,
					Receiver:        a.receiverArgument(member.Object, objType, pointerRecv, objectPtr),
				}
				a.checkCallArgs(expr, ft)
				return callResults(ft)
			}
		}
	}

	nameType := a.checkExpr(expr.Name)

	switch ct := nameType.(type) {
	case *FuncType:
		a.checkCallArgs(expr, ct)
		return callResults(ct)
	case InvalidType:
		a.evalArgTypes(expr)
		return []Type{InvalidType{}}
	default:
		a.errorf(expr, "%s is not a function", nameType.String())
		a.evalArgTypes(expr)
		return nil
	}
}

func (a *Analyser) receiverArgument(object ast.Expression, objType Type, pointerRecv, objectPtr bool) ast.Expression {
	var node *ast.UnaryExpr
	var tp Type

	switch {
	case pointerRecv && !objectPtr:
		node = &ast.UnaryExpr{Operator: "&", Value: object}
		tp = PointerType{Element: objType}
	case !pointerRecv && objectPtr:
		node = &ast.UnaryExpr{Operator: "*", Value: object}
		tp = objType.(PointerType).Element
	default:
		return object
	}

	line, col := object.Position()
	endLine, endCol := object.EndPosition()
	node.SetPos(line, col)
	node.SetEndPos(endLine, endCol)
	a.info.Types[node] = tp
	return node
}

func (a *Analyser) checkCallArgs(expr *ast.CallExpr, ft *FuncType) {
	argTypes := a.evalArgTypes(expr)

	if ft.Name == "len" {
		if len(argTypes) != 1 {
			a.errorf(expr, "len expects 1 argument, got %d", len(argTypes))
			return
		}
		switch t := argTypes[0].(type) {
		case ArrayType, SpanType, InvalidType:
		case PrimitiveType:
			if t.Name != "string" {
				a.errorf(expr, "len expects array, span or string, got %s", argTypes[0])
			}
		default:
			a.errorf(expr, "len expects array, span or string, got %s", argTypes[0])
		}
		return
	}

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
			if !a.coerceAt(expr.Args, len(argTypes), i, argTypes[i], ft.Params[i]) {
				a.errorf(expr, "argument %d from %s: expected %s, got %s", i+1, ft.Name, ft.Params[i].String(), argTypes[i].String())
				continue
			}
			a.checkConstFits(a.valueAt(expr.Args, len(argTypes), i), ft.Params[i])
		}

		variadicType := ft.Params[len(ft.Params)-1]
		for i := fixedCount; i < len(argTypes); i++ {
			if _, ok := argTypes[i].(InvalidType); ok {
				continue
			}
			if _, isAny := variadicType.(AnyType); !isAny && !a.coerceAt(expr.Args, len(argTypes), i, argTypes[i], variadicType) {
				a.errorf(expr, "argument %d from %s: expected %s (variadic), got %s", i+1, ft.Name, variadicType.String(), argTypes[i].String())
				continue
			}
			a.checkConstFits(a.valueAt(expr.Args, len(argTypes), i), variadicType)
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
		if !a.coerceAt(expr.Args, len(argTypes), i, at, ft.Params[i]) {
			a.errorf(expr, "argument %d from %s: expected %s, got %s", i+1, ft.Name, ft.Params[i].String(), at.String())
			continue
		}
		a.checkConstFits(a.valueAt(expr.Args, len(argTypes), i), ft.Params[i])
	}
}

func (a *Analyser) checkExprList(exprs []ast.Expression) []Type {
	var types []Type

	for _, e := range exprs {
		if pe, ok := e.(*ast.PropagateExpr); ok {
			values := a.checkPropagate(pe)
			a.recordValues(pe, values)
			types = append(types, values...)
			continue
		}

		if ce, ok := e.(*ast.CoalesceExpr); ok {
			values := a.checkCoalesce(ce)
			a.recordValues(ce, values)
			types = append(types, values...)
			continue
		}

		if call, ok := e.(*ast.CallExpr); ok {
			returns := a.checkCallExpr(call)
			if len(returns) == 0 {
				a.error(e, "function doesn't expect any return")
				types = append(types, InvalidType{})
				continue
			}
			if len(returns) == 1 {
				a.info.Types[call] = returns[0]
			}
			types = append(types, returns...)
		} else {
			types = append(types, a.checkExpr(e))
		}
	}

	return types
}

func (a *Analyser) recordValues(expr ast.Expression, values []Type) {
	if len(values) > 0 {
		a.info.Types[expr] = values[0]
	} else {
		a.info.Types[expr] = VoidType{}
	}
}

func (a *Analyser) evalArgTypes(expr *ast.CallExpr) []Type {
	types := make([]Type, len(expr.Args))
	for i, arg := range expr.Args {
		types[i] = a.checkExpr(arg)
	}
	return a.rejectFallible(expr.Args, types)
}

func (a *Analyser) checkMemberExpr(expr *ast.MemberExpr) Type {
	if enum, ok := a.enumTypeExpr(expr.Object); ok {
		return a.checkEnumMember(expr, enum)
	}

	objType := a.checkExpr(expr.Object)

	if _, ok := objType.(InvalidType); ok {
		return InvalidType{}
	}

	if mod, ok := objType.(*ModuleType); ok {
		sym, ok := mod.Exports[expr.Field]
		if !ok {
			a.suggest(a.errorf(expr, "module has no exported %s", expr.Field), expr.Field, exportNames(mod.Exports))
			return InvalidType{}
		}
		a.info.Members[expr] = &MemberInfo{Kind: MemberModule}
		return sym.Type
	}

	if _, ok := objType.(ErrorType); ok {
		fieldType, ok := errorFields[expr.Field]
		if !ok {
			names := make([]string, 0, len(errorFields))
			for name := range errorFields {
				names = append(names, name)
			}
			sort.Strings(names)
			d := a.errorf(expr, "Error has no field %s", expr.Field)
			a.suggest(d, expr.Field, names)
			if d.Help == "" {
				d.WithHelp("the fields of Error are %s", strings.Join(names, ", "))
			}
			return InvalidType{}
		}
		a.info.Members[expr] = &MemberInfo{Kind: MemberErrorField}
		return fieldType
	}

	byPointer := false
	if ptr, ok := objType.(PointerType); ok {
		objType = ptr.Element
		byPointer = true
	}

	st, ok := objType.(*StructType)
	if !ok {
		a.errorf(expr, "%s is not a struct", objType.String())
		return InvalidType{}
	}

	fieldType, ok := st.Fields[expr.Field]
	if !ok {
		a.suggest(a.errorf(expr, "invalid field %s in struct %s", expr.Field, st.Name), expr.Field, st.Order)
		return InvalidType{}
	}

	a.info.Members[expr] = &MemberInfo{Kind: MemberField, Struct: st, ByPointer: byPointer}
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
		a.checkConstIndex(expr.Index, t.Size)
		return t.Element
	case SpanType:
		a.requireNumeric(expr, idxType, "span index")
		a.checkConstIndex(expr.Index, -1)
		return t.Element
	case *MapType:
		if _, invalid := idxType.(InvalidType); !invalid && !idxType.Equals(t.Key) {
			a.errorf(expr, "map index expected %s, got %s", t.Key.String(), idxType.String())
		}
		return t.Value
	case PrimitiveType:
		if t.Name != "string" {
			a.errorf(expr, "%s can't be indexed", arrType.String())
			return InvalidType{}
		}
		a.requireNumeric(expr, idxType, "string index")
		a.checkConstIndex(expr.Index, -1)
		return PrimitiveType{Name: "char"}
	default:
		a.errorf(expr, "%s can't be indexed", arrType.String())
		return InvalidType{}
	}
}

func (a *Analyser) checkSliceExpr(expr *ast.SliceExpr) Type {
	arrType := a.checkExpr(expr.Array)
	if expr.Start != nil {
		a.requireNumeric(expr.Start, a.checkExpr(expr.Start), "slice start")
	}
	if expr.End != nil {
		a.requireNumeric(expr.End, a.checkExpr(expr.End), "slice end")
	}

	switch t := arrType.(type) {
	case ArrayType:
		a.checkConstSlice(expr, t.Size)
		return SpanType{Element: t.Element}
	case SpanType:
		a.checkConstSlice(expr, -1)
		return t
	case PrimitiveType:
		if t.Name == "string" {
			return t
		}
	case InvalidType:
		return InvalidType{}
	}

	a.errorf(expr, "%s can't be sliced", arrType)
	return InvalidType{}
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

	thenType, elseType = a.unifySlots(&expr.Then, &expr.Else, thenType, elseType)
	if !thenType.Equals(elseType) {
		a.errorf(expr, "incompatible ternary types (%s & %s)", thenType.String(), elseType.String())
		return InvalidType{}
	}

	return thenType
}

func (a *Analyser) checkPropagateExpr(expr *ast.PropagateExpr) Type {
	values := a.checkPropagate(expr)
	switch len(values) {
	case 1:
		return values[0]
	case 0:
		a.error(expr, "this call has no value to use")
	default:
		a.errorf(expr, "call yields %d values, expected 1 in this context", len(values))
	}
	return InvalidType{}
}

func (a *Analyser) checkPropagate(expr *ast.PropagateExpr) []Type {
	operand := a.checkExpr(expr.Value)
	if _, bad := operand.(InvalidType); bad {
		return []Type{InvalidType{}}
	}

	ft, ok := operand.(FallibleType)
	if !ok {
		a.errorf(expr, "! expects a call that can fail, got %s", operand.String())
		return []Type{InvalidType{}}
	}

	if !a.currentFallible {
		a.error(expr, "cannot propagate outside a fallible function: mark it with !, or handle the error with ?? or let v, e = ...")
		return []Type{InvalidType{}}
	}

	return ft.Values
}

func (a *Analyser) defineErrorBind(expr *ast.CoalesceExpr) {
	line, col := expr.Position()
	sym := &Symbol{Name: expr.ErrorBind, Kind: VAR, Type: ErrorType{}, Line: line, Col: col, File: a.currentFile, Used: true}
	a.scope.Define(sym)
	a.info.VarSymbols[expr] = []*Symbol{sym}
	a.info.VarTypes[expr] = []Type{ErrorType{}}
}

func (a *Analyser) checkCoalesceExpr(expr *ast.CoalesceExpr) Type {
	values := a.checkCoalesce(expr)
	switch len(values) {
	case 1:
		return values[0]
	case 0:
		a.error(expr, "this call has no value to use")
	default:
		a.errorf(expr, "call yields %d values, expected 1 in this context", len(values))
	}
	return InvalidType{}
}

func (a *Analyser) checkCoalesce(expr *ast.CoalesceExpr) []Type {
	invalid := []Type{InvalidType{}}

	left := a.checkExpr(expr.Left)
	if _, bad := left.(InvalidType); bad {
		return invalid
	}

	ft, ok := left.(FallibleType)
	if !ok {
		a.errorf(expr, "?? can only be used on a call that can fail, got %s", left)
		return invalid
	}

	if expr.Block != nil {
		if !a.checkHandlerBlock(expr) {
			return invalid
		}
		return ft.Values
	}

	if len(ft.Values) == 0 {
		a.error(expr, "?? on a call without a value needs a handler block")
		return invalid
	}
	if !a.checkDefaults(expr, ft.Values) {
		return invalid
	}
	return ft.Values
}

func (a *Analyser) checkHandlerBlock(expr *ast.CoalesceExpr) bool {
	b, ok := expr.Block.(*ast.BlockStmt)
	if !ok {
		a.errorf(expr, "expected valid block for coalesce")
		return false
	}
	if !a.handlerOK[expr] {
		a.error(expr, "a handler block can only be the value of a let, an assignment, a return or a statement")
	}

	a.enterScope()
	a.defineErrorBind(expr)
	a.checkBlock(b)
	a.exitScope()

	if !blockDiverges(b) {
		a.error(expr, "the handler block must end with return, break or continue")
		return false
	}
	return true
}

func (a *Analyser) checkDefaults(expr *ast.CoalesceExpr, values []Type) bool {
	if expr.Default == nil {
		a.error(expr, "expected default value or block in coalesce")
		return false
	}

	slots := []*ast.Expression{&expr.Default}
	if tuple, isTuple := expr.Default.(*ast.TupleExpr); isTuple {
		slots = make([]*ast.Expression, len(tuple.Elements))
		for i := range tuple.Elements {
			slots[i] = &tuple.Elements[i]
		}
	}

	if len(slots) != len(values) {
		if len(values) > 1 {
			a.errorf(expr, "?? needs %d values for this call, written as a tuple (a, b), got %d", len(values), len(slots))
		} else {
			a.errorf(expr, "?? needs 1 value for this call, got %d", len(slots))
		}
		return false
	}

	if expr.ErrorBind != "" {
		a.enterScope()
		a.defineErrorBind(expr)
		defer a.exitScope()
	}

	valid := true
	for i, slot := range slots {
		got := a.checkExpr(*slot)
		if !a.coerce(slot, got, values[i]) {
			if expr.ErrorBind != "" {
				a.errorf(expr, "expected %s for the handler value, got %s", values[i], got)
			} else {
				a.errorf(expr, "expected %s for default value, got %s", values[i], got)
			}
			valid = false
		}
	}
	return valid
}

func blockDiverges(block *ast.BlockStmt) bool {
	if blockTerminates(block) {
		return true
	}
	if len(block.Statements) == 0 {
		return false
	}
	switch block.Statements[len(block.Statements)-1].(type) {
	case *ast.BreakStmt, *ast.ContinueStmt:
		return true
	}
	return false
}

func (a *Analyser) rejectFallible(exprs []ast.Expression, types []Type) []Type {
	for i, t := range types {
		ft, ok := t.(FallibleType)
		if !ok {
			continue
		}
		var node ast.Node = exprs[0]
		if len(exprs) == len(types) {
			node = exprs[i]
		}
		a.errorf(node, "unhandled error: the call can fail (%s); use !, ??, or bind the error with let v, e = ...", ft.String())
		types[i] = InvalidType{}
	}
	return types
}

func (a *Analyser) expandFallible(values []ast.Expression, types []Type, want int) ([]Type, bool) {
	if len(values) != 1 || len(types) != 1 {
		return types, false
	}
	ft, ok := types[0].(FallibleType)
	if !ok || want != len(ft.Values)+1 {
		return types, false
	}
	return append(append([]Type{}, ft.Values...), ErrorType{}), true
}

func (a *Analyser) isErrorConstructor(expr *ast.CallExpr) bool {
	id, ok := expr.Name.(*ast.IdentLiteral)
	if !ok || id.Value != "Error" {
		return false
	}
	sym, found := a.scope.Resolve("Error")
	if !found {
		return false
	}
	_, isError := sym.Type.(ErrorType)
	return isError
}

func (a *Analyser) checkErrorConstructor(expr *ast.CallExpr) Type {
	if id, ok := expr.Name.(*ast.IdentLiteral); ok {
		if sym, found := a.scope.Resolve("Error"); found {
			a.info.Idents[id] = sym
		}
	}

	if len(expr.Args) < 1 || len(expr.Args) > 2 {
		a.error(expr, "Error expects an enum member and an optional message")
		a.evalArgTypes(expr)
		return ErrorType{}
	}

	codeType := a.checkExpr(expr.Args[0])
	if dot, isDot := codeType.(ImplicitEnumType); isDot {
		a.dropDot(expr.Args[0])
		a.errorf(expr.Args[0], "cannot infer the enum of .%s inside Error: write Enum.%s", dot.Name, dot.Name)
		return ErrorType{}
	}
	enum, ok := codeType.(NamedType)
	if _, bad := codeType.(InvalidType); bad {
		return ErrorType{}
	}
	if !ok || !enum.Enum {
		a.errorf(expr.Args[0], "the first argument of Error must be an enum member, got %s", codeType.String())
		return ErrorType{}
	}

	code, isConst := a.evalConst(expr.Args[0])
	if !isConst || code.Kind != ConstInt {
		a.error(expr.Args[0], "the first argument of Error must be a constant enum member")
		return ErrorType{}
	}

	member := enum.Name
	if value, isMember := a.info.EnumValues[expr.Args[0]]; isMember {
		member = value.Name
	}
	message := enum.Name + "." + member

	if len(expr.Args) == 2 {
		messageType := a.checkExpr(expr.Args[1])
		if _, bad := messageType.(InvalidType); !bad {
			text, isConst := a.evalConst(expr.Args[1])
			if !messageType.Equals(PrimitiveType{Name: "string"}) || !isConst || text.Kind != ConstString {
				a.error(expr.Args[1], "the message of Error must be a constant string")
				return ErrorType{}
			}
			message = text.Str
		}
	}

	line, _ := expr.Position()
	a.info.ErrorLiterals[expr] = &ErrorLiteral{
		Domain:  enum.Module + "." + enum.Name,
		Code:    code.Bits(),
		Message: message,
		File:    a.currentFile,
		Line:    line,
	}
	return ErrorType{}
}

func (a *Analyser) checkConstIndex(index ast.Expression, size int64) {
	value, ok := a.evalConst(index)
	if !ok || value.Kind != ConstInt {
		return
	}
	if value.Int.Sign() < 0 {
		a.error(index, "index out of range: negative index")
		return
	}
	if size >= 0 && value.Int.Cmp(big.NewInt(size)) >= 0 {
		a.errorf(index, "index out of range: %s is not below the array size %d", value.Int, size)
	}
}

func (a *Analyser) checkConstSlice(expr *ast.SliceExpr, size int64) {
	var start, end *big.Int
	for i, bound := range []ast.Expression{expr.Start, expr.End} {
		if bound == nil {
			continue
		}
		value, ok := a.evalConst(bound)
		if !ok || value.Kind != ConstInt {
			continue
		}
		if value.Int.Sign() < 0 {
			a.error(bound, "slice out of range: negative bound")
			return
		}
		if size >= 0 && value.Int.Cmp(big.NewInt(size)) > 0 {
			a.errorf(bound, "slice out of range: %s is above the array size %d", value.Int, size)
			return
		}
		if i == 0 {
			start = value.Int
		} else {
			end = value.Int
		}
	}
	if start != nil && end != nil && start.Cmp(end) > 0 {
		a.error(expr.End, "slice out of range: the start is above the end")
	}
}

func (a *Analyser) checkDivisor(divisor ast.Expression, tp Type) {
	if !isInteger(tp) {
		return
	}
	if value, ok := a.evalConst(divisor); ok && value.Kind == ConstInt && value.Int.Sign() == 0 {
		a.error(divisor, "division by zero")
	}
}

func (a *Analyser) isErrorField(expr ast.Expression) bool {
	member, ok := expr.(*ast.MemberExpr)
	if !ok {
		return false
	}
	_, isError := a.info.Types[member.Object].(ErrorType)
	return isError
}

func (a *Analyser) checkErrorComparison(expr *ast.BinaryExpr, left, right Type) (Type, bool) {
	_, leftErr := left.(ErrorType)
	_, rightErr := right.(ErrorType)
	if !leftErr && !rightErr {
		return nil, false
	}

	boolType := PrimitiveType{Name: "bool"}
	if leftErr && rightErr {
		a.error(expr, "two errors cannot be compared: compare an Error with null or with an enum member")
		return boolType, true
	}

	other := right
	if rightErr {
		other = left
	}
	a.retypeNull(expr.Left, left, right)
	a.retypeNull(expr.Right, right, left)

	switch o := other.(type) {
	case ImplicitEnumType:
		a.dropDot(expr.Left)
		a.dropDot(expr.Right)
		a.errorf(expr, "cannot infer the enum of .%s when comparing an Error: write Enum.%s", o.Name, o.Name)
		return boolType, true
	case NullType, InvalidType:
		return boolType, true
	case NamedType:
		if o.Enum {
			return boolType, true
		}
	}

	a.errorf(expr, "cannot compare Error with %s", other.String())
	return boolType, true
}

func (a *Analyser) checkInExpr(expr *ast.InExpr) Type {
	left := a.checkExpr(expr.Left)
	right := a.checkExpr(expr.Right)
	boolType := PrimitiveType{Name: "bool"}

	if _, ok := left.(InvalidType); ok {
		return boolType
	}
	if _, ok := right.(InvalidType); ok {
		return boolType
	}

	switch rt := right.(type) {
	case ArrayType:
		if !a.coerce(&expr.Left, left, rt.Element) {
			a.errorf(expr, "'in' expects %s, got %s", rt.Element, left)
		}
		a.checkConstFits(expr.Left, rt.Element)
	case SpanType:
		if !a.coerce(&expr.Left, left, rt.Element) {
			a.errorf(expr, "'in' expects %s, got %s", rt.Element, left)
		}
		a.checkConstFits(expr.Left, rt.Element)
	case *MapType:
		if !a.coerce(&expr.Left, left, rt.Key) {
			a.errorf(expr, "'in' expects %s, got %s", rt.Key, left)
		}
		a.checkConstFits(expr.Left, rt.Key)
	case PrimitiveType:
		if rt.Name != "string" {
			a.errorf(expr, "'in' not supported for %s", right)
			break
		}
		if _, untyped := left.(UntypedIntType); untyped || (!isChar(left) && !isString(left)) {
			a.errorf(expr, "'in' expects char or string, got %s", left)
		}
	default:
		a.errorf(expr, "'in' not supported for %s", right)
	}

	return boolType
}
