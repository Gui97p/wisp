package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkArithmetic(node ast.Node, op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if op == "+" && isString(left) && isString(right) {
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

	if !left.Equals(right) {
		a.errorf(node, "incompatible types: %s %s %s", left, op, right)
		return InvalidType{}
	}

	return left
}
