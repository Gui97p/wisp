package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) resolveIntLiteral(value int64) (Operand, error) {
	if int64(int32(value)) != value {
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			return nil, fmt.Errorf("x86-64: no available registers")
		}
		t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, 8), value)
		return Reg{Reg: reg, Size: 8}, nil
	}
	return Imm(value), nil
}

func (*X64Target) resolveBoolLiteral(value bool) (Operand, error) {
	if value {
		return Imm(1), nil
	} else {
		return Imm(0), nil
	}
}

func (t *X64Target) resolveStringLiteral(value string) (Operand, error) {
	decoded, err := decodeEscapes(value)
	if err != nil {
		return nil, err
	}
	label, ok := t.createData("str", string(decoded))
	if !ok {
		t.rodata.printf("%s db %s\n", label, bytesToAsm(append(decoded, 0)))
	}
	return strLit(label, len(decoded)), nil
}

func (t *X64Target) compileErrorLiteral(e *analyser.ErrorLiteral) (Operand, error) {
	dl, dn, err := t.stringData(e.Domain)
	if err != nil {
		return nil, err
	}
	ml, mn, err := t.stringData(e.Message)
	if err != nil {
		return nil, err
	}
	fl, fn, err := t.stringData(e.File)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s|%d|%s|%s|%d", e.Domain, e.Code, e.Message, e.File, e.Line)
	l, ok := t.createData("err", key)
	if !ok {
		t.rodata.printf("%s: dq %s, %d, %d, %s, %d, %s, %d, %d\n", l, dl, dn, e.Code, ml, mn, fl, fn, e.Line)
	}

	reg := t.ctx.AllocFreeRegister()
	t.text.printft("lea %s, [rel %s]\n", t.ctx.GetRegister(reg, 8), l)
	return Reg{reg, 8}, nil
}

var errFieldOffset = map[string]int{
	"domain":  0,
	"code":    16,
	"message": 24,
	"file":    40,
	"line":    56,
}
