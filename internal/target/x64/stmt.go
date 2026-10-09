package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileStatement(stmt ast.Statement) error {
	var err error

	switch s := stmt.(type) {
	case *ast.NativeStmt:
		err = t.compileNativeStmt(s)
	case *ast.ExpressionStmt:
		var op Operand
		op, err = t.compileExpr(s.Expr)
		if op != nil {
			t.freeOp(op)
		}
		return err
	case *ast.BlockStmt:
		for _, stmt := range s.Statements {
			err = t.compileStatement(stmt)
			if err != nil {
				break
			}
		}
	case *ast.GroupStmt:
		for _, stmt := range s.Statements {
			err = t.compileStatement(stmt)
			if err != nil {
				break
			}
		}
	case *ast.VarStmt:
		err = t.compileVarStmt(s)
	case *ast.ReturnStmt:
		err = t.compileReturnStmt(s)
	case *ast.AssignStmt:
		err = t.compileAssignStmt(s)
	case *ast.IncDecStmt:
		err = t.compileIncDecStmt(s)
	case *ast.IfStmt:
		err = t.compileIfStmt(s)
	case *ast.ForStmt:
		err = t.compileForStmt(s)
	case *ast.BreakStmt:
		err = t.compileBreakStmt(s)
	case *ast.ContinueStmt:
		err = t.compileContinueStmt(s)
	case *ast.LoopStmt:
		err = t.compileLoopStmt(s)
	case *ast.ConstStmt:
	default:
		return fmt.Errorf("x86-64: unsupported statement %T", stmt)
	}

	if t.err != nil {
		return t.err
	}

	if err == nil && len(t.ctx.AllocatedRegisters()) != 0 {
		line, col := stmt.Position()
		return fmt.Errorf("x86-64: register leakage detected in %d %d", line, col)
	}
	return err
}

func (t *X64Target) compileVarStmt(stmt *ast.VarStmt) error {
	if len(stmt.Values) == 0 {
		for i := range stmt.Vars {
			sym := t.info.VarSymbols[stmt][i]
			offset, ok := t.ctx.Get(sym)
			if !ok {
				return fmt.Errorf("x86-64: undeclared variable %s", sym.Name)
			}
			size := sizeOf(t.info.VarTypes[stmt][i])
			t.zero(slot(offset, size))
		}
		return nil
	}
	return t.eachValue(stmt.Values, func(i int, op Operand) error {
		sym := t.info.VarSymbols[stmt][i]
		offset, ok := t.ctx.Get(sym)
		if !ok {
			return fmt.Errorf("x86-64: undeclared variable %s", sym.Name)
		}
		size := sizeOf(t.info.VarTypes[stmt][i])
		t.store(op, slot(offset, size))
		return nil
	})
}

func (t *X64Target) compileReturnStmt(stmt *ast.ReturnStmt) error {
	if t.info.ErrorReturns[stmt] {
		op, err := t.compileExpr(stmt.Values[0])
		if err != nil {
			return err
		}
		t.returnError(op)
		return nil
	}

	ops := []Operand{}
	if err := t.eachValue(stmt.Values, func(i int, op Operand) error {
		ops = append(ops, op)
		return nil
	}); err != nil {
		return err
	}

	if t.fallible {
		ops = append(ops, Imm(0))
	}
	t.returnValue(ops, t.funcReturns, t.hidden)
	t.text.printlnt("jmp .return")

	return nil
}

func (t *X64Target) compileAssignStmt(stmt *ast.AssignStmt) error {
	if len(stmt.Targets) == 1 {
		memSlot, err := t.compileLValue(stmt.Targets[0])
		if err != nil {
			return err
		}

		op, err := t.compileExpr(stmt.Values[0])
		if err != nil {
			return err
		}

		if stmt.Op == "=" {
			t.store(op, memSlot)
		} else {
			targetType := t.info.Types[stmt.Targets[0]]
			valueType := t.info.Types[stmt.Values[0]]
			res, err := t.emitBinary(stmt.Op[:len(stmt.Op)-1], memSlot, op, targetType, valueType, targetType, nil)
			if err != nil {
				return err
			}
			t.store(res, memSlot)
		}
		t.freeOp(memSlot)

		return nil
	}

	err := t.eachValue(stmt.Values, func(i int, op Operand) error {
		t.pushValue(op)
		return nil
	})
	if err != nil {
		return err
	}

	for _, target := range slices.Backward(stmt.Targets) {
		memSlot, err := t.compileLValue(target)
		if err != nil {
			return err
		}
		size := sizeOf(t.info.Types[target])
		for w := (size+7)/8 - 1; w >= 0; w-- {
			reg := t.ctx.AllocFreeRegister()

			if reg == x64context.NoReg {
				return fmt.Errorf("x86-64: no available registers")
			}
			t.pop(reg)
			t.storeWord(memSlot.at(8*w, min(8, size-8*w)), Reg{reg, 8})
			t.ctx.FreeRegister(reg)
		}
		t.freeOp(memSlot)
	}

	return nil
}

