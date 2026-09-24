package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) newLabel(prefix string) string {
	label := fmt.Sprintf("__wisp_%s_%d", prefix, t.labelCounter)
	t.labelCounter++
	return label
}

func (t *LuaTarget) compileExprScratch(expr ast.Expression) (string, error) {
	var scratch strings.Builder
	if err := t.compileExpression(&scratch, expr); err != nil {
		return "", err
	}
	return scratch.String(), nil
}

func (t *LuaTarget) emitPending(line string) {
	t.pending = append(t.pending, line)
}

func (t *LuaTarget) flushPending(b *strings.Builder) {
	for _, line := range t.pending {
		b.WriteString(line)
	}
	t.pending = t.pending[:0]
}

type loopContext struct {
	Label            string
	BreakLabel       string
	ContinueLabel    string
	SupportsContinue bool
}

func (t *LuaTarget) pushLoop(ctx loopContext) {
	t.loopStack = append(t.loopStack, ctx)
}

func (t *LuaTarget) popLoop() {
	t.loopStack = t.loopStack[:len(t.loopStack)-1]
}

func (t *LuaTarget) currentLoop() *loopContext {
	if len(t.loopStack) == 0 {
		return nil
	}

	return &t.loopStack[len(t.loopStack)-1]
}

func (t *LuaTarget) currentContinuable() *loopContext {
	for i := len(t.loopStack) - 1; i >= 0; i-- {
		if t.loopStack[i].SupportsContinue {
			return &t.loopStack[i]
		}
	}
	return nil
}

func (t *LuaTarget) findLoop(label string) *loopContext {
	for i := len(t.loopStack) - 1; i >= 0; i-- {
		if t.loopStack[i].Label == label {
			return &t.loopStack[i]
		}
	}
	return nil
}

func (t *LuaTarget) pushCoalesceResult(varName string) {
	t.coalesceStack = append(t.coalesceStack, varName)
}

func (t *LuaTarget) popCoalesceResult() {
	t.coalesceStack = t.coalesceStack[:len(t.coalesceStack)-1]
}

func (t *LuaTarget) currentCoalesceResult() (string, bool) {
	if len(t.coalesceStack) == 0 {
		return "", false
	}
	return t.coalesceStack[len(t.coalesceStack)-1], true
}
