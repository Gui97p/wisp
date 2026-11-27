package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) lowerFromEnd(array ast.Expression, arrType Type, bound *ast.Expression) {
	fromEnd, ok := (*bound).(*ast.FromEndExpr)
	if !ok {
		return
	}
	*bound = fromEnd.Value

	var length ast.Expression
	switch t := arrType.(type) {
	case InvalidType:
		return
	case ArrayType:
		length = &ast.IntLiteral{Value: t.Size}
		a.synthetic(fromEnd, length)
	case SpanType:
	case PrimitiveType:
		if t.Name != "string" {
			a.fromEndNeedsLength(fromEnd, arrType)
			return
		}
	default:
		a.fromEndNeedsLength(fromEnd, arrType)
		return
	}

	if length == nil {
		if !repeatable(array) {
			a.error(fromEnd, "^ counts from the end of a value it has to read twice, and this one has a call or side effect in it").
				WithHelp("put the value in a variable first, then use ^ on the variable")
			return
		}
		if sym, found := a.scope.Resolve("len"); !found || !sym.Builtin {
			a.error(fromEnd, "^ needs the builtin len, and `len` means something else here")
			return
		}
		name := &ast.IdentLiteral{Value: "len"}
		a.synthetic(fromEnd, name)
		length = &ast.CallExpr{Name: name, Args: []ast.Expression{ast.Clone(array, ast.CloneOptions{})}}
		a.synthetic(fromEnd, length)
	}

	count := &ast.BinaryExpr{Left: length, Operator: "-", Right: fromEnd.Value, Synthetic: true}
	line, col := fromEnd.Position()
	endLine, endCol := fromEnd.EndPosition()
	count.SetPos(line, col)
	count.SetEndPos(endLine, endCol)
	*bound = count
}

func (a *Analyser) fromEndNeedsLength(at ast.Node, t Type) {
	a.errorf(at, "^ counts from the end, so it needs an array, a span or a string, and %s has no length", t.String())
}

func repeatable(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IdentLiteral, *ast.IntLiteral, *ast.StringLiteral:
		return true
	case *ast.MemberExpr:
		return repeatable(e.Object)
	case *ast.DerefExpr:
		return repeatable(e.Value)
	case *ast.IndexExpr:
		return repeatable(e.Array) && repeatable(e.Index)
	case *ast.BinaryExpr:
		return repeatable(e.Left) && repeatable(e.Right)
	case *ast.CallExpr:
		name, isName := e.Name.(*ast.IdentLiteral)
		return isName && name.Value == "len" && len(e.Args) == 1 && repeatable(e.Args[0])
	case *ast.UnaryExpr:
		return e.Operator != "&" && repeatable(e.Value)
	}
	return false
}
