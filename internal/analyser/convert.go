package analyser

import (
	"fmt"
	"sort"

	"github.com/Gui97p/wisp/internal/ast"
)

func isUntyped(t Type) bool {
	switch t.(type) {
	case UntypedIntType, UntypedFloatType:
		return true
	}
	return false
}

func containsUntyped(t Type) bool {
	switch v := t.(type) {
	case UntypedIntType, UntypedFloatType:
		return true
	case ArrayType:
		return containsUntyped(v.Element)
	case SpanType:
		return containsUntyped(v.Element)
	case *MapType:
		return containsUntyped(v.Key) || containsUntyped(v.Value)
	}
	return false
}

func defaultType(t Type) Type {
	switch v := t.(type) {
	case UntypedIntType:
		return PrimitiveType{Name: "int"}
	case UntypedFloatType:
		return PrimitiveType{Name: "float"}
	case ArrayType:
		v.Element = defaultType(v.Element)
		return v
	case SpanType:
		v.Element = defaultType(v.Element)
		return v
	case *MapType:
		return &MapType{Key: defaultType(v.Key), Value: defaultType(v.Value)}
	}
	return t
}

func underlying(t Type) Type {
	if nt, ok := t.(NamedType); ok {
		return nt.Underlying
	}
	return t
}

func numericLike(t Type) bool {
	return isNumeric(underlying(t))
}

func aliasPair(a, b string) bool {
	return (a == "int" && b == "int64") || (a == "int64" && b == "int") ||
		(a == "uint" && b == "uint64") || (a == "uint64" && b == "uint")
}

func LosslessWidening(from, to Type) bool {
	fp, ok1 := from.(PrimitiveType)
	tp, ok2 := to.(PrimitiveType)
	if !ok1 || !ok2 || fp.Name == tp.Name {
		return false
	}
	if fp.Name == "float32" && (tp.Name == "float64" || tp.Name == "float") {
		return true
	}
	if (fp.Name == "float" && tp.Name == "float64") || (fp.Name == "float64" && tp.Name == "float") {
		return true
	}

	fb, fs, fok := integerRange(fp)
	tb, ts, tok := integerRange(tp)
	if !fok || !tok {
		return false
	}

	switch {
	case fs == ts:
		return tb > fb || (tb == fb && aliasPair(fp.Name, tp.Name))
	case !fs && ts:
		return tb > fb
	}
	return false
}

func enumUnderlying(t Type) (Type, bool) {
	if nt, ok := t.(NamedType); ok && nt.Enum {
		return nt.Underlying, true
	}
	return nil, false
}

func implicitConversion(from, to Type) bool {
	if LosslessWidening(from, to) {
		return true
	}

	if u, ok := enumUnderlying(from); ok {
		if _, named := to.(NamedType); named {
			return false
		}
		return u.Equals(to) || LosslessWidening(u, to)
	}

	fp, ok1 := from.(PointerType)
	tp, ok2 := to.(PointerType)
	if ok1 && ok2 {
		_, fromVoid := fp.Element.(VoidType)
		_, toVoid := tp.Element.(VoidType)
		return fromVoid || toVoid
	}

	return false
}

func commonType(l, r Type) (Type, bool) {
	if l.Equals(r) && r.Equals(l) {
		return l, true
	}
	if implicitConversion(l, r) {
		return r, true
	}
	if implicitConversion(r, l) {
		return l, true
	}

	lp, lok := l.(PrimitiveType)
	rp, rok := r.(PrimitiveType)
	if !lok || !rok {
		return nil, false
	}

	lb, ls, ok1 := integerRange(lp)
	rb, rs, ok2 := integerRange(rp)
	if !ok1 || !ok2 || ls == rs {
		return nil, false
	}

	signedBits, unsignedBits := lb, rb
	if !ls {
		signedBits, unsignedBits = rb, lb
	}

	width := 8
	for width <= unsignedBits || width < signedBits {
		width *= 2
	}
	if width > 64 {
		return nil, false
	}

	name := fmt.Sprintf("int%d", width)
	if width == 64 && (lp.Name == "int" || rp.Name == "int") {
		name = "int"
	}
	return PrimitiveType{Name: name}, true
}

