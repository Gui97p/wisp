package analyser

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

type ConstKind int

const (
	ConstInt ConstKind = iota
	ConstFloat
	ConstBool
	ConstString
	ConstArray
	ConstStruct
)

type ConstValue struct {
	Kind  ConstKind
	Int   *big.Int
	Float float64
	Bool  bool
	Str   string

	Elems []ConstValue
	Type  Type
}

func (c ConstValue) clone() ConstValue {
	if c.Kind != ConstArray && c.Kind != ConstStruct {
		return c
	}
	out := c
	out.Elems = make([]ConstValue, len(c.Elems))
	for i, e := range c.Elems {
		out.Elems[i] = e.clone()
	}
	return out
}

func (c ConstValue) aggregate() bool {
	return c.Kind.IsAggregate()
}

func (k ConstKind) IsAggregate() bool {
	return k == ConstArray || k == ConstStruct
}

func (c ConstValue) Bits() int64 {
	if c.Int == nil {
		return 0
	}
	if c.Int.IsInt64() {
		return c.Int.Int64()
	}
	if c.Int.IsUint64() {
		return int64(c.Int.Uint64())
	}
	return 0
}

func (c ConstValue) String() string {
	switch c.Kind {
	case ConstInt:
		return c.Int.String()
	case ConstFloat:
		return fmt.Sprint(c.Float)
	case ConstBool:
		return fmt.Sprint(c.Bool)
	case ConstArray:
		return fmt.Sprintf("an array of %d elements", len(c.Elems))
	case ConstStruct:
		return "a struct value"
	}
	return c.Str
}

func constInt(v *big.Int) ConstValue {
	return ConstValue{Kind: ConstInt, Int: v}
}

func (c ConstValue) asFloat() float64 {
	if c.Kind == ConstInt {
		f, _ := new(big.Float).SetInt(c.Int).Float64()
		return f
	}
	return c.Float
}

func (a *Analyser) evalConst(expr ast.Expression) (ConstValue, bool) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		if e.Unsigned {
			return constInt(new(big.Int).SetUint64(uint64(e.Value))), true
		}
		return constInt(big.NewInt(e.Value)), true
	case *ast.FloatLiteral:
		return ConstValue{Kind: ConstFloat, Float: e.Value}, true
	case *ast.BoolLiteral:
		return ConstValue{Kind: ConstBool, Bool: e.Value}, true
	case *ast.CharLiteral:
		return constInt(big.NewInt(int64(e.Value))), true
	case *ast.StringLiteral:
		return ConstValue{Kind: ConstString, Str: e.Value}, true
	case *ast.IdentLiteral:
		if sym, ok := a.info.Idents[e]; ok {
			if value, found := a.staticVars[sym]; found {
				return value, true
			}
			if sym.Kind == CONST && sym.Const != nil {
				return *sym.Const, true
			}
		}
	case *ast.CastExpr:
		return a.evalConversion(e)
	case *ast.ConstRef:
		if ref, ok := a.info.ConstRefs[e]; ok {
			return ref.Value, true
		}
	case *ast.ArrayLiteral:
		return a.evalArrayLiteral(e)
	case *ast.StructLiteral:
		return a.evalStructLiteral(e)
	case *ast.IndexExpr:
		return a.evalIndex(e)
	case *ast.SliceExpr:
		return a.evalStringSlice(e)
	case *ast.SwitchExpr:
		if a.staticVars != nil {
			return a.evalSwitchExpr(e)
		}
	case *ast.CallExpr:
		if value, ok := a.info.LayoutValues[e]; ok {
			return layoutConst(value), true
		}
		if value, ok := a.evalLen(e); ok {
			return value, true
		}
		if e.Cast != nil {
			return a.evalConversion(e.Cast)
		}
		if a.staticVars != nil {
			return a.staticCall(e)
		}
	case *ast.TernaryExpr:
		condition, ok := a.evalConst(e.Condition)
		if !ok || condition.Kind != ConstBool {
			return ConstValue{}, false
		}
		if condition.Bool {
			return a.evalConst(e.Then)
		}
		return a.evalConst(e.Else)
	case *ast.BlockExpr:
		if value, ok := a.info.Statics[e]; ok {
			return value, true
		}
	case *ast.MemberExpr, *ast.DotIdent:
		if value, ok := a.info.EnumValues[e]; ok {
			return constInt(big.NewInt(value.Value)), true
		}
		if member, isMember := e.(*ast.MemberExpr); isMember {
			if value, ok := a.moduleConst(member); ok {
				return value, true
			}
			return a.evalField(member)
		}
	case *ast.UnaryExpr:
		return a.evalUnary(e)
	case *ast.BinaryExpr:
		return a.evalBinary(e)
	}
	return ConstValue{}, false
}

