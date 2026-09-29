package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileExpression(expr ast.Expression) (Operand, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return Imm(e.Value), nil
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(e.Value)
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined literal")
		}

		freeReg := t.ctx.AllocFreeRegister()
		size := t.sizeOf(t.info.Types[e])
		t.text.printft("mov %s, [rbp-%d]\n", t.ctx.GetRegister(freeReg, size), offset)

		return RegOperand{Reg: freeReg, Size: size}, nil

	case *ast.BinaryExpr:
		return t.compileBinaryExpression(e)
	default:
		return nil, fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}

func (t *X64Target) compileBinaryExpression(expr *ast.BinaryExpr) (Operand, error) {
	size := t.sizeOf(t.info.Types[expr])

	left, err := t.compileExpression(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpression(expr.Right)
	if err != nil {
		return nil, err
	}

	leftReg := t.ensureRegister(left, size)
	reg := t.ctx.GetRegister(leftReg, size)
	opText := t.operandText(right, size)
	if r, ok := right.(RegOperand); ok {
		t.ctx.FreeRegister(r.Reg)
	}
	switch expr.Operator {
	case "+":
		t.text.printft("add %s, %s\n", reg, opText)
	case "-":
		t.text.printft("sub %s, %s\n", reg, opText)
	case "*":
		t.text.printft("imul %s, %s\n", reg, opText)
	}
	return RegOperand{Reg: leftReg, Size: size}, nil
}
