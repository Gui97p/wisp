package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) typedOperand(left, right ast.Expression) analyser.Type {
	lt := t.info.Types[left]
	rt := t.info.Types[right]

	switch lt.(type) {
	case analyser.UntypedFloatType, analyser.UntypedIntType:
		return rt
	default:
		return lt
	}
}

func (t *X64Target) compileArithmetic(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size := sizeOf(t.info.Types[expr])
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

	tp := t.typedOperand(expr.Left, expr.Right)
	opSize := sizeOf(tp)
	leftReg := t.materialize(left, opSize)
	opText := t.opText(right, opSize)
	if t.err != nil {
		return nil, t.err
	}

	size := sizeOf(t.info.Types[expr])
	resultReg := t.ctx.AllocFreeRegister()
	if resultReg == x64context.NoReg {
		return nil, fmt.Errorf("x86-64: no available registers")
	}
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

func (t *X64Target) compileBitwise(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := t.compileExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	size := sizeOf(t.info.Types[expr])
	leftReg := t.materialize(left, size)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)
	reg := t.ctx.GetRegister(leftReg, size)

	switch expr.Operator {
	case "&", "|", "^":
		t.text.printft("%s %s, %s\n", mathOperators[expr.Operator], reg, opText)
	case "<<", ">>":
		signed := isSigned(t.info.Types[expr.Left])
		allocated := false
		length := size * 8
		zeroLabel := t.newLabel("zero")
		doneLabel := t.newLabel("done")

		switch o := right.(type) {
		case Imm:
			if int(o) >= length {
				if expr.Operator == ">>" && signed {
					t.text.printft("sar %s, %d\n", reg, length-1)
				} else {
					t.text.printft("xor %s, %s\n", reg, reg)
				}
				return Reg{leftReg, size}, nil
			}
		default:
			if leftReg == x64context.CX {
				leftReg = t.ctx.AllocFreeRegister()
				t.text.printft("mov %s, %s\n", t.ctx.GetRegister(leftReg, size), reg)
				reg = t.ctx.GetRegister(leftReg, size)
				t.ctx.FreeRegister(x64context.CX)
			}

			allocated = !t.ctx.AllocRegister(x64context.CX)
			if allocated {
				t.push(x64context.CX)
			}
			cnt := t.materialize(right, size)
			t.text.printft("cmp %s, %d\n", t.ctx.GetRegister(cnt, size), length)
			t.text.printft("jae %s\n", zeroLabel)
			t.text.printft("mov cl, %s\n", t.ctx.GetRegister(cnt, 1))
			t.ctx.FreeRegister(cnt)
			opText = "cl"
		}

		if expr.Operator == "<<" {
			t.text.printft("shl %s, %s\n", reg, opText)
		} else {
			inst := "shr"
			if signed {
				inst = "sar"
			}
			t.text.printft("%s %s, %s\n", inst, reg, opText)
		}

		t.text.printft("jmp %s\n", doneLabel)
		t.text.printf("%s:\n", zeroLabel)
		if expr.Operator == ">>" && signed {
			t.text.printft("sar %s, %d\n", reg, length-1)
		} else {
			t.text.printft("xor %s, %s\n", reg, reg)
		}

		t.text.printf("%s:", doneLabel)

		if allocated {
			t.pop(x64context.CX)
		} else {
			t.ctx.FreeRegister(x64context.CX)
		}

	default:
		return nil, fmt.Errorf("x86-64: binary operator %s not supported", expr.Operator)
	}
	return Reg{Reg: leftReg, Size: size}, nil
}
