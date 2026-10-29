package lua

import (
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

type deferScope struct {
	entries []*ast.DeferStmt
}

func (t *LuaTarget) pushDeferScope() {
	t.deferScopes = append(t.deferScopes, &deferScope{})
}

func (t *LuaTarget) popDeferScope() {
	t.deferScopes = t.deferScopes[:len(t.deferScopes)-1]
}

func (t *LuaTarget) registerDefer(d *ast.DeferStmt) {
	scope := t.deferScopes[len(t.deferScopes)-1]
	scope.entries = append(scope.entries, d)
}

func (t *LuaTarget) hasDefers(from int, onError bool) bool {
	for i := from; i < len(t.deferScopes); i++ {
		for _, d := range t.deferScopes[i].entries {
			if onError || !d.OnError {
				return true
			}
		}
	}
	return false
}

func (t *LuaTarget) emitDefers(b *strings.Builder, from int, onError bool, errValue ...string) error {
	for i := len(t.deferScopes) - 1; i >= from; i-- {
		entries := t.deferScopes[i].entries
		for j := len(entries) - 1; j >= 0; j-- {
			d := entries[j]
			if d.OnError && !onError {
				continue
			}

			outer := t.pending
			t.pending = nil
			b.WriteString("do\n")
			if d.ErrorBind != "" && len(errValue) > 0 {
				b.WriteString("local ")
				b.WriteString(luaIdent(d.ErrorBind))
				b.WriteString(" = ")
				b.WriteString(errValue[0])
				b.WriteString("\n")
			}
			err := t.compileStatement(b, d.Body)
			b.WriteString("end\n")
			t.pending = outer
			if err != nil {
				return err
			}
		}
	}
	return nil
}
