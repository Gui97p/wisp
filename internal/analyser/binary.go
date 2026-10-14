package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkArithmetic(node ast.Node, op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if op == "+" && isString(left) && (isString(right) || isChar(right)) {
		return left
	}

	if !isNumeric(left) || !isNumeric(right) {
		a.errorf(node, "invalid operator %s for %s and %s", op, left.String(), right.String())
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

	return PrimitiveType{Name: "bool"}
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
