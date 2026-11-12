package x64

import (
	"fmt"
	"math/big"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) pushLoop(ctx x64context.LoopContext) {
	ctx.DeferDepth = len(t.deferScopes)
	t.ctx.PushLoop(ctx)
}

func (t *X64Target) constantRange(stmt *ast.ForStmt) (end, step int64, up, ok bool) {
	start, okStart := t.info.Folded(stmt.Start)
	last, okEnd := t.info.Folded(stmt.End)
	if !okStart || !okEnd || start.Kind != analyser.ConstInt || last.Kind != analyser.ConstInt {
		return 0, 0, false, false
	}

	stepValue := big.NewInt(1)
	if stmt.Step != nil {
		s, okStep := t.info.Folded(stmt.Step)
		if !okStep || s.Kind != analyser.ConstInt {
			return 0, 0, false, false
		}
		stepValue = s.Int
	}

	up = stepValue.Sign() > 0
	after := new(big.Int)
	after.Add(last.Int, stepValue)
	if !after.IsInt64() || !last.Int.IsInt64() || !stepValue.IsInt64() {
		return 0, 0, false, false
	}

	end, step = last.Int.Int64(), stepValue.Int64()
	if int64(int32(end)) != end || int64(int32(step)) != step {
		return 0, 0, false, false
	}
	if (up && start.Bits() > last.Bits()) || (!up && start.Bits() < last.Bits()) {
		return 0, 0, false, false
	}
	return end, int64(abs(int(step))), up, true
}

func (t *X64Target) compileForConstant(stmt *ast.ForStmt, end, step int64, up bool) error {
	size := sizeOf(t.info.Types[stmt.Start])

	start, err := t.compileExpr(stmt.Start)
	if err != nil {
		return err
	}
	i := t.local(t.info.VarSymbols[stmt][0])
	t.store(start, i)

	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}
	t.pushLoop(ctx)
	defer t.ctx.PopLoop()

	body := t.newLabel("body")
	t.text.printf("%s:\n", body)
	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)
	instruction, jump := "add", "jle"
	if !up {
		instruction, jump = "sub", "jge"
	}
	t.text.printft("%s %s, %d\n", instruction, t.opText(i, size), step)
	t.text.printft("cmp %s, %d\n", t.opText(i, size), end)
	t.text.printft("%s %s\n", jump, body)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}

func (t *X64Target) compileForRange(stmt *ast.ForStmt) error {
	tp := t.info.Types[stmt.Range]
	base, limit, err := t.baseMem(stmt.Range)
	if err != nil {
		return err
	}

	var p, n, i Operand
	ownsN := true
	st := t.loopSyms[stmt]
	if st == nil {
		p = slot(t.ctx.Reserve(8), 8)
		n = slot(t.ctx.Reserve(8), 8)
		i = slot(t.ctx.Reserve(8), 8)
	} else {
		i = t.hiddenLocal(st.i, 8)
		if st.p != nil {
			p = t.hiddenLocal(st.p, 8)
		}
		ownsN = st.n != nil
		if ownsN {
			n = t.hiddenLocal(st.n, 8)
		} else {
			n = limit
		}
	}

	if p != nil {
		t.store(Addr{Of: base}, p)
	}
	if ownsN {
		t.store(limit, n)
	}
	t.store(Imm(0), i)

	t.freeWord(base)
	t.freeOp(limit)

	cond := t.newLabel("for_condition")
	body := t.newLabel("for_body")
	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.ctx.PopLoop()

	t.text.printf("%s:\n", cond)
	reg := t.readReg(i, 8)
	left := t.ctx.GetRegister(reg, 8)
	right := t.opText(n, 8)
	if c, ok := n.(Imm); (ok && c <= 0) || ownsN {
		t.text.printft("cmp %s, %s\n", left, right)
		t.text.printft("jae %s\n", ctx.BreakLabel)
	}
	t.text.printf("%s:\n", body)

	syms := t.info.VarSymbols[stmt]
	if stmt.Var != "" {
		t.store(i, t.local(syms[0]))
	}

	if stmt.Var2 != "" {
		pReg := t.readReg(p, 8)
		iReg := t.readReg(i, 8)

		elemSize := 1
		switch t := tp.(type) {
		case analyser.ArrayType:
			elemSize = sizeOf(t.Element)
		case analyser.SpanType:
			elemSize = sizeOf(t.Element)
		}
		mem, err := t.elementAt(Mem{Base: pReg}, Reg{iReg, 8}, elemSize)
		if err != nil {
			return err
		}

		t.store(mem, t.local(syms[1]))
		t.freeWord(mem)
	}
	t.ctx.FreeRegister(reg)

	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)
	t.text.printft("inc %s\n", t.opText(i, 8))
	bottom := t.readReg(i, 8)
	t.text.printft("cmp %s, %s\n", t.ctx.GetRegister(bottom, 8), right)
	t.text.printft("jb %s\n", body)
	t.ctx.FreeRegister(bottom)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}

