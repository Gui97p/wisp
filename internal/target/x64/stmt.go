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
	case *ast.VarStmt:
		err = t.compileVarStmt(s)
	case *ast.ReturnStmt:
		err = t.compileReturnStmt(s)
	case *ast.AssignStmt:
		err = t.compileAssignStmt(s)
	case *ast.IfStmt:
		err = t.compileIfStmt(s)
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
			for w := 0; w < (size+7)/8; w++ {
				t.storeWord(slot(offset, size).at(8*w, min(8, size-8*w)), Imm(0))
			}
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
	ops := []Operand{}

	if err := t.eachValue(stmt.Values, func(i int, op Operand) error {
		ops = append(ops, op)
		return nil
	}); err != nil {
		return err
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

		size := sizeOf(t.info.Types[stmt.Targets[0]])

		switch stmt.Op {
		case "=":
			t.store(op, memSlot)
		case "+=", "-=":
			inst := "add"
			if stmt.Op == "-=" {
				inst = "sub"
			}
			reg := t.materialize(op, size)
			t.text.printft("%s %s, %s\n", inst, t.memText(memSlot), t.ctx.GetRegister(reg, size))
			t.ctx.FreeRegister(reg)
		case "*=":
			reg := t.materialize(op, size)
			r2 := t.ctx.AllocFreeRegister()
			if reg == x64context.NoReg {
				return fmt.Errorf("x86-64: no avaiable registers")
			}
			t.loadWord(r2, memSlot)

			t.text.printft("imul %s %s, %s\n", sizeLabels[size], t.ctx.GetRegister(r2, size), t.ctx.GetRegister(reg, size))
			t.storeWord(memSlot, Reg{r2, size})

			t.ctx.FreeRegister(reg)
			t.ctx.FreeRegister(r2)
		default:
			return fmt.Errorf("x86-64: unsupported operator %s", stmt.Op)
		}
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
				return fmt.Errorf("x86-64: no avaiable registers")
			}
			t.pop(reg)
			t.storeWord(memSlot.at(8*w, min(8, size-8*w)), Reg{reg, 8})
			t.ctx.FreeRegister(reg)
		}
	}

	return nil
}

func (t *X64Target) compileIfStmt(stmt *ast.IfStmt) error {
	op, err := t.compileExpr(stmt.Condition)
	if err != nil {
		return err
	}
	size := sizeOf(t.info.Types[stmt.Condition])

	elseLabel := t.newLabel("else")
	endLabel := t.newLabel("end")

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
