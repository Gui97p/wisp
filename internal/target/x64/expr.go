package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

type Operand interface {
	op()
}

type Imm int64

func (Imm) op() {}

type RegOperand struct {
	Reg  x64context.Reg
	Size int
}

func (RegOperand) op() {}

func (t *X64Target) ensureRegister(op Operand, size int) x64context.Reg {
	if r, ok := op.(RegOperand); ok {
		return r.Reg
	}

	reg := t.ctx.AllocFreeRegister()
	t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, size), op.(Imm))
	return reg
}

func (t *X64Target) operandText(op Operand, size int) string {
	switch o := op.(type) {
	case Imm:
		return fmt.Sprint(o)
	case RegOperand:
		return t.ctx.GetRegister(o.Reg, size)
	default:
		return ""
	}
}

func (t *X64Target) compileExpression(expr ast.Expression) (Operand, error) {
	size := t.sizeOf(t.info.Types[expr])

	switch e := expr.(type) {
	case *ast.IntLiteral:
		return Imm(e.Value), nil
	case *ast.BinaryExpr:
		left, err := t.compileExpression(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := t.compileExpression(e.Right)
		if err != nil {
			return nil, err
		}

		leftReg := t.ensureRegister(left, size)
		reg := t.ctx.GetRegister(leftReg, size)
		opText := t.operandText(right, size)
		if r, ok := right.(RegOperand); ok {
			t.ctx.FreeRegister(r.Reg)
		}
		switch e.Operator {
		case "+":
			t.text.printft("add %s, %s\n", reg, opText)
		case "-":
			t.text.printft("sub %s, %s\n", reg, opText)
		case "*":
			t.text.printft("imul %s, %s\n", reg, opText)
		}
		return RegOperand{Reg: leftReg, Size: size}, nil
	default:
		return nil, fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}