func (t *X64Target) compileForNumeric(stmt *ast.ForStmt) error {
	if end, step, up, ok := t.constantRange(stmt); ok {
		return t.compileForConstant(stmt, end, step, up)
	}

	tp := t.info.Types[stmt.Start]
	size := sizeOf(tp)

	start, err := t.compileExpr(stmt.Start)
	if err != nil {
		return err
	}

	startSlot := t.local(t.info.VarSymbols[stmt][0])
	t.store(start, startSlot)

	end, err := t.compileExpr(stmt.End)
	if err != nil {
		return err
	}

	var endCmp string
	if i, ok := end.(Imm); ok && Imm(int32(i)) == i {
		endCmp = fmt.Sprintf("%d", i)
	} else {
		endOffset := t.ctx.Reserve(8)
		endSlot := slot(endOffset, size)
		t.store(end, endSlot)
		endCmp = t.memText(endSlot)
	}

	var stepValue int64 = 1
	known, up := false, true
	if stmt.Step == nil {
		known = true
	} else if s, ok := t.info.Folded(stmt.Step); ok {
		stepValue = s.Int.Int64()
		known, up = true, s.Int.Sign() > 0
	}

	fast := int64(int32(stepValue)) == stepValue && known

	var stepSlot Mem
	if !fast {
		stepOffset := t.ctx.Reserve(8)
		stepSlot = slot(stepOffset, size)
		var step Operand = Imm(1)
		if stmt.Step != nil {
			step, err = t.compileExpr(stmt.Step)
			if err != nil {
				return err
			}
		}
		t.store(step, stepSlot)
	}

	if stmt.Step != nil && !known && t.checks {
		stepOk := t.newLabel("step_ok")
		t.text.printft("cmp %s, 0\n", t.memText(stepSlot))
		t.text.printft("jne %s\n", stepOk)
		t.panic("range step cannot be zero")
		t.text.printf("%s:\n", stepOk)
	}

	var dirOffset int
	if !known {
		dirOffset = t.ctx.Reserve(8)
		t.text.printft("mov qword [rbp-%d], 0\n", dirOffset)
		t.text.printft("cmp %s, 0\n", t.memText(stepSlot))
		t.text.printft("setg byte [rbp-%d]\n", dirOffset)
	}

	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.ctx.PopLoop()

	cond := t.newLabel("for_condition")
	if !known {
		t.text.printf("%s:\n", cond)
	}

	var left x64context.Register
	var leftStr string
	ownsLeft := false
	if home, ok := startSlot.(Reg); ok {
		leftStr = t.ctx.GetRegister(home.Reg, size)
	} else {
		left = t.materialize(startSlot, size)
		leftStr = t.ctx.GetRegister(left, size)
		ownsLeft = true
	}

	down := t.newLabel("down")
	body := t.newLabel("for_body")

	jDir := "l"
	if known {
		if up {
			jDir = "g"
		}
	} else {
		t.text.printft("cmp qword [rbp-%d], 0\n", dirOffset)
		t.text.printft("je %s\n", down)
		t.text.printft("cmp %s, %s\n", leftStr, endCmp)
		t.text.printft("jg %s\n", ctx.BreakLabel)
		t.text.printft("jmp %s\n", body)

		t.text.printf("%s:\n", down)
	}
	t.text.printft("cmp %s, %s\n", leftStr, endCmp)
	t.text.printft("j%s %s\n", jDir, ctx.BreakLabel)

	if ownsLeft {
		t.ctx.FreeRegister(left)
	}
	t.text.printf("%s:\n", body)
	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)
	var right string
	if fast {
		right = fmt.Sprintf("%d", stepValue)
	} else {
		stepReg := t.materialize(stepSlot, size)
		right = t.ctx.GetRegister(stepReg, size)
		t.ctx.FreeRegister(stepReg)
	}
	t.text.printft("add %s, %s\n", t.opText(startSlot, size), right)
	if known {
		cmpLeft := leftStr
		var reload x64context.Register
		if _, home := startSlot.(Reg); !home {
			reload = t.materialize(startSlot, size)
			cmpLeft = t.ctx.GetRegister(reload, size)
		}
		t.text.printft("cmp %s, %s\n", cmpLeft, endCmp)
		jBack := "l"
		if known && !up {
			jBack = "g"
		}
		t.text.printft("j%se %s\n", jBack, body)
		if reload != x64context.NoReg {
			t.ctx.FreeRegister(reload)
		}
	} else {
		t.text.printft("jmp %s\n", cond)
	}
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

	var counter Operand
	if st := t.loopSyms[stmt]; st != nil {
		counter = t.hiddenLocal(st.count, size)
	} else {
		counter = slot(t.ctx.Reserve(8), size)
	}
	guard := true
	if i, ok := op.(Imm); ok && i > 0 {
		guard = false
	}
	t.store(op, counter)

	bodyLabel := t.newLabel("for_body")
	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("for_continue"),
		BreakLabel:       t.newLabel("for_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.ctx.PopLoop()

	guardInst := "je"
	cmpInst := "jne"
	if isSigned(tp) {
		guardInst = "jle"
		cmpInst = "jg"
	}
	if guard {
		t.text.printft("cmp %s, 0\n", t.opText(counter, size))
		t.text.printft("%s %s\n", guardInst, ctx.BreakLabel)
	}

	t.text.printf("%s:\n", bodyLabel)
	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)
	t.text.printft("dec %s\n", t.opText(counter, size))
	t.text.printft("%s %s\n", cmpInst, bodyLabel)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}