func (a *Analyser) fixUntyped(expr ast.Expression, target Type) {
	current, ok := a.info.Types[expr]
	if !ok || !isUntyped(current) || isUntyped(target) {
		return
	}
	if _, isAny := target.(AnyType); isAny {
		return
	}
	if !isNumeric(underlying(target)) {
		return
	}

	a.info.Types[expr] = target

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		switch e.Operator {
		case "+", "-", "*", "/", "%", "&", "|", "^":
			a.fixUntyped(e.Left, target)
			a.fixUntyped(e.Right, target)
		case "<<", ">>":
			a.fixUntyped(e.Left, target)
		}
	case *ast.UnaryExpr:
		switch e.Operator {
		case "-", "+", "~":
			a.fixUntyped(e.Value, target)
		}
	case *ast.TernaryExpr:
		a.fixUntyped(e.Then, target)
		a.fixUntyped(e.Else, target)
	case *ast.SwitchExpr:
		for _, arm := range switchArms(e) {
			if len(arm.Values) == 1 {
				a.fixUntyped(arm.Values[0], target)
			}
		}
		if values := a.info.SwitchValues[e]; len(values) == 1 {
			values[0] = target
		}
	}
}

func (a *Analyser) insertCast(slot *ast.Expression, to Type) {
	inner := *slot
	cast := &ast.CastExpr{Value: inner, Implicit: true}

	line, col := inner.Position()
	cast.SetPos(line, col)
	endLine, endCol := inner.EndPosition()
	cast.SetEndPos(endLine, endCol)

	a.info.Types[cast] = to
	*slot = cast
}

func (a *Analyser) insertSlice(slot *ast.Expression, to SpanType) {
	inner := *slot
	slice := &ast.SliceExpr{Array: inner}

	line, col := inner.Position()
	slice.SetPos(line, col)
	endLine, endCol := inner.EndPosition()
	slice.SetEndPos(endLine, endCol)

	a.info.Types[slice] = to
	*slot = slice
}

func (a *Analyser) coerce(slot *ast.Expression, from, to Type) bool {
	if _, bad := from.(InvalidType); bad {
		return true
	}
	if _, bad := to.(InvalidType); bad {
		return true
	}

	if a.retypeNull(*slot, from, to) {
		return true
	}

	if a.retypeEmpty(*slot, from, to) {
		return true
	}

	if isImplicitEnum(from) {
		return a.resolveDot(*slot, to)
	}

	if voidPointerConversion(from, to) {
		a.insertCast(slot, to)
		return true
	}

	if from.Equals(to) {
		if isUntyped(from) {
			a.fixUntyped(*slot, to)
		}
		return true
	}

	if !isUntyped(from) && implicitConversion(from, to) {
		a.insertCast(slot, to)
		return true
	}

	if at, isArray := from.(ArrayType); isArray {
		if st, isSpan := to.(SpanType); isSpan && at.Element.Equals(st.Element) && sliceable(*slot) {
			a.insertSlice(slot, st)
			return true
		}
	}

	return false
}

func (a *Analyser) valueSlot(values []ast.Expression, n, i int) *ast.Expression {
	if len(values) != n || i >= len(values) {
		return nil
	}
	return &values[i]
}

func (a *Analyser) coerceAt(values []ast.Expression, n, i int, from, to Type) bool {
	slot := a.valueSlot(values, n, i)
	if slot == nil {
		return from.Equals(to)
	}
	return a.coerce(slot, from, to)
}

