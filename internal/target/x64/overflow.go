package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

var signedMax = map[int]string{1: "0x7f", 2: "0x7fff", 4: "0x7fffffff", 8: "0x7fffffffffffffff"}
var signedMin = map[int]string{1: "0x80", 2: "0x8000", 4: "0x80000000", 8: "0x8000000000000000"}

func (t *X64Target) overflowCheck(okJump string) {
	ok := t.newLabel("no_overflow")
	t.text.printft("%s %s\n", okJump, ok)
	t.panic("integer overflow")
	t.text.printf("%s:\n", ok)
}

func (t *X64Target) checkedArithmetic(op string, left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	signed := isSigned(tp)

	if op == "*" {
		return t.checkedMultiply(left, right, tp)
	}

	leftReg := t.materialize(left, size)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)

	inst := "add"
	if op == "-" {
		inst = "sub"
	}
	t.text.printft("%s %s, %s\n", inst, t.ctx.GetRegister(leftReg, size), opText)
	if signed {
		t.overflowCheck("jno")
	} else {
		t.overflowCheck("jnc")
	}
	return Reg{Reg: leftReg, Size: size}, nil
}

func (t *X64Target) widened(op Operand, tp analyser.Type, size int) x64context.Register {
	return t.materialize(t.widen(op, tp, size), size)
}

func (t *X64Target) checkedMultiply(left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	signed := isSigned(tp)
	if _, ok := left.(Imm); ok {
		left, right = right, left
	}

	switch {
	case signed && size >= 2:
		leftReg := t.materialize(left, size)
		opText := t.opText(right, size)
		if t.err != nil {
			return nil, t.err
		}
		t.freeOp(right)
		t.text.printft("imul %s, %s\n", t.ctx.GetRegister(leftReg, size), opText)
		t.overflowCheck("jno")
		return Reg{leftReg, size}, nil
	case signed:
		l := t.widened(left, tp, 4)
		r := t.widened(right, tp, 4)
		l32, r32 := t.ctx.GetRegister(l, 4), t.ctx.GetRegister(r, 4)
		t.text.printft("imul %s, %s\n", l32, r32)
		t.ctx.FreeRegister(r)
		tmp := t.ctx.AllocFreeRegister()
		t.text.printft("movsx %s, %s\n", t.ctx.GetRegister(tmp, 4), t.ctx.GetRegister(l, 1))
		t.text.printft("cmp %s, %s\n", t.ctx.GetRegister(tmp, 4), l32)
		t.ctx.FreeRegister(tmp)
		t.overflowCheck("je")
		return Reg{l, 1}, nil
	case size <= 2:
		l := t.widened(left, tp, 4)
		r := t.widened(right, tp, 4)
		l32 := t.ctx.GetRegister(l, 4)
		t.text.printft("imul %s, %s\n", l32, t.ctx.GetRegister(r, 4))
		t.ctx.FreeRegister(r)
		t.text.printft("cmp %s, %d\n", l32, 1<<(8*size)-1)
		t.overflowCheck("jbe")
		return Reg{l, size}, nil
	case size == 4:
		l := t.widened(left, tp, 8)
		r := t.widened(right, tp, 8)
		l64 := t.ctx.GetRegister(l, 8)
		t.text.printft("imul %s, %s\n", l64, t.ctx.GetRegister(r, 8))
		t.ctx.FreeRegister(r)
		tmp := t.ctx.AllocFreeRegister()
		tmp64 := t.ctx.GetRegister(tmp, 8)
		t.text.printft("mov %s, %s\n", tmp64, l64)
		t.text.printft("shr %s, 32\n", tmp64)
		t.ctx.FreeRegister(tmp)
		t.overflowCheck("jz")
		return Reg{l, 4}, nil
	}

	return t.unsignedMultiply64(left, right, func() { t.overflowCheck("jnc") })
}

