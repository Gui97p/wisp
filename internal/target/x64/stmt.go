package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileStatement(stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, stmt := range s.Statements {
			if err := t.compileStatement(stmt); err != nil {
				return err
			}
		}
		return nil
	case *ast.VarStmt:
		return t.compileVarStmt(s)
	case *ast.ReturnStmt:
		return t.compileReturnStmt(s)
	case *ast.AssignStmt:
		return t.compileAssignStmt(s)
	case *ast.IfStmt:
		return t.compileIfStmt(s)
	default:
		return fmt.Errorf("x86-64: unsupported statement %T", stmt)
	}
}

func (t *X64Target) compileVarStmt(stmt *ast.VarStmt) error {
	for i, v := range stmt.Vars {
		offset, ok := t.ctx.Get(v.Name)
		if !ok {
			return fmt.Errorf("x86-64: undeclared variable %s", v.Name)
		}
		size := t.sizeOf(t.info.VarTypes[stmt][i])
		op, err := t.compileExpr(stmt.Values[i])
		if err != nil {
			return err
		}
		t.text.printft("mov [rbp-%d], %s\n", offset, t.operandText(op, size))
		if r, ok := op.(RegOperand); ok {
			t.ctx.FreeRegister(r.Reg)
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

	opText := t.operandText(op, size)
	if opText != "rax" {
		t.text.printft("mov rax, %s\n", opText)
	}

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

	opText := t.operandText(op, size)

	switch stmt.Op {
	case "=":
		t.text.printft("mov [rbp-%d], %s\n", offset, opText)
	case "+=":
		t.text.printft("add [rbp-%d], %s\n", offset, opText)
	case "-=":
		t.text.printft("sub [rbp-%d], %s\n", offset, opText)
	case "*=":
		reg := t.ctx.AllocFreeRegister()
		regStr := t.ctx.GetRegister(reg, size)

		t.text.printft("mov %s, [rbp-%d]\n", regStr, offset)
		t.text.printft("imul %s, %s\n", regStr, opText)
		t.text.printft("mov [rbp-%d], %s\n", offset, regStr)

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

	opText := t.operandText(op, size)
	t.text.printft("test %s, %s\n", opText, opText)
	t.text.printft("jz %s\n", elseLabel)

	t.operandFree(op)

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
