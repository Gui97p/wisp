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
	default:
		t = InvalidType{}
	}

	a.info.Types[expr] = t
	return t
}