func (a *Analyser) evalConversion(e *ast.CastExpr) (ConstValue, bool) {
	value, ok := a.evalConst(e.Value)
	if !ok {
		return ConstValue{}, false
	}

	target := underlying(a.info.Types[e])
	if (e.Truncate || e.Saturate) && value.Kind == ConstInt {
		if bits, signed, ok := IntegerRange(a.info.Types[e]); ok {
			if e.Saturate {
				return constInt(clampBig(value.Int, bits, signed)), true
			}
			return constInt(wrapBig(value.Int, bits, signed)), true
		}
	}
	switch {
	case isFloatType(target) && value.Kind == ConstInt:
		return ConstValue{Kind: ConstFloat, Float: value.asFloat()}, true
	case isInteger(target) && value.Kind == ConstFloat:
		truncated, _ := new(big.Float).SetFloat64(value.Float).Int(nil)
		return constInt(truncated), true
	}
	return value, true
}

func (a *Analyser) evalUnary(e *ast.UnaryExpr) (ConstValue, bool) {
	value, ok := a.evalConst(e.Value)
	if !ok {
		return ConstValue{}, false
	}

	switch e.Operator {
	case "+":
		return value, value.Kind == ConstInt || value.Kind == ConstFloat
	case "-":
		switch value.Kind {
		case ConstInt:
			return constInt(new(big.Int).Neg(value.Int)), true
		case ConstFloat:
			return ConstValue{Kind: ConstFloat, Float: -value.Float}, true
		}
	case "~":
		if value.Kind != ConstInt {
			return ConstValue{}, false
		}
		if bits, signed, ok := IntegerRange(a.info.Types[e]); ok && !signed {
			mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bits)), big.NewInt(1))
			return constInt(new(big.Int).Xor(value.Int, mask)), true
		}
		return constInt(new(big.Int).Not(value.Int)), true
	case "!":
		if value.Kind == ConstBool {
			return ConstValue{Kind: ConstBool, Bool: !value.Bool}, true
		}
	}
	return ConstValue{}, false
}

func (a *Analyser) evalBinary(e *ast.BinaryExpr) (ConstValue, bool) {
	left, ok := a.evalConst(e.Left)
	if !ok {
		return ConstValue{}, false
	}
	right, ok := a.evalConst(e.Right)
	if !ok {
		return ConstValue{}, false
	}

	switch {
	case left.Kind == ConstBool && right.Kind == ConstBool:
		return evalBoolBinary(e.Operator, left.Bool, right.Bool)
	case left.Kind == ConstString && right.Kind == ConstString:
		return evalStringBinary(e.Operator, left.Str, right.Str)
	case left.Kind == ConstInt && right.Kind == ConstInt:
		return a.evalIntBinary(e, left.Int, right.Int)
	case (left.Kind == ConstInt || left.Kind == ConstFloat) && (right.Kind == ConstInt || right.Kind == ConstFloat):
		return evalFloatBinary(e.Operator, left.asFloat(), right.asFloat())
	}
	return ConstValue{}, false
}

func evalBoolBinary(op string, l, r bool) (ConstValue, bool) {
	switch op {
	case "&&":
		return ConstValue{Kind: ConstBool, Bool: l && r}, true
	case "||":
		return ConstValue{Kind: ConstBool, Bool: l || r}, true
	case "==":
		return ConstValue{Kind: ConstBool, Bool: l == r}, true
	case "!=":
		return ConstValue{Kind: ConstBool, Bool: l != r}, true
	}
	return ConstValue{}, false
}