func (a *Analyser) fixArrayLiteral(expr ast.Expression, elem Type, size int64) {
	lit, ok := expr.(*ast.ArrayLiteral)
	if !ok {
		return
	}

	for i := range lit.Elements {
		et := a.info.Types[lit.Elements[i]]
		switch inner := elem.(type) {
		case ArrayType:
			a.fixArrayLiteral(lit.Elements[i], inner.Element, inner.Size)
		case SpanType:
			a.fixArrayLiteral(lit.Elements[i], inner.Element, -1)
		}
		a.coerce(&lit.Elements[i], et, elem)
	}

	if at, isArray := a.info.Types[lit].(ArrayType); isArray {
		at.Element = elem
		if size >= 0 {
			at.Size = size
		}
		a.info.Types[lit] = at
	}
}

func (a *Analyser) retypeNull(expr ast.Expression, from, to Type) bool {
	if _, isNull := from.(NullType); !isNull {
		return false
	}
	lit, ok := expr.(*ast.NullLiteral)
	if !ok {
		return false
	}
	switch to.(type) {
	case PointerType, ErrorType:
		a.info.Types[lit] = to
		return true
	}
	return false
}

func (a *Analyser) unifySlots(left, right *ast.Expression, lt, rt Type) (Type, Type) {
	if isImplicitEnum(lt) && isEnumType(rt) {
		a.resolveDot(*left, rt)
		return rt, rt
	}
	if isImplicitEnum(rt) && isEnumType(lt) {
		a.resolveDot(*right, lt)
		return lt, lt
	}
	if a.retypeNull(*left, lt, rt) {
		return rt, rt
	}
	if a.retypeNull(*right, rt, lt) {
		return lt, lt
	}
	if !numericLike(lt) || !numericLike(rt) {
		return lt, rt
	}

	lu, ru := isUntyped(lt), isUntyped(rt)
	switch {
	case lu && ru:
		return lt, rt
	case lu:
		if lt.Equals(rt) {
			a.fixUntyped(*left, rt)
			return rt, rt
		}
		return lt, rt
	case ru:
		if rt.Equals(lt) {
			a.fixUntyped(*right, lt)
			return lt, lt
		}
		return lt, rt
	}

	if lt.Equals(rt) && rt.Equals(lt) {
		return lt, rt
	}

	common, ok := commonType(lt, rt)
	if !ok {
		return lt, rt
	}
	if !a.coerce(left, lt, common) || !a.coerce(right, rt, common) {
		return lt, rt
	}
	return common, common
}

func (a *Analyser) commonElement(current, next Type) (Type, bool) {
	switch {
	case isImplicitEnum(current) && isImplicitEnum(next):
		return current, true
	case isImplicitEnum(current) && isEnumType(next):
		return next, true
	case isImplicitEnum(next) && isEnumType(current):
		return current, true
	}
	if isUntyped(current) && isUntyped(next) {
		return typedSide(current, next), true
	}
	if isUntyped(current) {
		if current.Equals(next) {
			return next, true
		}
		return nil, false
	}
	if isUntyped(next) {
		if next.Equals(current) {
			return current, true
		}
		return nil, false
	}
	if current.Equals(next) && next.Equals(current) {
		return current, true
	}
	if numericLike(current) && numericLike(next) {
		return commonType(current, next)
	}
	return nil, false
}

func (a *Analyser) fixMapLiteral(expr ast.Expression, key, value Type) {
	lit, ok := expr.(*ast.MapLiteral)
	if !ok {
		return
	}

	for i := range lit.Keys {
		a.coerce(&lit.Keys[i], a.info.Types[lit.Keys[i]], key)
		a.coerce(&lit.Values[i], a.info.Types[lit.Values[i]], value)
	}
	a.info.Types[lit] = &MapType{Key: key, Value: value}
}

func (a *Analyser) defaultUntyped() {
	for expr, t := range a.info.Types {
		if containsUntyped(t) {
			a.info.Types[expr] = defaultType(t)
		}
	}
	for _, values := range a.info.SwitchValues {
		for i, t := range values {
			if containsUntyped(t) {
				values[i] = defaultType(t)
			}
		}
	}
}

func (a *Analyser) conversionTarget(expr *ast.CallExpr) (Type, bool) {
	return a.typeCallee(expr, false)
}

