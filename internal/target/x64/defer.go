package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
)

type deferScope struct {
	entries []*ast.DeferStmt
}

func (t *X64Target) pushDeferScope() {
	t.deferScopes = append(t.deferScopes, &deferScope{})
}

func (t *X64Target) popDeferScope() {
	t.deferScopes = t.deferScopes[:len(t.deferScopes)-1]
}

func (t *X64Target) registerDefer(d *ast.DeferStmt) {
	scope := t.deferScopes[len(t.deferScopes)-1]
	scope.entries = append(scope.entries, d)
}

func (t *X64Target) hasDefers(from int, onError bool) bool {
	for i := from; i < len(t.deferScopes); i++ {
		for _, d := range t.deferScopes[i].entries {
			if onError || !d.OnError {
				return true
			}
		}
	}
	return false
}

func (t *X64Target) emitDefers(from int, onError bool, errSlot ...Mem) error {
	for i := len(t.deferScopes) - 1; i >= from; i-- {
		entries := t.deferScopes[i].entries
		for _, d := range slices.Backward(entries) {
			if d.OnError && !onError {
				continue
			}

			if d.ErrorBind != "" && len(errSlot) > 0 {
				bind := t.ctx.Set(t.info.VarSymbols[d][0], 8)
				t.store(errSlot[0], slot(bind, 8))
			}

			if d.Fallible && t.deferError != nil {
				end := t.newLabel("defer_end")
				t.deferExits = append(t.deferExits, end)
				err := t.compileStatement(d.Body)
				t.deferExits = t.deferExits[:len(t.deferExits)-1]
				if err != nil {
					return err
				}
				t.text.printf("%s:\n", end)
				continue
			}

			if err := t.compileStatement(d.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasFallibleDefer(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLiteral:
			return false
		case *ast.DeferStmt:
			if node.Fallible {
				found = true
			}
		}
		return !found
	})
	return found
}

func (t *X64Target) failDeferred(errOp Operand) error {
	if t.ctx.Pushed() != 0 {
		return fmt.Errorf("x86-64: a deferred ! cannot run while arguments are pushed")
	}
	end := t.deferExits[len(t.deferExits)-1]
	t.text.printft("cmp qword %s, 0\n", t.memText(*t.deferError))
	t.text.printft("jne %s\n", end)
	t.store(errOp, *t.deferError)
	t.text.printft("jmp %s\n", end)
	return nil
}
