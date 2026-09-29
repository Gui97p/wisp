package x64

import (
	"fmt"

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
