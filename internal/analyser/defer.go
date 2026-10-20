package analyser

import (
	"reflect"

	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) checkDeferStmt(s *ast.DeferStmt) {
	if s.OnError && !a.currentFallible {
		a.error(s, "defer! needs a function that can fail").
			WithHelp("mark the function with ! before its return list, or use defer")
	}

	a.rejectDeferredExits(s)

	a.enterScope()
	switch body := s.Body.(type) {
	case *ast.BlockStmt:
		a.checkBlock(body)
	default:
		a.checkStmt(body)
	}
	a.exitScope()

	if !s.OnError {
		a.recordDeferredFree(s)
	}
}

func (a *Analyser) recordDeferredFree(d *ast.DeferStmt) {
	stmt, ok := d.Body.(*ast.ExpressionStmt)
	if !ok {
		return
	}
	call, ok := stmt.Expr.(*ast.CallExpr)
	if !ok {
		return
	}
	if name, isIdent := call.Name.(*ast.IdentLiteral); !isIdent || name.Value != "free" {
		return
	}

	if a.deferredFree == nil {
		a.deferredFree = map[*Symbol]*ast.DeferStmt{}
	}
	for _, arg := range call.Args {
		id, isIdent := arg.(*ast.IdentLiteral)
		if !isIdent {
			continue
		}
		if sym, found := a.scope.Resolve(id.Value); found {
			a.deferredFree[sym] = d
		}
	}
}

func (a *Analyser) warnReturnedFree(stmt *ast.ReturnStmt) {
	for _, value := range stmt.Values {
		id, ok := value.(*ast.IdentLiteral)
		if !ok {
			continue
		}
		sym, found := a.scope.Resolve(id.Value)
		if !found {
			continue
		}
		d, deferred := a.deferredFree[sym]
		if !deferred {
			continue
		}

		line, col := d.Position()
		endLine, endCol := d.EndPosition()
		a.warnf(id, "%s is returned, but a deferred free releases it first", id.Value).
			Note(a.currentFile, line, col, endLine, endCol, "freed here").
			WithHelp("use defer! to free it only when the function fails")
	}
}

func (a *Analyser) rejectDeferredExits(d *ast.DeferStmt) {
	labels := map[string]bool{}
	var collect func(v reflect.Value)
	collect = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				return
			}
			switch n := v.Interface().(type) {
			case *ast.FuncLiteral:
				return
			case *ast.ForStmt:
				if n.Label != "" {
					labels[n.Label] = true
				}
			case *ast.LoopStmt:
				if n.Label != "" {
					labels[n.Label] = true
				}
			}
			collect(v.Elem())
		case reflect.Interface:
			if !v.IsNil() {
				collect(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				collect(v.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				collect(v.Index(i))
			}
		}
	}
	collect(reflect.ValueOf(d.Body))

	var walk func(v reflect.Value, loops int)
	walk = func(v reflect.Value, loops int) {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				return
			}
			next := loops
			switch n := v.Interface().(type) {
			case *ast.FuncLiteral:
				return
			case *ast.ReturnStmt:
				a.error(n, "a deferred statement cannot return").
					WithHelp("deferred code runs while the block is already being left")
			case *ast.BreakStmt:
				if (n.Label == "" && loops == 0) || (n.Label != "" && !labels[n.Label]) {
					a.error(n, "a deferred statement cannot break out of its block")
				}
			case *ast.ContinueStmt:
				if (n.Label == "" && loops == 0) || (n.Label != "" && !labels[n.Label]) {
					a.error(n, "a deferred statement cannot continue a loop outside of it")
				}
			case *ast.PropagateExpr:
				a.error(n, "a deferred statement cannot propagate an error").
					WithHelp("handle it here, for instance with ??")
			case *ast.ForStmt, *ast.LoopStmt:
				next++
			case *ast.IfStmt:
				if n.IsSwitch {
					next++
				}
			}
			walk(v.Elem(), next)
		case reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem(), loops)
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i), loops)
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), loops)
			}
		}
	}
	walk(reflect.ValueOf(d.Body), 0)
}
