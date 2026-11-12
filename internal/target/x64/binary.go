package x64

import (
	"fmt"
	"math/bits"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) emitBinary(op string, left, right Operand, leftType, rightType, resultType analyser.Type, rightExpr ast.Expression) (Operand, error) {
	if _, isAddr := right.(Addr); isAddr {
		opSize := sizeOf(rightType)
		r := t.materialize(right, opSize)
		right = Reg{Reg: r, Size: opSize}
	}

	if (op == "+" || op == "-") && (isPointer(leftType) || isPointer(rightType)) {
		return t.compilePointerArithmetic(op, left, right, leftType, rightType, resultType)
	}

	switch op {
	case "+", "-", "*", "/", "%":
		if isFloat(resultType) {
			return t.compileFloatArithmetic(op, left, right, resultType)
		}
		return t.compileArithmetic(op, left, right, resultType)
	case "==", "!=", ">", ">=", "<", "<=":
		if isFloat(leftType) || isFloat(rightType) {
			return t.compileFloatComparison(op, left, right, leftType, rightType, resultType)
		}
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

	ops, err := t.compileCall(t.useErreq(), false, []ast.Expression{errExpr, syn, member}, []analyser.Type{analyser.PrimitiveType{Name: "bool"}}, false)
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
	switch expr.Operator {
	case "==", "!=":
		ops, err := t.compileCall(t.useStreq(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{analyser.PrimitiveType{Name: "bool"}}, false)
		if err != nil {
			return nil, err
		}
		switch expr.Operator {
		case "==":
		case "!=":
			t.text.printft("xor %s, 1\n", t.opText(ops[0], 8))
		}
		return ops[0], nil
	case "<", "<=", ">", ">=":
		intType := analyser.PrimitiveType{Name: "int"}
		ops, err := t.compileCall(t.useStrcmp(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{intType}, false)
		if err != nil {
			return nil, err
		}

		return t.compileComparison(expr.Operator, ops[0], Imm(0), intType, intType, t.info.Types[expr])
	default:
		return nil, fmt.Errorf("x86-64: operator %s not supported", expr.Operator)
	}
}

func (t *X64Target) compilePointerArithmetic(op string, left, right Operand, leftType, rightType, resultType analyser.Type) (Operand, error) {
	leftPtr := isPointer(leftType)
	rightPtr := isPointer(rightType)

	switch {
	case leftPtr && rightPtr:
		diff, err := t.compileArithmetic("-", left, right, resultType)
		if err != nil {
			return nil, err
		}
		size := elemSize(leftType)

		switch {
		case size == 1:
			return diff, nil
		case size&(size-1) == 0:
			reg := t.materialize(diff, 8)
			t.text.printft("sar %s, %d\n", t.ctx.GetRegister(reg, 8), bits.TrailingZeros(uint(size)))
			return Reg{reg, 8}, nil
		default:
			return t.compileArithmetic("/", diff, Imm(size), resultType)
		}
	case leftPtr:
		off := t.offset(right, rightType, elemSize(leftType))
		return t.compileArithmetic(op, left, off, resultType)
	default:
		off := t.offset(left, leftType, elemSize(rightType))
		return t.compileArithmetic("+", right, off, resultType)
	}
}

func (t *X64Target) compileFloatArithmetic(op string, left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)

	leftReg := t.materializeXmm(left)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)
	reg := t.ctx.GetRegister(leftReg, size)

	sizeInst := "sd"
	if size == 4 {
		sizeInst = "ss"
	}
	switch op {
	case "+", "-", "*", "/":
		t.text.printft("%s%s %s, %s\n", mathOperators[op], sizeInst, reg, opText)
	default:
		return nil, fmt.Errorf("x86-64: binary operator %s not supported", op)
	}
	return Reg{Reg: leftReg, Size: size}, nil
}

func (t *X64Target) compileArithmetic(op string, left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	signed := isSigned(tp)

	if op == "*" {
		if _, ok := left.(Imm); ok {
			if _, ok := right.(Mem); ok {
				left, right = right, left
			}
		}

		if m, ok := left.(Mem); ok {
			if i, ok := right.(Imm); ok && size >= 4 && (i >= -2147483648 && i <= 2147483647) {
				dst := t.ctx.AllocFreeRegister()
				if dst == x64context.NoReg {
					return nil, fmt.Errorf("x86-64: no avaiable registers")
				}
				t.text.printft("imul %s, %s, %d\n", t.ctx.GetRegister(dst, size), t.memText(m), i)
				t.freeOp(left)
				return Reg{dst, size}, nil
			}
		}
	}

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
		if imm, ok := right.(Imm); ok && size >= 4 {
			if res, done := t.divideByConstant(op, leftReg, int64(imm), size, signed); done {
				return res, nil
			}
		}
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

		if imm, ok := right.(Imm); (!ok || (imm != 0 && imm != -1)) && t.checks {
			t.text.printft("test %s, %s\n", divisor, divisor)
			t.text.printft("jnz %s\n", notzeroLabel)
			t.panic("integer divide by zero")
			t.text.printf("%s:\n", notzeroLabel)
		}

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

func (t *X64Target) divideByConstant(op string, reg x64context.Register, imm int64, size int, signed bool) (Operand, bool) {
	if imm > 1 && imm&(imm-1) == 0 && imm <= 1<<30 {
		regStr := t.ctx.GetRegister(reg, size)
		k := bits.TrailingZeros(uint(imm))
		tmp := t.ctx.AllocFreeRegister()
		if tmp == x64context.NoReg {
			return nil, false
		}
		tmpStr := t.ctx.GetRegister(tmp, size)
		w := size * 8
		switch op {
		case "/":
			switch {
			case imm == 1:
			case signed:
				t.text.printft("mov %s, %s\n", tmpStr, regStr)
				t.text.printft("sar %s, %d\n", tmpStr, w-1)
				t.text.printft("shr %s, %d\n", tmpStr, w-k)
				t.text.printft("add %s, %s\n", regStr, tmpStr)
				t.text.printft("sar %s, %d\n", regStr, k)
			default:
				t.text.printft("shr %s, %d\n", regStr, k)
			}
		case "%":
			switch {
			case imm == 1:
				regStr32 := t.ctx.GetRegister(reg, 4)
				t.text.printft("xor %s, %s\n", regStr32, regStr32)
			case signed:
				t.text.printft("mov %s, %s\n", tmpStr, regStr)
				t.text.printft("sar %s, %d\n", tmpStr, w-1)
				t.text.printft("shr %s, %d\n", tmpStr, w-k)
				t.text.printft("add %s, %s\n", tmpStr, regStr)
				t.text.printft("and %s, %d\n", tmpStr, -imm)
				t.text.printft("sub %s, %s\n", regStr, tmpStr)
			default:
				t.text.printft("and %s, %d\n", regStr, imm-1)
			}
		}
		t.ctx.FreeRegister(tmp)
		return Reg{reg, size}, true
	}
	return nil, false
}

func (t *X64Target) compileFloatComparison(op string, left, right Operand, leftType, rightType, resultType analyser.Type) (Operand, error) {
	var tp analyser.Type
	switch leftType.(type) {
	case analyser.UntypedFloatType, analyser.UntypedIntType:
		tp = rightType
	default:
		tp = leftType
	}
	var opSize int
	var leftReg x64context.Register
	opSize = sizeOf(tp)
	switch op {
	case "<", "<=":
		right, left = left, right
	}
	leftReg = t.readXmm(left)
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

	sdop := "sd"
	if opSize == 4 {
		sdop = "ss"
	}

	leftCmp := t.ctx.GetRegister(leftReg, opSize)
	ccop := ucomiOperators[op]

	t.text.printft("xor %s, %s\n", resultRegStr, resultRegStr)
	t.text.printft("ucomi%s %s, %s\n", sdop, leftCmp, opText)
	switch op {
	case "==", "!=":
		tmp := t.ctx.AllocFreeRegister()
		tmpStr := t.ctx.GetRegister(tmp, 1)
		difop := "setp"
		cmpop := "or"
		if ccop == "sete" {
			difop = "setnp"
			cmpop = "and"
		}
		t.text.printft("%s %s\n%s %s\n", ccop, resultRegStr, difop, tmpStr)
		t.text.printft("%s %s, %s\n", cmpop, resultRegStr, tmpStr)
		t.ctx.FreeRegister(tmp)
	default:
		t.text.printft("%s %s\n", ccop, resultRegStr)
	}

	t.ctx.FreeRegister(leftReg)
	t.freeOp(right)

	return Reg{Reg: resultReg, Size: size}, nil
}

func (t *X64Target) branch(cond ast.Expression, label string, whenTrue bool) error {
	if c, folded := t.info.Folded(cond); folded && c.Kind == analyser.ConstBool {
		if c.Bool == whenTrue {
			t.text.printft("jmp %s\n", label)
		}
		return nil
	}

	switch c := cond.(type) {
	case *ast.UnaryExpr:
		if c.Operator == "!" {
			return t.branch(c.Value, label, !whenTrue)
		}
	case *ast.BinaryExpr:
		switch c.Operator {
		case "&&", "||":
			if whenTrue == (c.Operator == "||") {
				if err := t.branch(c.Left, label, whenTrue); err != nil {
					return err
				}
				return t.branch(c.Right, label, whenTrue)
			} else {
				skip := t.newLabel("skip")
				if err := t.branch(c.Left, skip, !whenTrue); err != nil {
					return err
				}
				err := t.branch(c.Right, label, whenTrue)
				t.text.printf("%s:\n", skip)
				return err
			}
		}
		if t.fusableComparison(c) {
			return t.branchCompare(c, label, whenTrue)
		}
	}

	return t.branchValue(cond, label, whenTrue)
}

func (t *X64Target) fusableComparison(c *ast.BinaryExpr) bool {
	switch c.Operator {
	case "==", "!=", "<", "<=", ">", ">=":
	default:
		return false
	}
	if _, isTypeCase := t.info.TypeCases[c]; isTypeCase {
		return false
	}
	left, right := t.info.Types[c.Left], t.info.Types[c.Right]
	if isString(left) || isString(right) || isFloat(left) || isFloat(right) {
		return false
	}
	if _, _, _, isError := t.errorMember(c); isError {
		return false
	}
	return true
}

func (t *X64Target) branchCompare(c *ast.BinaryExpr, label string, whenTrue bool) error {
	left, err := t.compileExpr(c.Left)
	if err != nil {
		return err
	}
	free := len(x64context.RegisterOrder) - len(t.ctx.AllocatedOrderRegisters())
	if regsNeeded(c.Right) > free {
		left = t.spill(left)
	}
	right, err := t.compileExpr(c.Right)
	if err != nil {
		return err
	}
	if _, isAddr := right.(Addr); isAddr {
		size := sizeOf(t.info.Types[c.Right])
		right = Reg{Reg: t.materialize(right, size), Size: size}
	}
	cc, err := t.emitCompare(c.Operator, left, right, t.info.Types[c.Left], t.info.Types[c.Right])
	if err != nil {
		return err
	}
	if !whenTrue {
		cc = inverseCondition[cc]
	}
	t.text.printft("j%s %s\n", cc, label)
	return nil
}

func (t *X64Target) branchValue(cond ast.Expression, label string, whenTrue bool) error {
	op, err := t.compileExpr(cond)
	if err != nil {
		return err
	}
	size := sizeOf(t.info.Types[cond])
	reg := t.readReg(op, size)
	if t.err != nil {
		return t.err
	}
	regStr := t.ctx.GetRegister(reg, size)
	t.text.printft("test %s, %s\n", regStr, regStr)
	t.ctx.FreeRegister(reg)
	jump := "jz"
	if whenTrue {
		jump = "jnz"
	}
	t.text.printft("%s %s\n", jump, label)
	return nil
}

func (t *X64Target) emitCompare(op string, left, right Operand, leftType, rightType analyser.Type) (string, error) {
	var tp analyser.Type
	switch leftType.(type) {
	case analyser.UntypedFloatType, analyser.UntypedIntType:
		tp = rightType
	default:
		tp = leftType
	}

	opSize := sizeOf(tp)
	leftReg := t.readReg(left, opSize)
	opText := t.opText(right, opSize)
	if t.err != nil {
		return "", t.err
	}

	ccop := usetccOperators[op]
	if isSigned(tp) {
		ccop = setccOperators[op]
	}

	t.text.printft("cmp %s, %s\n", t.ctx.GetRegister(leftReg, opSize), opText)

	t.ctx.FreeRegister(leftReg)
	t.freeOp(right)

	return ccop, nil
}

func (t *X64Target) compileComparison(op string, left, right Operand, leftType, rightType, resultType analyser.Type) (Operand, error) {
	size := sizeOf(resultType)
	resultReg := t.ctx.AllocFreeRegister()
	if resultReg == x64context.NoReg {
		return nil, fmt.Errorf("x86-64: no available registers")
	}
	resultRegStr := t.ctx.GetRegister(resultReg, size)
	t.text.printft("xor %s, %s\n", resultRegStr, resultRegStr)
	ccop, err := t.emitCompare(op, left, right, leftType, rightType)
	if err != nil {
		return nil, err
	}
	t.text.printft("set%s %s\n", ccop, resultRegStr)
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
	if op != "<<" && op != ">>" {
		t.freeOp(right)
	}
	reg := t.ctx.GetRegister(leftReg, size)

	switch op {
	case "&", "|", "^":
		t.text.printft("%s %s, %s\n", mathOperators[op], reg, opText)
	case "<<", ">>":
		signed := isSigned(leftType)
		allocated := false
		claimed := false
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
			inst := "shl"
			if op == ">>" {
				inst = "shr"
				if signed {
					inst = "sar"
				}
			}
			t.text.printft("%s %s, %d\n", inst, reg, o)
			return Reg{leftReg, size}, nil
		default:
			if leftReg == x64context.CX {
				leftReg = t.ctx.AllocFreeRegister()
				t.text.printft("mov %s, %s\n", t.ctx.GetRegister(leftReg, size), reg)
				reg = t.ctx.GetRegister(leftReg, size)
				t.ctx.FreeRegister(x64context.CX)
			}

			allocated = !t.ctx.AllocRegister(x64context.CX)
			claimed = !allocated
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

		t.text.printf("%s:\n", doneLabel)

		if allocated {
			t.pop(x64context.CX)
		} else if claimed {
			t.ctx.FreeRegister(x64context.CX)
		}

	default:
		return nil, fmt.Errorf("x86-64: binary operator %s not supported", op)
	}
	return Reg{Reg: leftReg, Size: size}, nil
}