func evalStringBinary(op string, l, r string) (ConstValue, bool) {
	switch op {
	case "+":
		return ConstValue{Kind: ConstString, Str: l + r}, true
	case "==":
		return ConstValue{Kind: ConstBool, Bool: l == r}, true
	case "!=":
		return ConstValue{Kind: ConstBool, Bool: l != r}, true
	case "<":
		return ConstValue{Kind: ConstBool, Bool: l < r}, true
	case "<=":
		return ConstValue{Kind: ConstBool, Bool: l <= r}, true
	case ">":
		return ConstValue{Kind: ConstBool, Bool: l > r}, true
	case ">=":
		return ConstValue{Kind: ConstBool, Bool: l >= r}, true
	}
	return ConstValue{}, false
}

func evalFloatBinary(op string, l, r float64) (ConstValue, bool) {
	float := func(v float64) (ConstValue, bool) { return ConstValue{Kind: ConstFloat, Float: v}, true }
	boolean := func(v bool) (ConstValue, bool) { return ConstValue{Kind: ConstBool, Bool: v}, true }

	switch op {
	case "+":
		return float(l + r)
	case "-":
		return float(l - r)
	case "*":
		return float(l * r)
	case "/":
		if r == 0 {
			return ConstValue{}, false
		}
		return float(l / r)
	case "==":
		return boolean(l == r)
	case "!=":
		return boolean(l != r)
	case "<":
		return boolean(l < r)
	case "<=":
		return boolean(l <= r)
	case ">":
		return boolean(l > r)
	case ">=":
		return boolean(l >= r)
	}
	return ConstValue{}, false
}

func (a *Analyser) evalIntBinary(e *ast.BinaryExpr, l, r *big.Int) (ConstValue, bool) {
	boolean := func(v bool) (ConstValue, bool) { return ConstValue{Kind: ConstBool, Bool: v}, true }

	switch e.Operator {
	case "+":
		return constInt(new(big.Int).Add(l, r)), true
	case "-":
		return constInt(new(big.Int).Sub(l, r)), true
	case "*":
		return constInt(new(big.Int).Mul(l, r)), true
	case "+%", "-%", "*%", "+|", "-|", "*|":
		return a.evalOverflowBinary(e, l, r), true
	case "/", "%":
		if r.Sign() == 0 {
			if a.quiet {
				return ConstValue{}, false
			}
			a.error(e, "division by zero in constant expression")
			return constInt(big.NewInt(0)), true
		}
		if e.Operator == "/" {
			return constInt(new(big.Int).Quo(l, r)), true
		}
		return constInt(new(big.Int).Rem(l, r)), true
	case "&":
		return constInt(new(big.Int).And(l, r)), true
	case "|":
		return constInt(new(big.Int).Or(l, r)), true
	case "^":
		return constInt(new(big.Int).Xor(l, r)), true
	case "<<", ">>":
		if r.Sign() < 0 || !r.IsUint64() || r.Uint64() > 4096 {
			if a.quiet {
				return ConstValue{}, false
			}
			a.error(e, "invalid shift count in constant expression")
			return constInt(big.NewInt(0)), true
		}
		if e.Operator == "<<" {
			return constInt(new(big.Int).Lsh(l, uint(r.Uint64()))), true
		}
		return constInt(new(big.Int).Rsh(l, uint(r.Uint64()))), true
	case "==":
		return boolean(l.Cmp(r) == 0)
	case "!=":
		return boolean(l.Cmp(r) != 0)
	case "<":
		return boolean(l.Cmp(r) < 0)
	case "<=":
		return boolean(l.Cmp(r) <= 0)
	case ">":
		return boolean(l.Cmp(r) > 0)
	case ">=":
		return boolean(l.Cmp(r) >= 0)
	}
	return ConstValue{}, false
}

func fitsBig(v *big.Int, bits int, signed bool) bool {
	one := big.NewInt(1)
	var lo, hi *big.Int
	if signed {
		hi = new(big.Int).Sub(new(big.Int).Lsh(one, uint(bits-1)), one)
		lo = new(big.Int).Neg(new(big.Int).Lsh(one, uint(bits-1)))
	} else {
		hi = new(big.Int).Sub(new(big.Int).Lsh(one, uint(bits)), one)
		lo = big.NewInt(0)
	}
	return v.Cmp(lo) >= 0 && v.Cmp(hi) <= 0
}