func (a *Analyser) typeCallee(expr *ast.CallExpr, allowStruct bool) (Type, bool) {
	isType := func(sym *Symbol) bool {
		return sym.Kind == TYPE || (allowStruct && sym.Kind == STRUCT)
	}

	if member, ok := expr.Name.(*ast.MemberExpr); ok {
		module, isIdent := member.Object.(*ast.IdentLiteral)
		if !isIdent {
			return nil, false
		}
		if sym, found := a.lookupType(a.scope, module.Value, member.Field); found && isType(sym) {
			return sym.Type, true
		}
		return nil, false
	}

	id, ok := expr.Name.(*ast.IdentLiteral)
	if !ok {
		return nil, false
	}

	if sym, found := a.scope.Resolve(id.Value); found {
		if isType(sym) {
			return sym.Type, true
		}
		return nil, false
	}

	if primitives[id.Value] {
		return PrimitiveType{Name: id.Value}, true
	}
	return nil, false
}

func underlyingOf(t Type) Type {
	for {
		nt, ok := t.(NamedType)
		if !ok {
			return t
		}
		t = nt.Underlying
	}
}

func explicitConversion(from, to Type) bool {
	if from.Equals(to) {
		return true
	}
	if numericLike(from) && numericLike(to) {
		return true
	}
	if isChar(from) && isString(to) {
		return true
	}
	if span, ok := from.(SpanType); ok && isString(to) && isByteLike(span.Element) {
		return true
	}
	fu, tu := underlyingOf(from), underlyingOf(to)
	if _, ok := fu.(PointerType); ok && isWideInteger(tu) {
		return true
	}
	_, fromNamed := from.(NamedType)
	_, toNamed := to.(NamedType)
	return (fromNamed || toNamed) && fu.Equals(tu) && tu.Equals(fu)
}

func (a *Analyser) checkConversion(expr *ast.CallExpr, target Type) Type {
	if len(expr.Args) != 1 {
		a.errorf(expr, "conversion to %s expects 1 argument, got %d", target.String(), len(expr.Args))
		a.evalArgTypes(expr)
		return target
	}

	from := a.checkExpr(expr.Args[0])
	if _, bad := from.(InvalidType); bad {
		return target
	}

	if at, isArray := from.(ArrayType); isArray && isString(target) && isByteLike(at.Element) && isAddressable(expr.Args[0]) {
		span := SpanType{Element: at.Element}
		a.insertSlice(&expr.Args[0], span)
		from = span
	}

	if !explicitConversion(from, target) {
		a.errorf(expr, "cannot convert %s to %s", from.String(), target.String())
		return InvalidType{}
	}

	if isUntyped(from) {
		if _, isFloat := from.(UntypedFloatType); isFloat && !isFloatType(target) {
			a.fixUntyped(expr.Args[0], PrimitiveType{Name: "float"})
		} else {
			a.checkConstFits(expr.Args[0], target)
			a.fixUntyped(expr.Args[0], target)
		}
	}

	cast := &ast.CastExpr{Value: expr.Args[0]}
	line, col := expr.Position()
	cast.SetPos(line, col)
	endLine, endCol := expr.EndPosition()
	cast.SetEndPos(endLine, endCol)

	a.info.Types[cast] = target
	expr.Cast = cast
	return target
}

func isFloatType(t Type) bool {
	p, ok := underlying(t).(PrimitiveType)
	return ok && isFloatName(p.Name)
}

func isWideInteger(t Type) bool {
	p, ok := t.(PrimitiveType)
	if !ok {
		return false
	}
	switch p.Name {
	case "int", "int64", "uint", "uint64":
		return true
	}
	return false
}

