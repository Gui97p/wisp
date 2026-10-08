package x64

import (
	"fmt"

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