func (a *Analyser) constantFor(node ast.Node, values []ast.Expression, n, i int, target Type, reported bool) *ConstValue {
	slot := a.valueSlot(values, n, i)
	if slot == nil {
		if !reported {
			a.error(node, "const initializer must be a constant expression")
		}
		return nil
	}

	value, ok := a.evalConst(*slot)
	if !ok {
		if !reported {
			a.error(*slot, "const initializer must be a constant expression")
		}
		return nil
	}

	if isUntyped(target) {
		return &value
	}

	if value.aggregate() {
		fitted, ok := a.fitConst(value, target)
		if !ok {
			if !reported {
				a.errorf(*slot, "this constant does not fit %s", target.String())
			}
			return nil
		}
		return &fitted
	}

	converted, ok := a.convertConst(value, target)
	if !ok {
		if _, _, literal := intLiteralValue(*slot); !literal && !reported {
			a.errorf(*slot, "constant %s overflows %s", value, target.String())
		}
		return nil
	}
	return &converted
}

func (a *Analyser) convertConst(value ConstValue, target Type) (ConstValue, bool) {
	base := underlying(target)

	if bits, signed, ok := integerRange(base); ok {
		switch value.Kind {
		case ConstFloat:
			truncated, accuracy := new(big.Float).SetFloat64(value.Float).Int(nil)
			if accuracy != big.Exact {
				return value, false
			}
			value = constInt(truncated)
		case ConstInt:
		default:
			return value, true
		}
		return value, fitsBig(value.Int, bits, signed)
	}

	if isFloatType(base) && value.Kind == ConstInt {
		return ConstValue{Kind: ConstFloat, Float: value.asFloat()}, true
	}
	return value, true
}

func (a *Analyser) moduleConst(member *ast.MemberExpr) (ConstValue, bool) {
	object, ok := member.Object.(*ast.IdentLiteral)
	if !ok {
		return ConstValue{}, false
	}
	sym, ok := a.info.Idents[object]
	if !ok || sym.Kind != MODULE {
		return ConstValue{}, false
	}
	module, ok := sym.Type.(*ModuleType)
	if !ok {
		return ConstValue{}, false
	}
	export, ok := module.Exports[member.Field]
	if !ok || export.Kind != CONST || export.Const == nil {
		return ConstValue{}, false
	}
	return *export.Const, true
}

func (a *Analyser) evalOverflowBinary(e *ast.BinaryExpr, l, r *big.Int) ConstValue {
	var exact *big.Int
	switch e.Operator[0] {
	case '+':
		exact = new(big.Int).Add(l, r)
	case '-':
		exact = new(big.Int).Sub(l, r)
	default:
		exact = new(big.Int).Mul(l, r)
	}

	bits, signed, ok := IntegerRange(a.info.Types[e])
	if !ok {
		bits, signed = 64, true
	}
	if e.Operator[1] == '|' {
		return constInt(clampBig(exact, bits, signed))
	}
	return constInt(wrapBig(exact, bits, signed))
}

func wrapBig(v *big.Int, bits int, signed bool) *big.Int {
	modulus := new(big.Int).Lsh(big.NewInt(1), uint(bits))
	wrapped := new(big.Int).Mod(v, modulus)
	if signed && wrapped.Cmp(new(big.Int).Rsh(modulus, 1)) >= 0 {
		wrapped.Sub(wrapped, modulus)
	}
	return wrapped
}

func clampBig(v *big.Int, bits int, signed bool) *big.Int {
	one := big.NewInt(1)
	var lo, hi *big.Int
	if signed {
		hi = new(big.Int).Sub(new(big.Int).Lsh(one, uint(bits-1)), one)
		lo = new(big.Int).Neg(new(big.Int).Lsh(one, uint(bits-1)))
	} else {
		hi = new(big.Int).Sub(new(big.Int).Lsh(one, uint(bits)), one)
		lo = big.NewInt(0)
	}
	switch {
	case v.Cmp(hi) > 0:
		return hi
	case v.Cmp(lo) < 0:
		return lo
	}
	return v
}

