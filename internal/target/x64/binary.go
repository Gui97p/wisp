package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) emitBinary(op string, left, right Operand, leftType, rightType, resultType analyser.Type) (Operand, error) {
	switch op {
	case "+", "-", "*", "/", "%":
		return t.compileArithmetic(op, left, right, resultType)
	case "==", "!=", ">", ">=", "<", "<=":
		return t.compileComparison(op, left, right, leftType, rightType, resultType)
	case "&", "|", "^", "<<", ">>":
		return t.compileBitwise(op, left, right, leftType, resultType)
	case "&&", "||":
		return t.compileLogical(op, left, right, leftType, rightType, resultType)
	default:
		return nil, fmt.Errorf("x86-64: unsupported operator %s", op)
	}
}

func (t *X64Target) compileArithmetic(op string, left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	leftReg := t.materialize(left, size)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)
	reg := t.ctx.GetRegister(leftReg, size)

	switch op {
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
		return nil, fmt.Errorf("x86-64: binary operator %s not supported", op)
	}
	return Reg{Reg: leftReg, Size: size}, nil
}

func (t *X64Target) compileComparison(op string, left, right Operand, leftType, rightType, resultType analyser.Type) (Operand, error) {
	var tp analyser.Type
	switch leftType.(type) {
	case analyser.UntypedFloatType, analyser.UntypedIntType:
		tp = rightType
	default:
		tp = leftType
	}
	opSize := sizeOf(tp)
	leftReg := t.materialize(left, opSize)
	opText := t.opText(right, opSize)
	if t.err != nil {
		return nil, t.err
	}

	size := sizeOf(resultType)
	resultReg := t.ctx.AllocFreeRegister()
	if resultReg == x64context.NoReg {
		return nil, fmt.Errorf("x86-64: no available registers")
	}
	resultRegStr := t.ctx.GetRegister(resultReg, size)

	ccop := usetccOperators[op]
	if isSigned(tp) {
		ccop = setccOperators[op]
	}

	t.text.printft("xor %s, %s\n", resultRegStr, resultRegStr)
	t.text.printft("cmp %s, %s\n", t.ctx.GetRegister(leftReg, opSize), opText)
	t.text.printft("%s %s\n", ccop, resultRegStr)

	t.ctx.FreeRegister(leftReg)
	t.freeOp(right)

	return Reg{Reg: resultReg, Size: size}, nil
}

func (t *X64Target) compileLogical(op string, left, right Operand, leftType, rightType, resultType analyser.Type) (Operand, error) {
	condLabel := t.newLabel("logcond")
	endLabel := t.newLabel("logend")

	inst, n := "jz", 1
	if op == "||" {
		inst = "jnz"
		n = 0
	}

	size := sizeOf(leftType)
	leftReg := t.materialize(left, size)
	if t.err != nil {
		return nil, t.err
	}
	leftRegStr := t.ctx.GetRegister(leftReg, size)
	t.text.printft("test %s, %s\n", leftRegStr, leftRegStr)
	t.text.printft("%s %s\n", inst, condLabel)

	size = sizeOf(rightType)
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

	return Reg{Reg: leftReg, Size: sizeOf(resultType)}, nil
}

func (t *X64Target) compileBitwise(op string, left, right Operand, leftType, resultType analyser.Type) (Operand, error) {
	size := sizeOf(resultType)
	leftReg := t.materialize(left, size)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)
	reg := t.ctx.GetRegister(leftReg, size)

	switch op {
	case "&", "|", "^":
		t.text.printft("%s %s, %s\n", mathOperators[op], reg, opText)
	case "<<", ">>":
		signed := isSigned(leftType)
		allocated := false
		length := size * 8
		zeroLabel := t.newLabel("zero")
		doneLabel := t.newLabel("done")

		switch o := right.(type) {
		case Imm:
			if int(o) >= length {
				if op == ">>" && signed {
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

		if op == "<<" {
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
		if op == ">>" && signed {
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
		return nil, fmt.Errorf("x86-64: binary operator %s not supported", op)
	}
	return Reg{Reg: leftReg, Size: size}, nil
}
