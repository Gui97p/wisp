package ast

import "fmt"

func ExprText(e Expression) string {
	switch v := e.(type) {
	case *IntLiteral:
		return fmt.Sprintf("%d", v.Value)
	case *IdentLiteral:
		return v.Value
	case *MemberExpr:
		return ExprText(v.Object) + "." + v.Field
	case *BinaryExpr:
		return ExprText(v.Left) + " " + v.Operator + " " + ExprText(v.Right)
	case *UnaryExpr:
		return v.Operator + ExprText(v.Value)
	}
	return "..."
}
