package analyser

import (
	"fmt"
	"math/big"

	"github.com/Gui97p/wisp/internal/ast"
)

type ConstKind int

const (
	ConstInt ConstKind = iota
	ConstFloat
	ConstBool
	ConstString
)

type ConstValue struct {
	Kind  ConstKind
	Int   *big.Int
	Float float64
	Bool  bool
	Str   string
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
		if sym, ok := a.info.Idents[e]; ok && sym.Kind == CONST && sym.Const != nil {
			return *sym.Const, true
		}
	case *ast.CastExpr:
		return a.evalConversion(e)
	case *ast.CallExpr:
		if e.Cast != nil {
			return a.evalConversion(e.Cast)
		}
	case *ast.MemberExpr, *ast.DotIdent:
		if value, ok := a.info.EnumValues[e]; ok {
			return constInt(big.NewInt(value.Value)), true
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
	case "/", "%":
		if r.Sign() == 0 {
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

func (a *Analyser) constantFor(node ast.Node, values []ast.Expression, n, i int, target Type) *ConstValue {
	slot := a.valueSlot(values, n, i)
	if slot == nil {
		a.error(node, "const initializer must be a constant expression")
		return nil
	}

	value, ok := a.evalConst(*slot)
	if !ok {
		a.error(*slot, "const initializer must be a constant expression")
		return nil
	}

	if isUntyped(target) {
		return &value
	}

	converted, ok := a.convertConst(value, target)
	if !ok {
		if _, _, literal := intLiteralValue(*slot); !literal {
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