func zeroValue(tp Type) (ConstValue, bool) {
	switch v := underlying(tp).(type) {
	case PrimitiveType:
		switch {
		case v.Name == "bool":
			return ConstValue{Kind: ConstBool}, true
		case v.Name == "string":
			return ConstValue{Kind: ConstString}, true
		case isFloatName(v.Name):
			return ConstValue{Kind: ConstFloat}, true
		case isInteger(v):
			return constInt(big.NewInt(0)), true
		}
	case ArrayType:
		elem, ok := zeroValue(v.Element)
		if !ok {
			return ConstValue{}, false
		}
		out := ConstValue{Kind: ConstArray, Type: tp, Elems: make([]ConstValue, v.Size)}
		for i := range out.Elems {
			out.Elems[i] = elem.clone()
		}
		return out, true
	case *StructType:
		if v.Sum != nil || v.Optional != nil {
			return ConstValue{}, false
		}
		out := ConstValue{Kind: ConstStruct, Type: tp, Elems: make([]ConstValue, len(v.Order))}
		for i, name := range v.Order {
			field, ok := zeroValue(v.Fields[name])
			if !ok {
				return ConstValue{}, false
			}
			out.Elems[i] = field
		}
		return out, true
	}
	return ConstValue{}, false
}

func (a *Analyser) fitConst(value ConstValue, target Type) (ConstValue, bool) {
	switch t := underlying(target).(type) {
	case ArrayType:
		if value.Kind != ConstArray || int64(len(value.Elems)) > t.Size {
			return value, false
		}
		out := ConstValue{Kind: ConstArray, Type: target, Elems: make([]ConstValue, t.Size)}
		for i := range out.Elems {
			if i < len(value.Elems) {
				elem, ok := a.fitConst(value.Elems[i], t.Element)
				if !ok {
					return value, false
				}
				out.Elems[i] = elem
				continue
			}
			zero, ok := zeroValue(t.Element)
			if !ok {
				return value, false
			}
			out.Elems[i] = zero
		}
		return out, true
	case *StructType:
		if value.Kind != ConstStruct || len(value.Elems) != len(t.Order) {
			return value, false
		}
		out := ConstValue{Kind: ConstStruct, Type: target, Elems: make([]ConstValue, len(t.Order))}
		for i, name := range t.Order {
			field, ok := a.fitConst(value.Elems[i], t.Fields[name])
			if !ok {
				return value, false
			}
			out.Elems[i] = field
		}
		return out, true
	}
	return a.convertConst(value, target)
}

func (a *Analyser) evalArrayLiteral(e *ast.ArrayLiteral) (ConstValue, bool) {
	out := ConstValue{Kind: ConstArray, Type: a.info.Types[e], Elems: make([]ConstValue, len(e.Elements))}
	var element Type
	if at, ok := underlying(out.Type).(ArrayType); ok {
		element = at.Element
	}
	for i, el := range e.Elements {
		value, ok := a.evalConst(el)
		if !ok {
			return ConstValue{}, false
		}
		if element != nil {
			if converted, can := a.fitConst(value, element); can {
				value = converted
			}
		}
		out.Elems[i] = value
	}
	return out, true
}

func (a *Analyser) evalStructLiteral(e *ast.StructLiteral) (ConstValue, bool) {
	tp := a.info.Types[e]
	st, ok := underlying(tp).(*StructType)
	if !ok || st.Sum != nil || st.Optional != nil {
		return ConstValue{}, false
	}
	out := ConstValue{Kind: ConstStruct, Type: tp, Elems: make([]ConstValue, len(st.Order))}
	given := map[string]ast.Expression{}
	for i, key := range e.Keys {
		given[key] = e.Values[i]
	}
	for i, name := range st.Order {
		expr, present := given[name]
		if !present {
			zero, ok := zeroValue(st.Fields[name])
			if !ok {
				return ConstValue{}, false
			}
			out.Elems[i] = zero
			continue
		}
		value, ok := a.evalConst(expr)
		if !ok {
			return ConstValue{}, false
		}
		if converted, can := a.fitConst(value, st.Fields[name]); can {
			value = converted
		}
		out.Elems[i] = value
	}
	return out, true
}

