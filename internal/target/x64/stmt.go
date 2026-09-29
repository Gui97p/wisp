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
			t.text.newLine()
		}
		return nil
	case *ast.VarStmt:
		return t.compileVarStmt(s)
	case *ast.ReturnStmt:
		return t.compileReturnStmt(s)
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
		op, err := t.compileExpression(stmt.Values[i])
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
	op, err := t.compileExpression(stmt.Values[0])
	if err != nil {
		return err
	}

	t.text.printft("mov rax, %s\n", t.operandText(op, size))

	return nil
}