func (t *X64Target) unsignedMultiply64(left, right Operand, afterMul func()) (Operand, error) {
	leftReg := t.materialize(left, 8)
	t.freeOp(right)

	var saved []x64context.Register
	for _, reg := range []x64context.Register{x64context.AX, x64context.DX} {
		if reg != leftReg && slices.Contains(t.ctx.AllocatedRegisters(), reg) {
			t.push(reg)
			t.ctx.FreeRegister(reg)
			saved = append(saved, reg)
		}
	}

	t.push(leftReg)
	t.ctx.FreeRegister(leftReg)
	t.pushValue(right)

	t.ctx.AllocRegister(x64context.AX)
	t.ctx.AllocRegister(x64context.DX)
	factor := t.ctx.AllocFreeRegister()
	t.pop(factor)
	t.pop(x64context.AX)

	t.text.printft("mul %s\n", t.ctx.GetRegister(factor, 8))
	afterMul()

	result := t.ctx.AllocFreeRegister()
	t.text.printft("mov %s, rax\n", t.ctx.GetRegister(result, 8))
	t.ctx.FreeRegister(factor)
	t.ctx.FreeRegister(x64context.AX)
	t.ctx.FreeRegister(x64context.DX)
	for _, reg := range slices.Backward(saved) {
		t.pop(reg)
		t.ctx.AllocRegister(reg)
	}
	return Reg{result, 8}, nil
}

func (t *X64Target) saturatingArithmetic(op string, left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	signed := isSigned(tp)

	if op == "*|" {
		return t.saturatingMultiply(left, right, tp)
	}

	leftReg := t.materialize(left, size)
	opText := t.opText(right, size)
	if t.err != nil {
		return nil, t.err
	}
	t.freeOp(right)

	reg := t.ctx.GetRegister(leftReg, size)
	inst := "add"
	if op == "-|" {
		inst = "sub"
	}
	t.text.printft("%s %s, %s\n", inst, reg, opText)

	done := t.newLabel("saturate_done")
	if !signed {
		t.text.printft("jnc %s\n", done)
		if op == "+|" {
			t.text.printft("or %s, -1\n", reg)
		} else {
			t.text.printft("xor %s, %s\n", t.ctx.GetRegister(leftReg, 4), t.ctx.GetRegister(leftReg, 4))
		}
		t.text.printf("%s:\n", done)
		return Reg{leftReg, size}, nil
	}

	high := t.newLabel("saturate_high")
	t.text.printft("jno %s\n", done)
	t.text.printft("js %s\n", high)
	t.text.printft("mov %s, %s\n", reg, signedMin[size])
	t.text.printft("jmp %s\n", done)
	t.text.printf("%s:\n", high)
	t.text.printft("mov %s, %s\n", reg, signedMax[size])
	t.text.printf("%s:\n", done)
	return Reg{leftReg, size}, nil
}

