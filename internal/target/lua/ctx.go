package lua

import "fmt"

func (t *LuaTarget) newLabel(prefix string) string {
	label := fmt.Sprintf("__wisp_%s_%d", prefix, t.labelCounter)
	t.labelCounter++
	return label
}

type loopContext struct {
	Label         string
	BreakLabel    string
	ContinueLabel string
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

func (t *LuaTarget) findLoop(label string) *loopContext {
	for i := len(t.loopStack) - 1; i >= 0; i-- {
		if t.loopStack[i].Label == label {
			return &t.loopStack[i]
		}
	}
	return nil
}
