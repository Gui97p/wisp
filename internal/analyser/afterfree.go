package analyser

import (
	"reflect"

	"github.com/Gui97p/wisp/internal/ast"
)

type freedVariable struct {
	index int
	sym   *Symbol
	stmt  ast.Statement
}

func (a *Analyser) freedSymbol(stmt ast.Statement) *Symbol {
	expression, ok := stmt.(*ast.ExpressionStmt)
	if !ok {
		return nil
	}
	call, ok := expression.Expr.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return nil
	}
	callee := call.Name
	if call.Generic != nil {
		callee = call.Generic
	}
	member, ok := callee.(*ast.MemberExpr)
	if !ok || member.Field != "free" {
		return nil
	}

	arg := call.Args[len(call.Args)-1]
	for {
		cast, isCast := arg.(*ast.CastExpr)
		if !isCast || !cast.Implicit {
			break
		}
		arg = cast.Value
	}
	id, ok := arg.(*ast.IdentLiteral)
	if !ok {
		return nil
	}
	sym := a.info.Idents[id]
	if sym == nil || (sym.Kind != VAR && sym.Kind != PARAM) {
		return nil
	}
	switch sym.Type.(type) {
	case PointerType, SpanType, *InterfaceType:
		return sym
	}
	return nil
}

func (a *Analyser) warnUseAfterFree(stmts []ast.Statement, frees []freedVariable) {
	for _, freed := range frees {
		for _, later := range stmts[freed.index+1:] {
			if use := a.findUse(later, freed.sym); use != nil {
				line, col := freed.stmt.Position()
				endLine, endCol := freed.stmt.EndPosition()
				a.warnf(use, "%s is used after it was freed", freed.sym.Name).
					Note(a.currentFile, line, col, endLine, endCol, "freed here").
					WithHelp("give %s a new value first, or move the free after its last use", freed.sym.Name)
				break
			}
			if a.revives(later, freed.sym) {
				break
			}
			switch later.(type) {
			case *ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.YieldStmt:
			default:
				continue
			}
			break
		}
	}
}

func (a *Analyser) revives(stmt ast.Statement, sym *Symbol) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || assign.Op != "=" {
		return false
	}
	for _, target := range assign.Targets {
		if id, isIdent := target.(*ast.IdentLiteral); isIdent && a.info.Idents[id] == sym {
			return true
		}
	}
	return false
}

func (a *Analyser) findUse(stmt ast.Statement, sym *Symbol) ast.Node {
	var found ast.Node
	if assign, ok := stmt.(*ast.AssignStmt); ok && assign.Op == "=" {
		for _, value := range assign.Values {
			if found = a.findUseIn(reflect.ValueOf(value), sym); found != nil {
				return found
			}
		}
		for _, target := range assign.Targets {
			if _, isIdent := target.(*ast.IdentLiteral); isIdent {
				continue
			}
			if found = a.findUseIn(reflect.ValueOf(target), sym); found != nil {
				return found
			}
		}
		return nil
	}
	return a.findUseIn(reflect.ValueOf(stmt), sym)
}

func (a *Analyser) findUseIn(v reflect.Value, sym *Symbol) ast.Node {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		if id, ok := v.Interface().(*ast.IdentLiteral); ok {
			if a.info.Idents[id] == sym {
				return id
			}
			return nil
		}
		if call, ok := v.Interface().(*ast.CallExpr); ok && a.isLengthOf(call, sym) {
			return nil
		}
		if binary, ok := v.Interface().(*ast.BinaryExpr); ok && (binary.Operator == "==" || binary.Operator == "!=") {
			return a.findUseBesideComparison(binary, sym)
		}
		return a.findUseIn(v.Elem(), sym)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			if found := a.findUseIn(v.Field(i), sym); found != nil {
				return found
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if found := a.findUseIn(v.Index(i), sym); found != nil {
				return found
			}
		}
	}
	return nil
}

func (a *Analyser) isLengthOf(call *ast.CallExpr, sym *Symbol) bool {
	name, ok := call.Name.(*ast.IdentLiteral)
	if !ok || name.Value != "len" || len(call.Args) != 1 {
		return false
	}
	arg, ok := call.Args[0].(*ast.IdentLiteral)
	return ok && a.info.Idents[arg] == sym
}

func (a *Analyser) findUseBesideComparison(binary *ast.BinaryExpr, sym *Symbol) ast.Node {
	for _, side := range []ast.Expression{binary.Left, binary.Right} {
		if id, ok := side.(*ast.IdentLiteral); ok && a.info.Idents[id] == sym {
			continue
		}
		if found := a.findUseIn(reflect.ValueOf(side), sym); found != nil {
			return found
		}
	}
	return nil
}
