package analyser

import (
	"math/big"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) recordFold(expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.BinaryExpr, *ast.UnaryExpr, *ast.CastExpr:
	case *ast.CallExpr:
		if e.Cast == nil {
			return
		}
	default:
		return
	}

	a.quiet = true
	value, ok := a.evalConst(expr)
	a.quiet = false
	if ok && (value.Kind == ConstInt || value.Kind == ConstBool) {
		a.info.Folds[expr] = value
	}
}

func (i *Info) Folded(expr ast.Expression) (ConstValue, bool) {
	if !i.exact(expr) {
		return ConstValue{}, false
	}
	if value, ok := i.Folds[expr]; ok {
		return value, true
	}
	return i.leaf(expr)
}

func (i *Info) exact(expr ast.Expression) bool {
	var child ast.Expression
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if !i.exact(e.Left) || !i.exact(e.Right) {
			return false
		}
	case *ast.UnaryExpr:
		child = e.Value
	case *ast.CastExpr:
		child = e.Value
	case *ast.CallExpr:
		if e.Cast == nil {
			return false
		}
		child = e.Cast.Value
	default:
		value, ok := i.leaf(expr)
		return ok && i.fits(expr, value)
	}

	if child != nil && !i.exact(child) {
		return false
	}
	value, ok := i.Folds[expr]
	return ok && i.fits(expr, value)
}

func (i *Info) leaf(expr ast.Expression) (ConstValue, bool) {
	var value ConstValue
	switch e := expr.(type) {
	case *ast.IntLiteral:
		if e.Unsigned {
			value = constInt(new(big.Int).SetUint64(uint64(e.Value)))
		} else {
			value = constInt(big.NewInt(e.Value))
		}
	case *ast.CharLiteral:
		value = constInt(big.NewInt(int64(e.Value)))
	case *ast.BoolLiteral:
		value = ConstValue{Kind: ConstBool, Bool: e.Value}
	case *ast.IdentLiteral:
		sym, ok := i.Idents[e]
		if !ok || sym.Kind != CONST || sym.Const == nil {
			return ConstValue{}, false
		}
		value = *sym.Const
	case *ast.MemberExpr, *ast.DotIdent:
		if enum, ok := i.EnumValues[e]; ok {
			value = constInt(big.NewInt(enum.Value))
			break
		}
		member, isMember := e.(*ast.MemberExpr)
		if !isMember {
			return ConstValue{}, false
		}
		object, ok := member.Object.(*ast.IdentLiteral)
		if !ok {
			return ConstValue{}, false
		}
		sym, ok := i.Idents[object]
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
		value = *export.Const
	default:
		return ConstValue{}, false
	}
	if value.Kind != ConstInt && value.Kind != ConstBool {
		return ConstValue{}, false
	}
	return value, true
}

func (i *Info) fits(expr ast.Expression, value ConstValue) bool {
	t, ok := i.Types[expr]
	if !ok || t == nil {
		return false
	}
	switch t.(type) {
	case UntypedIntType:
		return value.Kind == ConstInt && fitsBig(value.Int, 64, true)
	}
	base := underlying(t)
	if p, ok := base.(PrimitiveType); ok && p.Name == "bool" {
		return value.Kind == ConstBool
	}
	bits, signed, ok := integerRange(base)
	return ok && value.Kind == ConstInt && fitsBig(value.Int, bits, signed)
}
