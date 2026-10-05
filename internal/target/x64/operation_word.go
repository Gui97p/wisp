package x64

import (
	"slices"

	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) loadWord(dst x64context.Register, w Operand) {
	switch o := w.(type) {
	case Imm:
		t.text.printft("mov %s, %d\n", t.ctx.GetRegister(dst, 8), o)
	case Reg:
		if o.Reg != dst {
			t.text.printft("mov %s, %s\n", t.ctx.GetRegister(dst, o.Size), t.ctx.GetRegister(o.Reg, o.Size))
		}
	case Mem:
		t.text.printft("mov %s, %s\n", t.ctx.GetRegister(dst, o.Size), t.memText(o))
	case Addr:
		t.text.printft("lea %s, %s\n", t.ctx.GetRegister(dst, 8), t.memText(o.Of))
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
}

func (t *X64Target) storeWord(dst Mem, w Operand) {
	switch o := w.(type) {
	case Imm:
		t.text.printft("mov %s, %d\n", t.memText(dst), o)
	case Reg:
		t.text.printft("mov %s, %s\n", t.memText(dst), t.ctx.GetRegister(o.Reg, dst.Size))
	case Mem:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return
		}
		t.loadWord(reg, o)
		t.text.printft("mov %s, %s\n", t.memText(dst), t.ctx.GetRegister(reg, o.Size))
		t.ctx.FreeRegister(reg)
	case Addr:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return
		}
		regStr := t.ctx.GetRegister(reg, 8)
		t.text.printft("lea %s, %s\n", regStr, t.memText(o.Of))
		t.text.printft("mov %s, %s\n", t.memText(dst), regStr)
		t.ctx.FreeRegister(reg)
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
}

func (t *X64Target) pushWord(w Operand) {
	switch o := w.(type) {
	case Imm:
		t.text.printft("push %d\n", o)
	case Reg:
		t.text.printft("push %s\n", t.ctx.GetRegister(o.Reg, 8))
	case Mem:
		if o.Size < 8 {
			reg := t.ctx.AllocFreeRegister()
			if reg == x64context.NoReg {
				t.fail("x86-64: no available registers")
				return
			}
			t.text.printft("mov %s, %s\n", t.ctx.GetRegister(reg, o.Size), t.memText(o))
			t.text.printft("push %s\n", t.ctx.GetRegister(reg, 8))
			t.ctx.FreeRegister(reg)
		} else {
			t.text.printft("push %s\n", t.memText(o))
		}
	case Addr:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return
		}
		regStr := t.ctx.GetRegister(reg, 8)
		t.text.printft("lea %s, %s\n", regStr, t.memText(o.Of))
		t.text.printft("push %s\n", regStr)
		t.ctx.FreeRegister(reg)
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
	t.ctx.AddPushed(1)
}

func (t *X64Target) freeWord(w Operand) {
	switch o := w.(type) {
	case Imm:
	case Reg:
		t.ctx.FreeRegister(o.Reg)
	case Mem:
		if slices.Contains(x64context.RegisterOrder, o.Base) {
			t.ctx.FreeRegister(o.Base)
		}
		if slices.Contains(x64context.RegisterOrder, o.Index) {
			t.ctx.FreeRegister(o.Index)
		}
	case Addr:
		t.freeWord(o.Of)
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
}