func (t *X64Target) compileIncDecStmt(stmt *ast.IncDecStmt) error {
	memSlot, err := t.compileLValue(stmt.Target)
	if err != nil {
		return err
	}
	t.text.printft("%s %s\n", incDecOperators[stmt.Op], t.memText(memSlot))
	t.freeOp(memSlot)
	return nil
}

func (t *X64Target) compileIfStmt(stmt *ast.IfStmt) error {
	elseLabel := t.newLabel("else")
	endLabel := t.newLabel("end")

	if stmt.IsSwitch {
		ctx := x64context.LoopContext{
			BreakLabel:       endLabel,
			SupportsContinue: false,
		}
		t.ctx.PushLoop(ctx)
	}

	op, err := t.compileExpr(stmt.Condition)
	if err != nil {
		return err
	}
	size := sizeOf(t.info.Types[stmt.Condition])

	reg := t.materialize(op, size)
	opText := t.ctx.GetRegister(reg, size)
	if t.err != nil {
		return t.err
	}
	t.text.printft("test %s, %s\n", opText, opText)
	t.text.printft("jz %s\n", elseLabel)

	t.ctx.FreeRegister(reg)

	if err = t.compileStatement(stmt.Then); err != nil {
		return err
	}
	t.text.printft("jmp %s\n", endLabel)

	t.text.printf("%s:\n", elseLabel)
	if stmt.Else != nil {
		if err = t.compileStatement(stmt.Else); err != nil {
			return err
		}
	}

	t.text.printf("%s:\n", endLabel)
	return nil
}

func (t *X64Target) compileForStmt(stmt *ast.ForStmt) error {
	switch {
	case stmt.Range != nil:
		return t.compileForRange(stmt)

	case stmt.Start != nil:
		return t.compileForNumeric(stmt)

	default:
		return t.compileForCount(stmt)
	}
}

func (t *X64Target) loopCondition(cond ast.Expression, breakLabel string, until bool) error {
	tp := t.info.Types[cond]
	size := sizeOf(tp)

	if cond == nil {
		return nil
	}
	op, err := t.compileExpr(cond)
	if err != nil {
		return err
	}
	reg := t.materialize(op, size)
	regStr := t.ctx.GetRegister(reg, size)

	t.text.printft("test %s, %s\n", regStr, regStr)
	t.ctx.FreeRegister(reg)

	if until {
		t.text.printft("jnz ")
	} else {
		t.text.printft("jz ")
	}
	t.text.println(breakLabel)

	return nil
}

func (t *X64Target) compileLoopStmt(stmt *ast.LoopStmt) error {
	condLabel := t.newLabel("cond")

	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("loop_continue"),
		BreakLabel:       t.newLabel("loop_break"),
		SupportsContinue: true,
	}

	t.ctx.PushLoop(ctx)
	defer t.ctx.PopLoop()

	t.text.printf("%s:\n", condLabel)

	if err := t.loopCondition(stmt.Condition, ctx.BreakLabel, false); err != nil {
		return err
	}

	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)

	if err := t.loopCondition(stmt.UntilCondition, ctx.BreakLabel, true); err != nil {
		return err
	}

	t.text.printft("jmp %s\n", condLabel)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}

func (t *X64Target) compileBreakStmt(stmt *ast.BreakStmt) error {
	var loop *x64context.LoopContext

	if stmt.Label == "" {
		loop = t.ctx.CurrentLoop()
	} else {
		loop = t.ctx.FindLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("break: no matching loop")
	}

	t.text.printft("jmp %s\n", loop.BreakLabel)

	return nil
}

func (t *X64Target) compileContinueStmt(stmt *ast.ContinueStmt) error {
	var loop *x64context.LoopContext

	if stmt.Label == "" {
		loop = t.ctx.CurrentContinuable()
	} else {
		loop = t.ctx.FindLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("continue: no matching loop")
	}

	t.text.printft("jmp %s\n", loop.ContinueLabel)

	return nil
}
