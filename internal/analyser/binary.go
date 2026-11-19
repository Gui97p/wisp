package analyser

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

func (a *Analyser) checkArithmetic(node ast.Node, op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !isNumeric(left) || !isNumeric(right) {
		a.errorf(node, "invalid operator %s for %s and %s", op, left.String(), right.String())
		return InvalidType{}
	}

	if op == "%" && (isFloat(left) || isFloat(right)) {
		a.errorf(node, "operator %% is not defined for floating point numbers")
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf(node, "incompatible types: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	return typedSide(left, right)
}

func (a *Analyser) checkOverflowArithmetic(node ast.Node, op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !isInteger(underlyingOf(left)) || !isInteger(underlyingOf(right)) {
		a.errorf(node, "operator %s is for integers, got %s and %s", op, left.String(), right.String()).
			WithHelp("+%% -%% *%% wrap around and +| -| *| stop at the limits, and neither exists for floating point numbers")
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf(node, "incompatible types: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	return typedSide(left, right)
}

func typedSide(left, right Type) Type {
	switch left.(type) {
	case UntypedIntType, UntypedFloatType:
		switch right.(type) {
		case UntypedFloatType:
			return right
		case UntypedIntType:
		default:
			return right
		}
	}
	return left
}

func (a *Analyser) checkComparison(node ast.Node, op string, left, right Type) Type {
	a.comparisonNode = node
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf(node, "incompatible types: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	if reason := a.uncomparable(op, left); reason != nil {
		reason.WithHelp("%s", a.comparisonHelp(op, left))
		return InvalidType{}
	}

	return PrimitiveType{Name: "bool"}
}

func (a *Analyser) uncomparable(op string, t Type) *diag.Diagnostic {
	ordering := op != "==" && op != "!="
	switch v := definedBase(t).(type) {
	case *StructType:
		if v.Optional == nil && v.Sum == nil {
			return a.errorf(a.comparisonNode, "operator %s is not defined for %s", op, t)
		}
	case ArrayType, SpanType:
		return a.errorf(a.comparisonNode, "operator %s is not defined for %s", op, t)
	case *FuncType:
		if ordering {
			return a.errorf(a.comparisonNode, "operator %s is not defined for %s", op, t)
		}
	}
	return nil
}

func (a *Analyser) comparisonHelp(op string, t Type) string {
	if st, ok := t.(*StructType); ok && strings.HasPrefix(st.key, "opaque:") {
		if op == "==" || op == "!=" {
			return "constrain the type parameter with Comparable to compare its values"
		}
		return "constrain the type parameter with an interface that lists the types that can be ordered"
	}
	return "compare the fields, or the elements, one by one"
}

func (a *Analyser) checkLogical(node ast.Node, op string, left, right Type) Type {
	boolType := PrimitiveType{Name: "bool"}

	if _, ok := left.(InvalidType); !ok && !left.Equals(boolType) {
		a.errorf(node, "operator %s expects bool, got %s", op, left.String())
	}
	if _, ok := right.(InvalidType); !ok && !right.Equals(boolType) {
		a.errorf(node, "operator %s expects bool, got %s", op, right.String())
	}

	return boolType
}

func (a *Analyser) checkBitwise(node ast.Node, op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !isInteger(left) || !isInteger(right) {
		a.errorf(node, "invalid operator %s for %s and %s", op, left, right)
		return InvalidType{}
	}

	if op == "<<" || op == ">>" {
		return left
	}

	if !left.Equals(right) {
		a.errorf(node, "incompatible types: %s %s %s", left, op, right)
		return InvalidType{}
	}

	return typedSide(left, right)
}

func isVoidPointer(t Type) bool {
	p, ok := t.(PointerType)
	if !ok {
		return false
	}
	_, void := p.Element.(VoidType)
	return void
}

func (a *Analyser) pointerArithmetic(expr *ast.BinaryExpr, left, right Type) (Type, bool) {
	lp, leftPtr := left.(PointerType)
	rp, rightPtr := right.(PointerType)
	if !leftPtr && !rightPtr {
		return nil, false
	}
	if expr.Operator != "+" && expr.Operator != "-" {
		return nil, false
	}
	if _, bad := left.(InvalidType); bad {
		return InvalidType{}, true
	}
	if _, bad := right.(InvalidType); bad {
		return InvalidType{}, true
	}

	if a.rejectOptionalUse(expr, left, "arithmetic") || a.rejectOptionalUse(expr, right, "arithmetic") {
		return InvalidType{}, true
	}

	if isVoidPointer(left) || isVoidPointer(right) {
		a.errorf(expr, "arithmetic on %s: convert it to a typed pointer first", a.pointerName(left, right))
		return InvalidType{}, true
	}

	switch {
	case leftPtr && rightPtr:
		if expr.Operator == "+" {
			a.error(expr, "cannot add two pointers")
			return InvalidType{}, true
		}
		if !lp.Element.Equals(rp.Element) {
			a.errorf(expr, "cannot subtract %s from %s", right.String(), left.String())
			return InvalidType{}, true
		}
		return PrimitiveType{Name: "int"}, true
	case leftPtr:
		if !isInteger(right) {
			a.errorf(expr, "pointer offset must be an integer, got %s", right.String())
			return InvalidType{}, true
		}
		a.fixUntyped(expr.Right, PrimitiveType{Name: "int"})
		return left, true
	default:
		if expr.Operator == "-" {
			a.error(expr, "cannot subtract a pointer from an integer")
			return InvalidType{}, true
		}
		if !isInteger(left) {
			a.errorf(expr, "pointer offset must be an integer, got %s", left.String())
			return InvalidType{}, true
		}
		a.fixUntyped(expr.Left, PrimitiveType{Name: "int"})
		return right, true
	}
}

func (a *Analyser) pointerName(left, right Type) string {
	if isVoidPointer(left) {
		return left.String()
	}
	return right.String()
}
