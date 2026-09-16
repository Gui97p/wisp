package lua

import "fmt"

var labelCounter uint64

func newLabel(prefix string) string {
	label := fmt.Sprintf("__wisp_%s_%d", prefix, labelCounter)
	labelCounter++
	return label
}

type loopContext struct {
	Label         string
	BreakLabel    string
	ContinueLabel string
}

var loopStack []loopContext

func pushLoop(ctx loopContext) {
	loopStack = append(loopStack, ctx)
}

func popLoop() {
	loopStack = loopStack[:len(loopStack)-1]
}

func currentLoop() *loopContext {
	if len(loopStack) == 0 {
		return nil
	}

	return &loopStack[len(loopStack)-1]
}

func findLoop(label string) *loopContext {
	for i := len(loopStack) - 1; i >= 0; i-- {
		if loopStack[i].Label == label {
			return &loopStack[i]
		}
	}
	return nil
}
