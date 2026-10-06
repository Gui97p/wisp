package x64

import (
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileForRange(stmt *ast.ForStmt) error {
	return nil
}

func (t *X64Target) compileForNumeric(stmt *ast.ForStmt) error {
	tp := t.info.Types[stmt.Start]
	size := sizeOf(tp)

	start, err := t.compileExpr(stmt.Start)
	if err != nil {
		return err
	}

	startOffset, _ := t.ctx.Get(t.info.VarSymbols[stmt][0])
	startSlot := slot(startOffset, size)
	t.store(start, startSlot)

	end, err := t.compileExpr(stmt.End)
	if err != nil {
		return err
	}

	endOffset := t.ctx.Reserve(8)
	endSlot := slot(endOffset, size)
	t.store(end, endSlot)

	stepOffset := t.ctx.Reserve(8)
	stepSlot := slot(stepOffset, size)
	if stmt.Step != nil {
		step, err := t.compileExpr(stmt.Step)
		if err != nil {
			return err
		}
		t.store(step, stepSlot)
		stepOk := t.newLabel("step_ok")
		t.text.printft("cmp %s, 0\n", t.memText(stepSlot))
		t.text.printft("jg %s\n", stepOk)
		t.panic("range step must be positive")
		t.text.printf("%s:\n", stepOk)
	} else {
		t.store(Imm(1), stepSlot)
	}

	reg := t.ctx.AllocFreeRegister()
	regStr := t.ctx.GetRegister(reg, size)

	condLabel := t.newLabel("cond")
	dirOffset := t.ctx.Reserve(8)
	t.text.printft("mov qword [rbp-%d], 0\n", dirOffset)
	t.text.printft("mov %s, %s\n", regStr, t.memText(startSlot))
	t.text.printft("cmp %s, %s\n", regStr, t.memText(endSlot))
	t.text.printft("setle byte [rbp-%d]\n", dirOffset)
	t.text.printft("jle %s\n", condLabel)
	t.text.printft("neg %s\n", t.memText(stepSlot))
	t.ctx.FreeRegister(reg)

	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.ctx.PushLoop(ctx)
	defer t.ctx.PopLoop()

	t.text.printf("%s:\n", condLabel)
	left := t.materialize(startSlot, size)
	leftStr := t.ctx.GetRegister(left, size)

	down := t.newLabel("down")
	body := t.newLabel("body")

	t.text.printft("cmp qword [rbp-%d], 0\n", dirOffset)
	t.text.printft("je %s\n", down)
	t.text.printft("cmp %s, %s\n", leftStr, t.memText(endSlot))
	t.text.printft("jg %s\n", ctx.BreakLabel)
	t.text.printft("jmp %s\n", body)

	t.text.printf("%s:\n", down)
	t.text.printft("cmp %s, %s\n", leftStr, t.memText(endSlot))
	t.text.printft("jl %s\n", ctx.BreakLabel)

	t.ctx.FreeRegister(left)
	t.text.printf("%s:\n", body)
	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)
	stepReg := t.materialize(stepSlot, size)
	t.text.printft("add %s, %s\n", t.memText(startSlot), t.ctx.GetRegister(stepReg, size))
	t.ctx.FreeRegister(stepReg)
	t.text.printft("jmp %s\n", condLabel)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}

func (t *X64Target) compileForCount(stmt *ast.ForStmt) error {
	tp := t.info.Types[stmt.End]
	size := sizeOf(tp)

	op, err := t.compileExpr(stmt.End)
	if err != nil {
		return err
	}

	offset := t.ctx.Reserve(8)
	memSlot := slot(offset, size)
	t.store(op, memSlot)

	condLabel := t.newLabel("cond")
	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.ctx.PushLoop(ctx)
	defer t.ctx.PopLoop()

	t.text.printf("%s:\n", condLabel)
	t.text.printft("cmp %s, 0\n", t.memText(memSlot))
	if isSigned(tp) {
		t.text.printt("jle ")
	} else {
		t.text.printt("je ")
	}
	t.text.println(ctx.BreakLabel)

	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)
	t.text.printft("dec %s\n", t.memText(memSlot))
	t.text.printft("jmp %s\n", condLabel)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}
