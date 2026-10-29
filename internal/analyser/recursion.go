package analyser

import (
	"reflect"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) warnInfiniteRecursion(fd *ast.FuncDecl) {
	if fd.Body == nil || fd.Generated {
		return
	}
	if !a.alwaysRecurses(fd, fd.Body.Statements) {
		return
	}
	line, col := fd.Position()
	endCol := col + len("func") - 1
	if fd.Receiver == nil {
		col += len("func ")
		endCol = col + len(fd.Name) - 1
	}
	a.errors.Warn(a.currentFile, line, col, line, endCol, "function %s calls itself on every path", fd.Name).
		WithHelp("add a condition that returns before the call, or the stack will overflow")
}

func (a *Analyser) alwaysRecurses(fd *ast.FuncDecl, list []ast.Statement) bool {
	for _, stmt := range list {
		if a.stmtRecurses(fd, stmt) {
			return true
		}
		if mayLeave(stmt) {
			return false
		}
	}
	return false
}

func (a *Analyser) stmtRecurses(fd *ast.FuncDecl, stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.ExpressionStmt:
		return a.exprRecurses(fd, s.Expr)
	case *ast.VarStmt:
		return a.anyRecurses(fd, s.Values)
	case *ast.AssignStmt:
		return a.anyRecurses(fd, s.Values)
	case *ast.ReturnStmt:
		return a.anyRecurses(fd, s.Values)
	case *ast.IfStmt:
		if a.exprRecurses(fd, s.Condition) {
			return true
		}
		return s.Else != nil && a.alwaysRecurses(fd, s.Then.Statements) && a.stmtRecurses(fd, s.Else)
	case *ast.BlockStmt:
		return a.alwaysRecurses(fd, s.Statements)
	case *ast.LoopStmt:
		return s.Condition == nil && s.UntilCondition == nil && a.alwaysRecurses(fd, s.Body.Statements)
	case *ast.ForStmt:
		return a.exprRecurses(fd, s.Range) || a.exprRecurses(fd, s.Start) || a.exprRecurses(fd, s.End)
	default:
		return false
	}
}

func (a *Analyser) anyRecurses(fd *ast.FuncDecl, exprs []ast.Expression) bool {
	for _, e := range exprs {
		if a.exprRecurses(fd, e) {
			return true
		}
	}
	return false
}

func (a *Analyser) exprRecurses(fd *ast.FuncDecl, expr ast.Expression) bool {
	switch e := expr.(type) {
	case nil:
		return false
	case *ast.CallExpr:
		return a.callsItself(fd, e) || a.exprRecurses(fd, e.Name) || a.anyRecurses(fd, e.Args)
	case *ast.BinaryExpr:
		if e.Operator == "&&" || e.Operator == "||" {
			return a.exprRecurses(fd, e.Left)
		}
		return a.exprRecurses(fd, e.Left) || a.exprRecurses(fd, e.Right)
	case *ast.TernaryExpr:
		return a.exprRecurses(fd, e.Condition) || (a.exprRecurses(fd, e.Then) && a.exprRecurses(fd, e.Else))
	case *ast.CoalesceExpr:
		return a.exprRecurses(fd, e.Left)
	case *ast.PropagateExpr:
		return a.exprRecurses(fd, e.Value)
	case *ast.UnaryExpr:
		return a.exprRecurses(fd, e.Value)
	case *ast.MemberExpr:
		return a.exprRecurses(fd, e.Object)
	case *ast.IndexExpr:
		return a.exprRecurses(fd, e.Array) || a.exprRecurses(fd, e.Index)
	case *ast.TupleExpr:
		return a.anyRecurses(fd, e.Elements)
	default:
		return false
	}
}

func (a *Analyser) callsItself(fd *ast.FuncDecl, call *ast.CallExpr) bool {
	if fd.Receiver == nil {
		id, ok := call.Name.(*ast.IdentLiteral)
		if !ok {
			return false
		}
		sym := a.info.Idents[id]
		return sym != nil && sym.Kind == FUNC && sym.Name == fd.Name
	}
	member, ok := call.Name.(*ast.MemberExpr)
	method := a.info.MethodCalls[call]
	return ok && method != nil && member.Field == fd.Name && method.Owner == fd.Receiver.Type.Name && method.Module == a.module
}

func mayLeave(stmt ast.Statement) bool {
	found := false
	var visit func(v reflect.Value)
	visit = func(v reflect.Value) {
		if found {
			return
		}
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				return
			}
			switch n := v.Interface().(type) {
			case *ast.FuncLiteral:
				return
			case *ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.YieldStmt:
				found = true
				return
			case *ast.PropagateExpr:
				if !n.Assert {
					found = true
					return
				}
			}
			visit(v.Elem())
		case reflect.Interface:
			if !v.IsNil() {
				visit(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		}
	}
	visit(reflect.ValueOf(stmt))
	return found
}
