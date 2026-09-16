package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileStatement(b *strings.Builder, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return compileBlockStatement(b, s)
	case *ast.VarStmt:
		return compileVarStatement(b, s)
	case *ast.ReturnStmt:
		return compileReturnStatement(b, s)
	case *ast.ExpressionStmt:
		return compileExpressionStatement(b, s)
	case *ast.IfStmt:
		return compileIfStatement(b, s)
	case *ast.ForStmt:
		return compileForStatement(b, s)
	// case *ast.LoopStmt:
	// 	return compileLoopStatement(b, s)
	case *ast.BreakStmt:
		return compileBreakStatement(b, s)
	case *ast.ContinueStmt:
		return compileContinueStatement(b, s)
	// case *ast.AssignStmt:
	// 	return compileAssignStatement(b, s)
	// case *ast.IncDecStmt:
	// 	return compileIncDecStatement(b, s)
	default:
		return fmt.Errorf("lua: unsupported statement %T", stmt)
	}
}

func compileBlockStatement(b *strings.Builder, block *ast.BlockStmt) error {
	for _, stmt := range block.Statements {
		if err := compileStatement(b, stmt); err != nil {
			return err
		}
	}
	return nil
}

func compileVarStatement(b *strings.Builder, stmt *ast.VarStmt) error {
	b.WriteString("local ")
	for k, variable := range stmt.Vars {
		b.WriteString(variable.Name)
		if k != len(stmt.Vars)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteString(" = ")
	for k, value := range stmt.Values {
		compileExpression(b, value)
		if k != len(stmt.Values)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte('\n')
	return nil
}

func compileReturnStatement(b *strings.Builder, stmt *ast.ReturnStmt) error {
	b.WriteString("return ")
	for k, value := range stmt.Values {
		if err := compileExpression(b, value); err != nil {
			return err
		}
		if k != len(stmt.Values)-1 {
			b.WriteByte(',')
		}
	}

	b.WriteRune('\n')
	return nil
}

func compileExpressionStatement(b *strings.Builder, stmt *ast.ExpressionStmt) error {
	if err := compileExpression(b, stmt.Expr); err != nil {
		return err
	}
	b.WriteRune('\n')
	return nil
}

func compileIfStatement(b *strings.Builder, stmt *ast.IfStmt) error {
	b.WriteString("if ")
	compileExpression(b, stmt.Condition)
	b.WriteString(" then\n")
	compileStatement(b, stmt.Then)

	if stmt.Else != nil {
		b.WriteString("else\n")
		compileStatement(b, stmt.Else)
	}

	b.WriteString("end\n")

	return nil
}

func compileForStatement(b *strings.Builder, stmt *ast.ForStmt) error {
	switch {
	case stmt.Range != nil:
		return compileForRange(b, stmt)

	case stmt.Start != nil:
		return compileForNumeric(b, stmt)

	default:
		return compileForCount(b, stmt)
	}

}

func compileForCount(b *strings.Builder, stmt *ast.ForStmt) error {
	countVar := newLabel("count")
	conditionLabel := newLabel("for_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: newLabel("for_continue"),
		BreakLabel:    newLabel("for_break"),
	}

	pushLoop(ctx)
	defer popLoop()

	fmt.Fprintf(b, "local %s = ", countVar)

	if err := compileExpression(b, stmt.End); err != nil {
		return err
	}

	fmt.Fprintf(b, "\n::%s::\n", conditionLabel)

	fmt.Fprintf(
		b,
		"if %s <= 0 then goto %s end\n",
		countVar,
		ctx.BreakLabel,
	)

	if err := compileStatement(b, stmt.Body); err != nil {
		return err
	}

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "%s = %s - 1\n", countVar, countVar)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func compileForRange(b *strings.Builder, stmt *ast.ForStmt) error {
	rangeVar := newLabel("range")
	keyVar := newLabel("range_key")
	valueVar := newLabel("range_value")

	conditionLabel := newLabel("for_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: newLabel("for_continue"),
		BreakLabel:    newLabel("for_break"),
	}

	pushLoop(ctx)
	defer popLoop()

	fmt.Fprintf(b, "local %s = ", rangeVar)

	if err := compileExpression(b, stmt.Range); err != nil {
		return err
	}

	b.WriteByte('\n')

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

	if err := compileStatement(b, stmt.Body); err != nil {
		return err
	}

	b.WriteString("end\n")

	fmt.Fprintf(b, "::%s::\n", ctx.ContinueLabel)
	fmt.Fprintf(b, "goto %s\n", conditionLabel)

	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func compileForNumeric(b *strings.Builder, stmt *ast.ForStmt) error {
	conditionLabel := newLabel("for_condition")

	ctx := loopContext{
		Label:         stmt.Label,
		ContinueLabel: newLabel("for_continue"),
		BreakLabel:    newLabel("for_break"),
	}

	pushLoop(ctx)
	defer popLoop()

	fmt.Fprintf(b, "local %s = ", stmt.Var)

	if stmt.Start == nil {
		b.WriteByte('1')
	} else if err := compileExpression(b, stmt.Start); err != nil {
		return err
	}
	b.WriteByte('\n')

	fmt.Fprintf(b, "::%s::\n", conditionLabel)
	fmt.Fprintf(b, "if %s > ", stmt.Var)
	compileExpression(b, stmt.End)
	fmt.Fprintf(b, " then goto %s end", ctx.BreakLabel)

	if err := compileStatement(b, stmt.Body); err != nil {
		return err
	}

	fmt.Fprintf(b, "::%s::", ctx.ContinueLabel)

	fmt.Fprintf(b, "%s = %s + ", stmt.Var, stmt.Var)

	if stmt.Step == nil {
		b.WriteByte('1')
	} else if err := compileExpression(b, stmt.Step); err != nil {
		return err
	}
	b.WriteByte('\n')

	fmt.Fprintf(b, "goto %s\n", conditionLabel)
	fmt.Fprintf(b, "::%s::\n", ctx.BreakLabel)

	return nil
}

func compileBreakStatement(b *strings.Builder, stmt *ast.BreakStmt) error {
	var loop *loopContext

	if stmt.Label == "" {
		loop = currentLoop()
	} else {
		loop = findLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("break: no matching loop")
	}

	fmt.Fprintf(b, "goto %s\n", loop.BreakLabel)

	return nil
}

func compileContinueStatement(b *strings.Builder, stmt *ast.ContinueStmt) error {
	var loop *loopContext

	if stmt.Label == "" {
		loop = currentLoop()
	} else {
		loop = findLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("continue: no matching loop")
	}

	fmt.Fprintf(b, "goto %s\n", loop.ContinueLabel)

	return nil
}
