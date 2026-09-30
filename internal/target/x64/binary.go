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

	size := t.sizeOf(t.info.Types[expr.Left])
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

func (t *X64Target) compileComparison(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size := t.sizeOf(t.info.Types[expr.Left])
	leftReg := t.ensureRegister(left, size)
	leftRegStr := t.ctx.GetRegister(leftReg, size)
	opText := t.operandText(right, size)

	size = t.sizeOf(t.info.Types[expr])
	resultReg := t.ctx.AllocFreeRegister()
	resultRegStr := t.ctx.GetRegister(resultReg, size)

	t.text.printft("xor %s, %s\n", resultRegStr, resultRegStr)
	t.text.printft("cmp %s, %s\n", leftRegStr, opText)
	t.text.printft("%s %s\n", setccOperators[expr.Operator], resultRegStr)

	t.ctx.FreeRegister(leftReg)
	t.operandFree(right)

	return RegOperand{Reg: resultReg, Size: size}, nil
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

	size := t.sizeOf(t.info.Types[expr.Left])
	leftReg := t.ensureRegister(left, size)
	leftRegStr := t.ctx.GetRegister(leftReg, size)
	t.text.printft("test %s, %s\n", leftRegStr, leftRegStr)
	t.text.printft("%s %s\n", inst, condLabel)

	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size = t.sizeOf(t.info.Types[expr.Right])
	rightReg := t.ensureRegister(right, size)
	rightRegStr := t.ctx.GetRegister(rightReg, size)
	t.text.printft("test %s, %s\n", rightRegStr, rightRegStr)

	t.text.printft("%s %s\n", inst, condLabel)
	t.ctx.FreeRegister(rightReg)

	t.text.printft("mov %s, %d\n", leftRegStr, n)
	t.text.printft("jmp %s\n", endLabel)

	t.text.printf("%s:\n", condLabel)
	t.text.printft("mov %s, %d\n", leftRegStr, (n+1)%2)

	t.text.printf("%s:\n", endLabel)

	return RegOperand{Reg: leftReg, Size: t.sizeOf(t.info.Types[expr])}, nil
}

func (t *X64Target) compileBitwise(_ *ast.BinaryExpr) (Operand, error) {
	return nil, fmt.Errorf("x86-64: bitwise operations are not supported")
}