func (a *Analyser) evalIndex(e *ast.IndexExpr) (ConstValue, bool) {
	base, ok := a.evalConst(e.Array)
	if !ok {
		return ConstValue{}, false
	}
	index, ok := a.evalConst(e.Index)
	if !ok || index.Kind != ConstInt || !index.Int.IsInt64() {
		return ConstValue{}, false
	}
	i := index.Int.Int64()
	switch base.Kind {
	case ConstArray:
		if i < 0 || i >= int64(len(base.Elems)) {
			if !a.quiet && a.staticVars != nil {
				a.errorf(e, "index %d is out of range: the array has %d elements", i, len(base.Elems))
			}
			return ConstValue{}, false
		}
		return base.Elems[i], true
	case ConstString:
		if i < 0 || i >= int64(len(base.Str)) {
			if !a.quiet && a.staticVars != nil {
				a.errorf(e, "index %d is out of range: the string has %d bytes", i, len(base.Str))
			}
			return ConstValue{}, false
		}
		return constInt(big.NewInt(int64(base.Str[i]))), true
	}
	return ConstValue{}, false
}

func (a *Analyser) evalField(e *ast.MemberExpr) (ConstValue, bool) {
	object, ok := a.evalConst(e.Object)
	if !ok || object.Kind != ConstStruct {
		return ConstValue{}, false
	}
	st, ok := underlying(object.Type).(*StructType)
	if !ok {
		return ConstValue{}, false
	}
	for i, name := range st.Order {
		if name == e.Field && i < len(object.Elems) {
			return object.Elems[i], true
		}
	}
	return ConstValue{}, false
}

func (a *Analyser) evalLen(e *ast.CallExpr) (ConstValue, bool) {
	id, ok := e.Name.(*ast.IdentLiteral)
	if !ok || id.Value != "len" || len(e.Args) != 1 || a.info.Idents[id] != a.universe.symbols["len"] {
		return ConstValue{}, false
	}
	value, ok := a.evalConst(e.Args[0])
	if !ok {
		return ConstValue{}, false
	}
	switch value.Kind {
	case ConstString:
		return constInt(big.NewInt(int64(len(value.Str)))), true
	case ConstArray:
		return constInt(big.NewInt(int64(len(value.Elems)))), true
	}
	return ConstValue{}, false
}

func (a *Analyser) evalStringSlice(e *ast.SliceExpr) (ConstValue, bool) {
	base, ok := a.evalConst(e.Array)
	if !ok || base.Kind != ConstString {
		return ConstValue{}, false
	}
	start, end := int64(0), int64(len(base.Str))
	if e.Start != nil {
		value, ok := a.evalConst(e.Start)
		if !ok || value.Kind != ConstInt || !value.Int.IsInt64() {
			return ConstValue{}, false
		}
		start = value.Int.Int64()
	}
	if e.End != nil {
		value, ok := a.evalConst(e.End)
		if !ok || value.Kind != ConstInt || !value.Int.IsInt64() {
			return ConstValue{}, false
		}
		end = value.Int.Int64()
	}
	if start < 0 || end > int64(len(base.Str)) || start > end {
		if !a.quiet && a.staticVars != nil {
			a.errorf(e, "the slice %d:%d is out of range: the string has %d bytes", start, end, len(base.Str))
		}
		return ConstValue{}, false
	}
	return ConstValue{Kind: ConstString, Str: base.Str[start:end]}, true
}

func (c ConstValue) Preview(limit int) string {
	switch c.Kind {
	case ConstString:
		return strconv.Quote(c.Str)
	case ConstArray:
		var parts []string
		for i, e := range c.Elems {
			if i == limit {
				parts = append(parts, fmt.Sprintf("... %d more", len(c.Elems)-limit))
				break
			}
			parts = append(parts, e.Preview(limit))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ConstStruct:
		st, ok := underlying(c.Type).(*StructType)
		if !ok {
			return c.String()
		}
		var parts []string
		for i, name := range st.Order {
			if i < len(c.Elems) {
				parts = append(parts, name+": "+c.Elems[i].Preview(limit))
			}
		}
		return TypeText(c.Type) + "{" + strings.Join(parts, ", ") + "}"
	}
	return c.String()
}
