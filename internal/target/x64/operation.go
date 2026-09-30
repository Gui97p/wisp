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

func (t *X64Target) operandFree(op Operand) {
	o, ok := op.(RegOperand)
	if !ok {
		return
	}

	t.ctx.FreeRegister(o.Reg)
}

var setccOperators = map[string]string{
	"==": "sete",
	"!=": "setne",

	"<":  "setl",
	"<=": "setle",

	">":  "setg",
	">=": "setge",
}

func (t *X64Target) push(operand string) {
	t.text.printft("push %s\n", operand)
	t.ctx.AddPushed(1)
}

func (t *X64Target) pop(operand string) {
	t.text.printft("pop %s\n", operand)
	t.ctx.AddPushed(-1)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (t *X64Target) alignStack(n int) {
	if n == 0 {
		return
	}
	inst := "add"
	if n < 0 {
		inst = "sub"
	}
	t.ctx.AddPushed(-n / 8)
	t.text.printft("%s rsp, %d\n", inst, abs(n))
}