func (t *X64Target) saturatingMultiply(left, right Operand, tp analyser.Type) (Operand, error) {
	size := sizeOf(tp)
	signed := isSigned(tp)
	if _, ok := left.(Imm); ok {
		left, right = right, left
	}
	done := t.newLabel("saturate_done")

	switch {
	case signed && size <= 4:
		wide := 4
		if size == 4 {
			wide = 8
		}
		l := t.widened(left, tp, wide)
		r := t.widened(right, tp, wide)
		lw := t.ctx.GetRegister(l, wide)
		t.text.printft("imul %s, %s\n", lw, t.ctx.GetRegister(r, wide))
		t.ctx.FreeRegister(r)
		upper := t.newLabel("saturate_in_high")
		maxText := fmt.Sprintf("%d", int64(1)<<(8*size-1)-1)
		minText := fmt.Sprintf("%d", -(int64(1) << (8*size - 1)))
		t.text.printft("cmp %s, %s\n", lw, maxText)
		t.text.printft("jle %s\n", upper)
		t.text.printft("mov %s, %s\n", lw, maxText)
		t.text.printft("jmp %s\n", done)
		t.text.printf("%s:\n", upper)
		t.text.printft("cmp %s, %s\n", lw, minText)
		t.text.printft("jge %s\n", done)
		t.text.printft("mov %s, %s\n", lw, minText)
		t.text.printf("%s:\n", done)
		return Reg{l, size}, nil
	case signed:
		l := t.materialize(left, 8)
		r := t.materialize(right, 8)
		sign := t.ctx.AllocFreeRegister()
		l64, r64, s64 := t.ctx.GetRegister(l, 8), t.ctx.GetRegister(r, 8), t.ctx.GetRegister(sign, 8)
		t.text.printft("mov %s, %s\n", s64, l64)
		t.text.printft("xor %s, %s\n", s64, r64)
		t.text.printft("imul %s, %s\n", l64, r64)
		t.ctx.FreeRegister(r)
		t.text.printft("jno %s\n", done)
		low := t.newLabel("saturate_low")
		t.text.printft("test %s, %s\n", s64, s64)
		t.text.printft("js %s\n", low)
		t.text.printft("mov %s, %s\n", l64, signedMax[8])
		t.text.printft("jmp %s\n", done)
		t.text.printf("%s:\n", low)
		t.text.printft("mov %s, %s\n", l64, signedMin[8])
		t.text.printf("%s:\n", done)
		t.ctx.FreeRegister(sign)
		return Reg{l, 8}, nil
	case size <= 2:
		l := t.widened(left, tp, 4)
		r := t.widened(right, tp, 4)
		l32 := t.ctx.GetRegister(l, 4)
		t.text.printft("imul %s, %s\n", l32, t.ctx.GetRegister(r, 4))
		t.ctx.FreeRegister(r)
		maxText := fmt.Sprintf("%d", 1<<(8*size)-1)
		t.text.printft("cmp %s, %s\n", l32, maxText)
		t.text.printft("jbe %s\n", done)
		t.text.printft("mov %s, %s\n", l32, maxText)
		t.text.printf("%s:\n", done)
		return Reg{l, size}, nil
	case size == 4:
		l := t.widened(left, tp, 8)
		r := t.widened(right, tp, 8)
		l64 := t.ctx.GetRegister(l, 8)
		t.text.printft("imul %s, %s\n", l64, t.ctx.GetRegister(r, 8))
		t.ctx.FreeRegister(r)
		tmp := t.ctx.AllocFreeRegister()
		tmp64 := t.ctx.GetRegister(tmp, 8)
		t.text.printft("mov %s, %s\n", tmp64, l64)
		t.text.printft("shr %s, 32\n", tmp64)
		t.ctx.FreeRegister(tmp)
		t.text.printft("jz %s\n", done)
		t.text.printft("mov %s, 0xffffffff\n", t.ctx.GetRegister(l, 4))
		t.text.printf("%s:\n", done)
		return Reg{l, 4}, nil
	}

	return t.unsignedMultiply64(left, right, func() {
		t.text.printft("jnc %s\n", done)
		t.text.printlnt("mov rax, -1")
		t.text.printf("%s:\n", done)
	})
}

func (t *X64Target) loadBound(text string) x64context.Register {
	reg := t.ctx.AllocFreeRegister()
	t.text.printft("mov %s, %s\n", t.ctx.GetRegister(reg, 8), text)
	return reg
}

func conversionNeedsCheck(src, dst analyser.Type) bool {
	sb, db := sizeOf(src)*8, sizeOf(dst)*8
	ss, ds := isSigned(src), isSigned(dst)
	switch {
	case ss == ds:
		return db < sb
	case ss && !ds:
		return true
	default:
		return db <= sb
	}
}

func (t *X64Target) conversionFailure(fail string) {
	ok := t.newLabel("convert_ok")
	t.text.printft("jmp %s\n", ok)
	t.text.printf("%s:\n", fail)
	t.panic("integer conversion")
	t.text.printf("%s:\n", ok)
}

