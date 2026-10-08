package analyser

import (
	"fmt"

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
		return PrimitiveType{Name: "float64"}
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
	if fp.Name == "float32" && tp.Name == "float64" {
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

	if isImplicitEnum(from) {
		return a.resolveDot(*slot, to)
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
}

func (a *Analyser) conversionTarget(expr *ast.CallExpr) (Type, bool) {
	if member, ok := expr.Name.(*ast.MemberExpr); ok {
		module, isIdent := member.Object.(*ast.IdentLiteral)
		if !isIdent {
			return nil, false
		}
		if sym, found := a.lookupType(a.scope, module.Value, member.Field); found && sym.Kind == TYPE {
			return sym.Type, true
		}
		return nil, false
	}

	id, ok := expr.Name.(*ast.IdentLiteral)
	if !ok {
		return nil, false
	}

	if sym, found := a.scope.Resolve(id.Value); found {
		if sym.Kind == TYPE {
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
	fu, tu := underlyingOf(from), underlyingOf(to)
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

	if !explicitConversion(from, target) {
		a.errorf(expr, "cannot convert %s to %s", from.String(), target.String())
		return InvalidType{}
	}

	if isUntyped(from) {
		if _, isFloat := from.(UntypedFloatType); isFloat && !isFloatType(target) {
			a.fixUntyped(expr.Args[0], PrimitiveType{Name: "float64"})
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
	return ok && (p.Name == "float32" || p.Name == "float64")
}
