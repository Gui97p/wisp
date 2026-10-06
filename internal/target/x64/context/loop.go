package x64context

type LoopContext struct {
	Label            string
	BreakLabel       string
	ContinueLabel    string
	SupportsContinue bool
}

func (t *Context) PushLoop(ctx LoopContext) {
	t.loopStack = append(t.loopStack, ctx)
}

func (t *Context) PopLoop() {
	t.loopStack = t.loopStack[:len(t.loopStack)-1]
}

func (t *Context) CurrentLoop() *LoopContext {
	if len(t.loopStack) == 0 {
		return nil
	}

	return &t.loopStack[len(t.loopStack)-1]
}

func (t *Context) CurrentContinuable() *LoopContext {
	for i := len(t.loopStack) - 1; i >= 0; i-- {
		if t.loopStack[i].SupportsContinue {
			return &t.loopStack[i]
		}
	}
	return nil
}

func (t *Context) FindLoop(label string) *LoopContext {
	for i := len(t.loopStack) - 1; i >= 0; i-- {
		if t.loopStack[i].Label == label {
			return &t.loopStack[i]
		}
	}
	return nil
}
