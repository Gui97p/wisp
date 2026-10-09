package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) emitBinary(op string, left, right Operand, leftType, rightType, resultType analyser.Type, rightExpr ast.Expression) (Operand, error) {
	switch op {
	case "+", "-", "*", "/", "%":
		return t.compileArithmetic(op, left, right, resultType)
	case "==", "!=", ">", ">=", "<", "<=":
		return t.compileComparison(op, left, right, leftType, rightType, resultType)
	case "&", "|", "^", "<<", ">>":
		return t.compileBitwise(op, left, right, leftType, resultType)
	case "&&", "||":
		return t.compileLogical(op, left, rightExpr, leftType, rightType, resultType)
	default:
		return nil, fmt.Errorf("x86-64: unsupported operator %s", op)
	}
}

func (t *X64Target) compileScalarBinary(expr *ast.BinaryExpr) (Operand, error) {
	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}

	var right Operand
	if expr.Operator != "&&" && expr.Operator != "||" {
		free := len(x64context.RegisterOrder) - len(t.ctx.AllocatedOrderRegisters())
		if regsNeeded(expr.Right) > free {
			left = t.spill(left)
		}

		right, err = t.compileExpr(expr.Right)
		if err != nil {
			return nil, err
		}
	}

	return t.emitBinary(expr.Operator, left, right, t.info.Types[expr.Left], t.info.Types[expr.Right], t.info.Types[expr], expr.Right)
}

func (t *X64Target) compileErrorBinary(expr *ast.BinaryExpr, errExpr, member ast.Expression, enum analyser.NamedType) (Operand, error) {
	syn := &ast.StringLiteral{Value: enum.Module + "." + enum.Name}
	t.info.Types[syn] = analyser.PrimitiveType{Name: "string"}

	ops, err := t.compileCall(t.useErreq(), false, []ast.Expression{errExpr, syn, member}, []analyser.Type{analyser.PrimitiveType{Name: "bool"}})
	if err != nil {
		return nil, err
	}
	switch expr.Operator {
	case "==":
	case "!=":
		t.text.printft("xor %s, 1\n", t.opText(ops[0], 8))
	default:
		return nil, fmt.Errorf("x86-64: unsupported error comparison")
	}
	return ops[0], nil
}

func (t *X64Target) compileStringBinary(expr *ast.BinaryExpr) (Operand, error) {
	ops, err := t.compileCall(t.useStreq(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{analyser.PrimitiveType{Name: "bool"}})
	if err != nil {
		return nil, err
	}
	switch expr.Operator {
	case "==":
	case "!=":
		t.text.printft("xor %s, 1\n", t.opText(ops[0], 8))
	default:
		return nil, fmt.Errorf("x86-64: string ordering not supported")
	}
	return ops[0], nil
}

func (t *X64Target) compileArithmetic(op string, left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	signed := isSigned(tp)

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
	case "/", "%":
		savedRegisters := []x64context.Register{}
		for _, reg := range []x64context.Register{x64context.AX, x64context.DX} {
			if reg != leftReg && slices.Contains(t.ctx.AllocatedRegisters(), reg) {
				t.push(reg)
				t.ctx.FreeRegister(reg)
				savedRegisters = append(savedRegisters, reg)
			}
		}

		t.push(leftReg)
		t.ctx.FreeRegister(leftReg)
		t.pushValue(right)

		t.ctx.AllocRegister(x64context.AX)
		t.ctx.AllocRegister(x64context.DX)

		divReg := t.ctx.AllocFreeRegister()
		t.pop(divReg)
		t.pop(x64context.AX)

		width := 4
		if size == 8 {
			width = 8
		}
		divisor := t.ctx.GetRegister(divReg, width)
		acc := t.ctx.GetRegister(x64context.AX, width)

		if size < 4 {
			inst := "movzx"
			if signed {
				inst = "movsx"
			}
			t.text.printft("%s eax, %s\n", inst, t.ctx.GetRegister(x64context.AX, size))
			t.text.printft("%s %s, %s\n", inst, divisor, t.ctx.GetRegister(divReg, size))
		}

		notzeroLabel := t.newLabel("ok")
		doneLabel := t.newLabel("done")

		t.text.printft("test %s, %s\n", divisor, divisor)
		t.text.printft("jnz %s\n", notzeroLabel)
		t.panic("integer divide by zero")
		t.text.printf("%s:\n", notzeroLabel)

		inst := "div"
		if signed {
			inst = "idiv"
		}

		if signed && size >= 4 {
			normalLabel := t.newLabel("normal")
			t.text.printft("cmp %s, -1\n", divisor)
			t.text.printft("jne %s\n", normalLabel)
			t.text.printft("neg %s\n", acc)
			t.text.printlnt("xor edx, edx")
			t.text.printft("jmp %s\n", doneLabel)
			t.text.printf("%s:\n", normalLabel)
		}

		switch {
		case !signed:
			t.text.printlnt("xor edx, edx")
		case size == 8:
			t.text.printlnt("cqo")
		default:
			t.text.printlnt("cdq")
		}

		t.text.printft("%s %s\n", inst, divisor)
		t.text.printf("%s:\n", doneLabel)

		leftReg = t.ctx.AllocFreeRegister()
		source := x64context.AX
		if op == "%" {
			source = x64context.DX
		}
		t.text.printft("mov %s, %s\n", t.ctx.GetRegister(leftReg, size), t.ctx.GetRegister(source, size))

		t.ctx.FreeRegister(divReg)
		t.ctx.FreeRegister(x64context.AX)
		t.ctx.FreeRegister(x64context.DX)

		for _, reg := range slices.Backward(savedRegisters) {
			t.pop(reg)
			t.ctx.AllocRegister(reg)
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

func (t *X64Target) compileLogical(op string, left Operand, rightExpr ast.Expression, leftType, rightType, resultType analyser.Type) (Operand, error) {
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

	right, err := t.compileExpr(rightExpr)
	if err != nil {
		return nil, err
	}

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
