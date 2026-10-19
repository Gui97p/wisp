package lsp

import "github.com/Gui97p/wisp/internal/ast"

func walkProgram(program *ast.Program, fn func(ast.Expression)) {
	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body != nil {
				walkStmt(d.Body, fn)
			}
		case *ast.ConstDecl:
			walkExprs(d.Values, fn)
		case *ast.VarDecl:
			walkExprs(d.Values, fn)
		}
	}
}

func walkExprs(exprs []ast.Expression, fn func(ast.Expression)) {
	for _, e := range exprs {
		walkExpr(e, fn)
	}
}

func walkStmt(stmt ast.Statement, fn func(ast.Expression)) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		walkExpr(s.Expr, fn)
	case *ast.BlockStmt:
		for _, inner := range s.Statements {
			walkStmt(inner, fn)
		}
	case *ast.GroupStmt:
		for _, inner := range s.Statements {
			walkStmt(inner, fn)
		}
	case *ast.VarStmt:
		walkExprs(s.Values, fn)
	case *ast.ConstStmt:
		walkExprs(s.Values, fn)
	case *ast.ReturnStmt:
		walkExprs(s.Values, fn)
	case *ast.IfStmt:
		walkExpr(s.Condition, fn)
		walkStmt(s.Then, fn)
		walkStmt(s.Else, fn)
	case *ast.ForStmt:
		walkExprs([]ast.Expression{s.Range, s.Start, s.End, s.Step}, fn)
		walkStmt(s.Body, fn)
	case *ast.LoopStmt:
		walkExprs([]ast.Expression{s.Condition, s.UntilCondition}, fn)
		walkStmt(s.Body, fn)
	case *ast.AssignStmt:
		walkExprs(s.Targets, fn)
		walkExprs(s.Values, fn)
	case *ast.IncDecStmt:
		walkExpr(s.Target, fn)
	}
}

func walkExpr(expr ast.Expression, fn func(ast.Expression)) {
	if expr == nil {
		return
	}
	fn(expr)

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		walkExprs([]ast.Expression{e.Left, e.Right}, fn)
	case *ast.UnaryExpr:
		walkExpr(e.Value, fn)
	case *ast.CallExpr:
		walkExpr(e.Name, fn)
		walkExprs(e.Args, fn)
	case *ast.MemberExpr:
		walkExpr(e.Object, fn)
	case *ast.IndexExpr:
		walkExprs([]ast.Expression{e.Array, e.Index}, fn)
	case *ast.SliceExpr:
		walkExprs([]ast.Expression{e.Array, e.Start, e.End}, fn)
	case *ast.TernaryExpr:
		walkExprs([]ast.Expression{e.Condition, e.Then, e.Else}, fn)
	case *ast.CoalesceExpr:
		walkExprs([]ast.Expression{e.Left, e.Default}, fn)
		walkStmt(e.Block, fn)
	case *ast.TupleExpr:
		walkExprs(e.Elements, fn)
	case *ast.CastExpr:
		walkExpr(e.Value, fn)
	case *ast.ConvExpr:
		walkExprs(e.Args, fn)
	case *ast.InExpr:
		walkExprs([]ast.Expression{e.Left, e.Right}, fn)
	case *ast.PropagateExpr:
		walkExpr(e.Value, fn)
	case *ast.ArrayLiteral:
		walkExprs(e.Elements, fn)
	case *ast.MapLiteral:
		walkExprs(e.Keys, fn)
		walkExprs(e.Values, fn)
	case *ast.StructLiteral:
		walkExprs(e.Values, fn)
	case *ast.FuncLiteral:
		walkStmt(e.Block, fn)
	}
}
