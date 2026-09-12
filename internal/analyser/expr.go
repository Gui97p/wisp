package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) checkExpr(expr ast.Expression) Type {
	var t Type

	switch e := expr.(type) {
	case *ast.IntLiteral:
		t = PrimitiveType{Name: "int"}
	case *ast.FloatLiteral:
		t = PrimitiveType{Name: "float64"}
	case *ast.StringLiteral:
		t = PrimitiveType{Name: "string"}
	case *ast.CharLiteral:
		t = PrimitiveType{Name: "char"}
	case *ast.BoolLiteral:
		t = PrimitiveType{Name: "bool"}
	case *ast.IdentLiteral:
		symbol, ok := a.scope.Resolve(e.Value)
		if !ok {
			a.errorf("identifier %s not declared in this scope", e.Value)
			t = InvalidType{}
			break
		}
		a.info.Idents[e] = symbol
		t = symbol.Type
	case *ast.BinaryExpr:
		t = a.checkBinaryExpr(e)
	default:
		t = InvalidType{}
	}

	a.info.Types[expr] = t
	return t
}

func (a *Analyser) checkBinaryExpr(expr *ast.BinaryExpr) Type {
	left := a.checkExpr(expr.Left)
	right := a.checkExpr(expr.Right)

	switch expr.Operator {
	case "+", "-", "*", "/", "%":
		return a.checkArithmetic(expr.Operator, left, right)
	case "==", "!=", ">", ">=", "<", "<=":
		return a.checkComparison(expr.Operator, left, right)
	case "&&", "||":
		return a.checkLogical(expr.Operator, left, right)
	default:
		a.errorf("unknown operator %s", expr.Operator)
		return InvalidType{}
	}
}

func (a *Analyser) checkArithmetic(op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !isNumeric(left) || !isNumeric(right) {
		a.errorf("invalid operator %s for %s and %s", op, left.String(), right.String())
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf("incompatible types: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	return left
}

func (a *Analyser) checkComparison(op string, left, right Type) Type {
	if _, ok := left.(InvalidType); ok {
		return InvalidType{}
	}
	if _, ok := right.(InvalidType); ok {
		return InvalidType{}
	}

	if !left.Equals(right) {
		a.errorf("tipos incompatíveis: %s %s %s", left.String(), op, right.String())
		return InvalidType{}
	}

	return PrimitiveType{Name: "bool"}
}

func (a *Analyser) checkLogical(op string, left, right Type) Type {
	boolType := PrimitiveType{Name: "bool"}

	if _, ok := left.(InvalidType); !ok && !left.Equals(boolType) {
		a.errorf("operador %s espera bool, recebeu %s", op, left.String())
	}
	if _, ok := right.(InvalidType); !ok && !right.Equals(boolType) {
		a.errorf("operador %s espera bool, recebeu %s", op, right.String())
	}

	return boolType
}