func (a *Analyser) checkConvExpr(expr *ast.ConvExpr) Type {
	result := a.resolveTypeRef(expr, a.scope, expr.Type)
	if isInvalidType(result) {
		for _, arg := range expr.Args {
			a.checkExpr(arg)
		}
		return result
	}
	pointer, isPointer := result.(PointerType)
	if !isPointer {
		a.errorf(expr, "%s is not a pointer type: write a conversion to it as %s(x)", result.String(), result.String())
		for _, arg := range expr.Args {
			a.checkExpr(arg)
		}
		return InvalidType{}
	}

	if len(expr.Args) != 1 {
		a.errorf(expr, "conversion to %s expects 1 argument, got %d", result.String(), len(expr.Args))
		for _, arg := range expr.Args {
			a.checkExpr(arg)
		}
		return result
	}

	arg := expr.Args[0]
	from := a.checkExpr(arg)
	switch {
	case isInvalidType(from):
		return result
	case isUntyped(from) && isInteger(from):
		a.checkConstFits(arg, PrimitiveType{Name: "uint64"})
		a.fixUntyped(arg, PrimitiveType{Name: "uint64"})
	case isWideInteger(underlyingOf(from)):
	case isString(underlyingOf(from)) && isByteLike(pointer.Element):
	case sequenceMatches(underlyingOf(from), pointer.Element):
		if _, isArray := underlyingOf(from).(ArrayType); isArray && !isAddressable(arg) {
			a.errorf(arg, "this array has no address to point at").
				WithHelp("store it in a variable first, then convert the variable")
			return InvalidType{}
		}
	default:
		if _, isPtr := underlyingOf(from).(PointerType); !isPtr {
			d := a.errorf(expr, "cannot convert %s to %s", from.String(), result.String())
			if _, hasElement := sequenceElement(underlyingOf(from)); hasElement {
				d.WithHelp("an array or span converts to a pointer to its own element type, or to *void")
			}
			return InvalidType{}
		}
	}

	cast := &ast.CastExpr{Value: arg}
	line, col := expr.Position()
	cast.SetPos(line, col)
	endLine, endCol := expr.EndPosition()
	cast.SetEndPos(endLine, endCol)

	a.info.Types[cast] = result
	expr.Cast = cast
	return result
}

func isInvalidType(t Type) bool {
	_, bad := t.(InvalidType)
	return bad
}

func isByteLike(t Type) bool {
	p, ok := underlyingOf(t).(PrimitiveType)
	if !ok {
		return false
	}
	switch p.Name {
	case "char", "int8", "uint8":
		return true
	}
	return false
}

func sequenceElement(t Type) (Type, bool) {
	switch v := t.(type) {
	case ArrayType:
		return v.Element, true
	case SpanType:
		return v.Element, true
	}
	return nil, false
}

func sequenceMatches(from Type, pointee Type) bool {
	element, ok := sequenceElement(from)
	if !ok {
		return false
	}
	if _, isVoid := pointee.(VoidType); isVoid {
		return true
	}
	return element.Equals(pointee)
}

func (a *Analyser) retypeEmpty(expr ast.Expression, from, to Type) bool {
	if _, isEmpty := from.(EmptyArrayType); !isEmpty {
		return false
	}
	lit, ok := expr.(*ast.ArrayLiteral)
	if !ok {
		return false
	}
	if _, isSpan := to.(SpanType); !isSpan {
		return false
	}
	a.info.Types[lit] = to
	delete(a.pendingEmpty, lit)
	return true
}

func (a *Analyser) reportUnresolvedEmpty() {
	nodes := make([]*ast.ArrayLiteral, 0, len(a.pendingEmpty))
	for node := range a.pendingEmpty {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		li, ci := nodes[i].Position()
		lj, cj := nodes[j].Position()
		if li != lj {
			return li < lj
		}
		return ci < cj
	})
	for _, node := range nodes {
		a.error(node, "cannot infer the type of an empty []").
			WithHelp("it takes the type of a slice that receives it, so write it where a slice is expected")
	}
}

func voidPointerConversion(from, to Type) bool {
	fp, ok := from.(PointerType)
	if !ok {
		return false
	}
	tp, ok := to.(PointerType)
	if !ok {
		return false
	}
	_, fromVoid := fp.Element.(VoidType)
	_, toVoid := tp.Element.(VoidType)
	return fromVoid != toVoid
}
