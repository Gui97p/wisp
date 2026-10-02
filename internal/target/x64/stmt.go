package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileStatement(stmt ast.Statement) error {
	var err error

	switch s := stmt.(type) {
	case *ast.NativeStmt:
		err = t.compileNativeStmt(s)
	case *ast.ExpressionStmt:
		var op Operand
		op, err = t.compileExpr(s.Expr)
		t.freeOperand(op)
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
	for i, v := range stmt.Vars {
		offset, ok := t.ctx.Get(t.info.VarSymbols[stmt][i])
		if !ok {
			return fmt.Errorf("x86-64: undeclared variable %s", v.Name)
		}
		size := t.sizeOf(t.info.VarTypes[stmt][i])
		if i < len(stmt.Values) && stmt.Values[i] != nil {
			op, err := t.compileExpr(stmt.Values[i])
			if err != nil {
				return err
			}
			if err := t.storeOperand(op, offset, size); err != nil {
				return err
			}
			t.freeOperand(op)
		} else {
			for off := 0; off < size; off += 8 {
				t.text.printft("mov %s, 0\n", t.mem(offset-off, size-off))
			}
		}
	}

	return nil
}

func (t *X64Target) compileReturnStmt(stmt *ast.ReturnStmt) error {
	if len(stmt.Values) != 1 {
		return fmt.Errorf("x86-64: only single-value return supported for now")
	}

	size := t.sizeOf(t.info.Types[stmt.Values[0]])
	op, err := t.compileExpr(stmt.Values[0])
	if err != nil {
		return err
	}

	opText, err := t.operandText(op, size)
	if err != nil {
		return err
	}
	if opText != "rax" {
		t.text.printft("mov rax, %s\n", opText)
	}

	t.freeOperand(op)

	t.text.printlnt("jmp .return")

	return nil
}

func (t *X64Target) compileAssignStmt(stmt *ast.AssignStmt) error {
	offset, err := t.compileLValue(stmt.Target)
	if err != nil {
		return err
	}
	size := t.sizeOf(t.info.Types[stmt.Target])

	op, err := t.compileExpr(stmt.Value)
	if err != nil {
		return err
	}

	var opText string
	if stmt.Op != "=" {
		opText, err = t.operandText(op, size)
		if err != nil {
			return err
		}
	}

	switch stmt.Op {
	case "=":
		if err := t.storeOperand(op, offset, size); err != nil {
			return err
		}
	case "+=":
		t.text.printft("add %s, %s\n", t.mem(offset, size), opText)
	case "-=":
		t.text.printft("sub %s, %s\n", t.mem(offset, size), opText)
	case "*=":
		reg := t.ctx.AllocFreeRegister()
		regStr := t.ctx.GetRegister(reg, size)

		t.text.printft("mov %s %s, [rbp-%d]\n", sizeLabels[size], regStr, offset)
		t.text.printft("imul %s %s, %s\n", sizeLabels[size], regStr, opText)
		t.text.printft("mov %s, %s\n", t.mem(offset, size), regStr)

		t.ctx.FreeRegister(reg)
	default:
		return fmt.Errorf("x86-64: unsupported operator %s", stmt.Op)
	}

	if r, ok := op.(RegOperand); ok {
		t.ctx.FreeRegister(r.Reg)
	}

	return nil
}

func (t *X64Target) compileIfStmt(stmt *ast.IfStmt) error {
	op, err := t.compileExpr(stmt.Condition)
	if err != nil {
		return err
	}
	size := t.sizeOf(t.info.Types[stmt.Condition])

	elseLabel := t.newLabel("else")
	endLabel := t.newLabel("end")

	opText, err := t.operandText(op, size)
	if err != nil {
		return err
	}
	t.text.printft("test %s, %s\n", opText, opText)
	t.text.printft("jz %s\n", elseLabel)

	t.freeOperand(op)

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