func (t *X64Target) checkedConversion(op Operand, src, dst analyser.Type) Operand {
	reg := t.widened(op, src, 8)
	r64 := t.ctx.GetRegister(reg, 8)
	fail := t.newLabel("convert_fail")
	dstSize := sizeOf(dst)
	bits := dstSize * 8

	if isSigned(dst) {
		if isSigned(src) {
			hi := t.loadBound(fmt.Sprintf("%d", uint64(1)<<(bits-1)-1))
			t.text.printft("cmp %s, %s\n", r64, t.ctx.GetRegister(hi, 8))
			t.text.printft("jg %s\n", fail)
			t.ctx.FreeRegister(hi)
			lo := t.loadBound(fmt.Sprintf("-%d", uint64(1)<<(bits-1)))
			t.text.printft("cmp %s, %s\n", r64, t.ctx.GetRegister(lo, 8))
			t.text.printft("jl %s\n", fail)
			t.ctx.FreeRegister(lo)
		} else if bits == 64 {
			t.text.printft("test %s, %s\n", r64, r64)
			t.text.printft("js %s\n", fail)
		} else {
			hi := t.loadBound(fmt.Sprintf("%d", uint64(1)<<(bits-1)-1))
			t.text.printft("cmp %s, %s\n", r64, t.ctx.GetRegister(hi, 8))
			t.text.printft("ja %s\n", fail)
			t.ctx.FreeRegister(hi)
		}
	} else {
		if isSigned(src) {
			t.text.printft("test %s, %s\n", r64, r64)
			t.text.printft("js %s\n", fail)
		}
		if bits < 64 {
			hi := t.loadBound(fmt.Sprintf("%d", uint64(1)<<bits-1))
			t.text.printft("cmp %s, %s\n", r64, t.ctx.GetRegister(hi, 8))
			t.text.printft("ja %s\n", fail)
			t.ctx.FreeRegister(hi)
		}
	}
	t.conversionFailure(fail)
	return Reg{reg, dstSize}
}

func (t *X64Target) saturateInteger(op Operand, src, dst analyser.Type) Operand {
	reg := t.widened(op, src, 8)
	r64 := t.ctx.GetRegister(reg, 8)
	dstSize := sizeOf(dst)
	bits := dstSize * 8

	clamp := func(bound, cmov string) {
		b := t.loadBound(bound)
		b64 := t.ctx.GetRegister(b, 8)
		t.text.printft("cmp %s, %s\n", r64, b64)
		t.text.printft("%s %s, %s\n", cmov, r64, b64)
		t.ctx.FreeRegister(b)
	}

	switch {
	case isSigned(dst) && isSigned(src):
		if bits < 64 {
			clamp(fmt.Sprintf("%d", uint64(1)<<(bits-1)-1), "cmovg")
			clamp(fmt.Sprintf("-%d", uint64(1)<<(bits-1)), "cmovl")
		}
	case isSigned(dst):
		clamp(fmt.Sprintf("%d", uint64(1)<<(bits-1)-1), "cmova")
	case isSigned(src):
		zero := t.loadBound("0")
		t.text.printft("test %s, %s\n", r64, r64)
		t.text.printft("cmovs %s, %s\n", r64, t.ctx.GetRegister(zero, 8))
		t.ctx.FreeRegister(zero)
		if bits < 64 {
			clamp(fmt.Sprintf("%d", uint64(1)<<bits-1), "cmova")
		}
	default:
		if bits < 64 {
			clamp(fmt.Sprintf("%d", uint64(1)<<bits-1), "cmova")
		}
	}
	return Reg{reg, dstSize}
}

func (t *X64Target) floatLiteralText(value float64, size int) (string, error) {
	mem, err := t.resolveFloatLiteral(value, size)
	if err != nil {
		return "", err
	}
	return t.opText(mem, size), nil
}

func (t *X64Target) floatBounds(dst analyser.Type, size int) (lo, hi string, err error) {
	bits := sizeOf(dst) * 8
	lowValue, highValue := 0.0, float64(uint64(1)<<(bits-1))*2
	if isSigned(dst) {
		lowValue = -float64(uint64(1) << (bits - 1))
		highValue = float64(uint64(1) << (bits - 1))
	}
	if bits == 64 && !isSigned(dst) {
		highValue = 18446744073709551616.0
	}
	if lo, err = t.floatLiteralText(lowValue, size); err != nil {
		return
	}
	hi, err = t.floatLiteralText(highValue, size)
	return
}

