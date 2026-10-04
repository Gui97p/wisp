package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileArithmetic(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size := sizeOf(t.info.Types[expr.Left])
	leftReg := t.materialize(left, size)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)
	reg := t.ctx.GetRegister(leftReg, size)

	switch expr.Operator {
	case "+":
		t.text.printft("add %s, %s\n", reg, opText)
	case "-":
		t.text.printft("sub %s, %s\n", reg, opText)
	case "*":
		if size == 1 {
			right := t.materialize(right, 4)
			t.text.printft("imul %s, %s\n", t.ctx.GetRegister(leftReg, 4), t.ctx.GetRegister(right, 4))
			t.ctx.FreeRegister(right)
		} else {
			t.text.printft("imul %s, %s\n", reg, opText)
		}
	default:
		return nil, fmt.Errorf("x86-64: binary operator %s not supported", expr.Operator)
	}
	return Reg{Reg: leftReg, Size: size}, nil
}

func (t *X64Target) compileComparison(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	tp := t.info.Types[expr.Left]
	opSize := sizeOf(tp)
	leftReg := t.materialize(left, opSize)
	opText := t.opText(right, opSize)
	if t.err != nil {
		return nil, t.err
	}

	size := sizeOf(t.info.Types[expr])
	resultReg := t.ctx.AllocFreeRegister()
	resultRegStr := t.ctx.GetRegister(resultReg, size)

	ccop := usetccOperators[expr.Operator]
	if isSigned(tp) {
		ccop = setccOperators[expr.Operator]
	}

	t.text.printft("xor %s, %s\n", resultRegStr, resultRegStr)
	t.text.printft("cmp %s, %s\n", t.ctx.GetRegister(leftReg, opSize), opText)
	t.text.printft("%s %s\n", ccop, resultRegStr)

	t.ctx.FreeRegister(leftReg)
	t.freeOp(right)

	return Reg{Reg: resultReg, Size: size}, nil
}

func (t *X64Target) compileLogical(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}

	condLabel := t.newLabel("logcond")
	endLabel := t.newLabel("logend")

	inst, n := "jz", 1
	if expr.Operator == "||" {
		inst = "jnz"
		n = 0
	}

	size := sizeOf(t.info.Types[expr.Left])
	leftReg := t.materialize(left, size)
	if t.err != nil {
		return nil, t.err
	}
	leftRegStr := t.ctx.GetRegister(leftReg, size)
	t.text.printft("test %s, %s\n", leftRegStr, leftRegStr)
	t.text.printft("%s %s\n", inst, condLabel)

	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size = sizeOf(t.info.Types[expr.Right])
	rightReg := t.materialize(right, size)
	if t.err != nil {
		return nil, t.err
	}
	rightRegStr := t.ctx.GetRegister(rightReg, size)
	t.text.printft("test %s, %s\n", rightRegStr, rightRegStr)

	t.text.printft("%s %s\n", inst, condLabel)
	t.ctx.FreeRegister(rightReg)

	t.text.printft("mov %s, %d\n", leftRegStr, n)
	t.text.printft("jmp %s\n", endLabel)

	t.text.printf("%s:\n", condLabel)
	t.text.printft("mov %s, %d\n", leftRegStr, (n+1)%2)

	t.text.printf("%s:\n", endLabel)

	return Reg{Reg: leftReg, Size: sizeOf(t.info.Types[expr])}, nil
}

func (t *X64Target) compileBitwise(_ *ast.BinaryExpr) (Operand, error) {
	return nil, fmt.Errorf("x86-64: bitwise operations are not supported")
}
