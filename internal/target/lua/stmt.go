package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileStatement(b *strings.Builder, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return t.compileBlockStatement(b, s)
	case *ast.GroupStmt:
		return t.compileGroupStatement(b, s)
	case *ast.ConstStmt:
		return t.compileDefinition(b, s, s.Vars, s.Values)
	case *ast.VarStmt:
		return t.compileDefinition(b, s, s.Vars, s.Values)
	case *ast.ReturnStmt:
		return t.compileReturnStatement(b, s)
	case *ast.ExpressionStmt:
		return t.compileExpressionStatement(b, s)
	case *ast.IfStmt:
		return t.compileIfStatement(b, s)
	case *ast.ForStmt:
		return t.compileForStatement(b, s)
	case *ast.LoopStmt:
		return t.compileLoopStatement(b, s)
	case *ast.BreakStmt:
		return t.compileBreakStatement(b, s)
	case *ast.ContinueStmt:
		return t.compileContinueStatement(b, s)
	case *ast.AssignStmt:
		return t.compileAssignStatement(b, s)
	case *ast.IncDecStmt:
		return t.compileIncDecStatement(b, s)
	default:
		return fmt.Errorf("lua: unsupported statement %T", stmt)
	}
}

func (t *LuaTarget) compileBlockStatement(b *strings.Builder, block *ast.BlockStmt) error {
	for _, stmt := range block.Statements {
		if err := t.compileStatement(b, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (t *LuaTarget) compileGroupStatement(b *strings.Builder, group *ast.GroupStmt) error {
	for _, stmt := range group.Statements {
		if err := t.compileStatement(b, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (t *LuaTarget) compileReturnStatement(b *strings.Builder, stmt *ast.ReturnStmt) error {
	if result, ok := t.currentCoalesceResult(); ok {
		text, err := t.compileExprScratch(stmt.Values[0])
		if err != nil {
			return err
		}

		t.flushPending(b)

		fmt.Fprintf(b, "%s = %s\n", result, text)
		return nil
	}

	texts := make([]string, len(stmt.Values))
	for k, value := range stmt.Values {
		text, err := t.compileExprScratch(value)
		if err != nil {
			return err
		}

		if k == t.currentFallibleIndex {
			if isError(t.info.Types[value]) {
				text = fmt.Sprintf("{value = nil, err = %s}", text)
			} else {
				text = fmt.Sprintf("{value = %s, err = nil}", text)
			}
		}
		texts[k] = text
	}

	t.flushPending(b)

	b.WriteString("return ")
	b.WriteString(strings.Join(texts, ", "))
	b.WriteRune('\n')
	return nil
}

func (t *LuaTarget) compileExpressionStatement(b *strings.Builder, stmt *ast.ExpressionStmt) error {
	text, err := t.compileExprScratch(stmt.Expr)
	if err != nil {
		return err
	}

	t.flushPending(b)

	b.WriteString(text)
	b.WriteRune('\n')
	return nil
}

func (t *LuaTarget) compileIfStatement(b *strings.Builder, stmt *ast.IfStmt) error {
	cond, err := t.compileExprScratch(stmt.Condition)
	if err != nil {
		return err
	}

	t.flushPending(b)

	b.WriteString("if ")
	b.WriteString(cond)
	b.WriteString(" then\n")
	t.compileStatement(b, stmt.Then)

	if stmt.Else != nil {
		b.WriteString("else\n")
		t.compileStatement(b, stmt.Else)
	}

	b.WriteString("end\n")

	return nil
}

func (t *LuaTarget) compileForStatement(b *strings.Builder, stmt *ast.ForStmt) error {
	switch {
	case stmt.Range != nil:
		return t.compileForRange(b, stmt)

	case stmt.Start != nil:
		return t.compileForNumeric(b, stmt)

	default:
		return t.compileForCount(b, stmt)
	}

}

func (t *LuaTarget) compileForCount(b *strings.Builder, stmt *ast.ForStmt) error {
	countVar := t.newLabel("count")
	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: t.newLabel("for_continue"),
		BreakLabel:    t.newLabel("for_break"),
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	end, err := t.compileExprScratch(stmt.End)
	if err != nil {
		return err
	}
	t.flushPending(b)

	fmt.Fprintf(b, "local %s = %s\n", countVar, end)

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"if %s <= 0 then goto %s end\n",
		countVar,
		ctx.BreakLabel,
	)

	b.WriteString("do\n")

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "%s = %s - 1\n", countVar, countVar)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileForRange(b *strings.Builder, stmt *ast.ForStmt) error {
	rangeVar := t.newLabel("range")
	keyVar := t.newLabel("range_key")
	valueVar := t.newLabel("range_value")

	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: t.newLabel("for_continue"),
		BreakLabel:    t.newLabel("for_break"),
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	rangeText, err := t.compileExprScratch(stmt.Range)
	if err != nil {
		return err
	}
	t.flushPending(b)

	fmt.Fprintf(b, "local %s = %s\n", rangeVar, rangeText)

	fmt.Fprintf(b, "local %s = nil\n", keyVar)
	fmt.Fprintf(b, "local %s = nil\n", valueVar)

	if stmt.Var != "" {
		fmt.Fprintf(b, "local %s = nil\n", stmt.Var)
	}

	if stmt.Var2 != "" {
		fmt.Fprintf(b, "local %s = nil\n", stmt.Var2)
	}

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"%s, %s = next(%s, %s)\n",
		keyVar,
		valueVar,
		rangeVar,
		keyVar,
	)

	fmt.Fprintf(
		b,
		"if %s == nil then goto %s end\n",
		keyVar,
		ctx.BreakLabel,
	)

	b.WriteString("do\n")

	if stmt.Var != "" {
		if stmt.Var2 != "" {
			fmt.Fprintf(b, "%s = %s\n", stmt.Var, keyVar)
			fmt.Fprintf(b, "%s = %s\n", stmt.Var2, valueVar)
		} else {
			fmt.Fprintf(b, "%s = %s\n", stmt.Var, keyVar)
		}
	}

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)

	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileForNumeric(b *strings.Builder, stmt *ast.ForStmt) error {
	conditionLabel := t.newLabel("for_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: t.newLabel("for_continue"),
		BreakLabel:    t.newLabel("for_break"),
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	start := "1"
	if stmt.Start != nil {
		text, err := t.compileExprScratch(stmt.Start)
		if err != nil {
			return err
		}
		t.flushPending(b)
		start = text
	}
	fmt.Fprintf(b, "local %s = %s\n", stmt.Var, start)

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	end, err := t.compileExprScratch(stmt.End)
	if err != nil {
		return err
	}
	t.flushPending(b)
	fmt.Fprintf(b, "if %s > %s then goto %s end\n", stmt.Var, end, ctx.BreakLabel)

	b.WriteString("do\n")

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)

	step := "1"
	if stmt.Step != nil {
		text, err := t.compileExprScratch(stmt.Step)
		if err != nil {
			return err
		}
		t.flushPending(b)
		step = text
	}
	fmt.Fprintf(b, "%s = %s + %s\n", stmt.Var, stmt.Var, step)

	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileLoopStatement(b *strings.Builder, stmt *ast.LoopStmt) error {
	conditionLabel := t.newLabel("loop_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: t.newLabel("loop_continue"),
		BreakLabel:    t.newLabel("loop_break"),
	}

	t.pushLoop(ctx)
	defer t.popLoop()

	fmt.Fprintf(b, "::%s::\n", conditionLabel)

	if stmt.Condition != nil {
		cond, err := t.compileExprScratch(stmt.Condition)
		if err != nil {
			return err
		}
		t.flushPending(b)
		fmt.Fprintf(b, "if not (%s) then goto %s end\n", cond, ctx.BreakLabel)
	}

	if err := t.compileStatement(b, stmt.Body); err != nil {
		return err
	}

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)

	if stmt.UntilCondition != nil {
		cond, err := t.compileExprScratch(stmt.UntilCondition)
		if err != nil {
			return err
		}
		t.flushPending(b)
		fmt.Fprintf(b, "if %s then goto %s end\n", cond, ctx.BreakLabel)
	}

	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func (t *LuaTarget) compileBreakStatement(b *strings.Builder, stmt *ast.BreakStmt) error {
	var loop *loopContext

	if stmt.Label == "" {
		loop = t.currentLoop()
	} else {
		loop = t.findLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("break: no matching loop")
	}

	fmt.Fprintf(b, "goto %s\n", loop.BreakLabel)

	return nil
}

func (t *LuaTarget) compileContinueStatement(b *strings.Builder, stmt *ast.ContinueStmt) error {
	var loop *loopContext

	if stmt.Label == "" {
		loop = t.currentLoop()
	} else {
		loop = t.findLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("continue: no matching loop")
	}

	fmt.Fprintf(b, "goto %s\n", loop.ContinueLabel)

	return nil
}

func (t *LuaTarget) compileAssignStatement(b *strings.Builder, stmt *ast.AssignStmt) error {
	target, err := t.compileExprScratch(stmt.Target)
	if err != nil {
		return err
	}
	value, err := t.compileExprScratch(stmt.Value)
	if err != nil {
		return err
	}

	t.flushPending(b)

	b.WriteString(target)
	b.WriteString(" = ")

	if stmt.Op == "=" {
		b.WriteString(value)
	} else {
		op := strings.TrimSuffix(stmt.Op, "=")
		switch op {
		case "^":
			op = "~"
		case "+":
			if pt, ok := t.info.Types[stmt.Target].(analyser.PrimitiveType); ok && pt.Name == "string" {
				op = ".."
				if pt, ok := t.info.Types[stmt.Value].(analyser.PrimitiveType); ok && pt.Name == "char" {
					value = fmt.Sprintf("string.char(%s)", value)
				}
			}
		}
		fmt.Fprintf(b, "(%s) %s (%s)", target, op, value)
	}
	b.WriteByte('\n')

	return nil
}

func (t *LuaTarget) compileIncDecStatement(b *strings.Builder, stmt *ast.IncDecStmt) error {
	if err := t.compileExpression(b, stmt.Target); err != nil {
		return err
	}
	b.WriteString(" = ")
	if err := t.compileExpression(b, stmt.Target); err != nil {
		return err
	}

	op := stmt.Op[0]
	fmt.Fprintf(b, " %c 1\n", op)

	return nil
}