func (t *X64Target) checkFloatRange(xmm x64context.Register, src, dst analyser.Type) error {
	size := sizeOf(src)
	inst := "sd"
	if size == 4 {
		inst = "ss"
	}
	x := t.ctx.GetRegister(xmm, size)
	fail := t.newLabel("convert_fail")

	lo, hi, err := t.floatBounds(dst, size)
	if err != nil {
		return err
	}
	jumpLow := "jb"
	if !isSigned(dst) {
		negative, err := t.floatLiteralText(-1.0, size)
		if err != nil {
			return err
		}
		lo, jumpLow = negative, "jbe"
	}

	t.text.printft("ucomi%s %s, %s\n", inst, x, x)
	t.text.printft("jp %s\n", fail)
	t.text.printft("ucomi%s %s, %s\n", inst, x, lo)
	t.text.printft("%s %s\n", jumpLow, fail)
	t.text.printft("ucomi%s %s, %s\n", inst, x, hi)
	t.text.printft("jae %s\n", fail)
	t.conversionFailure(fail)
	return nil
}

func (t *X64Target) saturateFloat(op Operand, src, dst analyser.Type) (Operand, error) {
	size := sizeOf(src)
	inst := "sd"
	if size == 4 {
		inst = "ss"
	}
	xmm := t.materializeXmm(op)
	x := t.ctx.GetRegister(xmm, size)
	reg := t.ctx.AllocFreeRegister()
	r64 := t.ctx.GetRegister(reg, 8)
	dstSize := sizeOf(dst)
	bits := dstSize * 8

	lo, hi, err := t.floatBounds(dst, size)
	if err != nil {
		return nil, err
	}
	done := t.newLabel("saturate_done")
	low := t.newLabel("saturate_low")
	high := t.newLabel("saturate_high")

	t.text.printft("xor %s, %s\n", t.ctx.GetRegister(reg, 4), t.ctx.GetRegister(reg, 4))
	t.text.printft("ucomi%s %s, %s\n", inst, x, x)
	t.text.printft("jp %s\n", done)
	t.text.printft("ucomi%s %s, %s\n", inst, x, lo)
	t.text.printft("jbe %s\n", low)
	t.text.printft("ucomi%s %s, %s\n", inst, x, hi)
	t.text.printft("jae %s\n", high)
	if isUint64(dst) {
		big := t.newLabel("saturate_big")
		half, err := t.floatLiteralText(9223372036854775808.0, size)
		if err != nil {
			return nil, err
		}
		tmp := t.ctx.AllocFreeRegister()
		t.text.printft("ucomi%s %s, %s\n", inst, x, half)
		t.text.printft("jae %s\n", big)
		t.text.printft("cvtt%s2si %s, %s\n", inst, r64, x)
		t.text.printft("jmp %s\n", done)
		t.text.printf("%s:\n", big)
		t.text.printft("sub%s %s, %s\n", inst, x, half)
		t.text.printft("cvtt%s2si %s, %s\n", inst, r64, x)
		t.text.printft("mov %s, 0x8000000000000000\n", t.ctx.GetRegister(tmp, 8))
		t.text.printft("xor %s, %s\n", r64, t.ctx.GetRegister(tmp, 8))
		t.ctx.FreeRegister(tmp)
	} else {
		t.text.printft("cvtt%s2si %s, %s\n", inst, r64, x)
	}
	t.text.printft("jmp %s\n", done)
	t.text.printf("%s:\n", low)
	if isSigned(dst) {
		t.text.printft("mov %s, -%d\n", r64, uint64(1)<<(bits-1))
	}
	t.text.printft("jmp %s\n", done)
	t.text.printf("%s:\n", high)
	switch {
	case isSigned(dst):
		t.text.printft("mov %s, %d\n", r64, uint64(1)<<(bits-1)-1)
	case bits == 64:
		t.text.printft("mov %s, -1\n", r64)
	default:
		t.text.printft("mov %s, %d\n", r64, uint64(1)<<bits-1)
	}
	t.text.printf("%s:\n", done)
	t.ctx.FreeRegister(xmm)
	t.freeOp(op)
	return Reg{reg, dstSize}, nil
}

func (t *X64Target) checkedIncDec(stmt *ast.IncDecStmt, target Operand, tp analyser.Type) {
	size := sizeOf(tp)
	inst := "add"
	if stmt.Op == "--" {
		inst = "sub"
	}
	t.text.printft("%s %s, 1\n", inst, t.opText(target, size))
	if isSigned(tp) {
		t.overflowCheck("jno")
	} else {
		t.overflowCheck("jnc")
	}
}
