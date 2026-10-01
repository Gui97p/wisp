package x64

import (
	"fmt"
	"slices"
	"strings"

	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

type Operand interface {
	op()
}

type Imm int64

func (Imm) op() {}

type Reg struct {
	Reg  x64context.Register
	Size int
}

func (Reg) op() {}

type Mem struct {
	Base  x64context.Register
	Index x64context.Register
	Scale int
	Disp  int
	Label string
	Size  int
}

func (Mem) op() {}
func (m Mem) at(delta, size int) Mem {
	m.Disp += delta
	m.Size = size
	return m
}

type Addr struct {
	Of Mem
}

func (Addr) op() {}

type Wide struct {
	Words []Operand
	Size  int
}

func (Wide) op() {}

func slot(offset, size int) Mem {
	return Mem{
		Base: x64context.BP,
		Disp: -offset,
		Size: size,
	}
}

func label(l string, size int) Mem {
	return Mem{
		Label: l,
		Size:  size,
	}
}

func deref(reg x64context.Register, disp, size int) Mem {
	return Mem{
		Base: reg,
		Disp: disp,
		Size: size,
	}
}

func indexed(reg, index x64context.Register, scale, disp, size int) (Mem, error) {
	if reg == x64context.NoReg {
		return Mem{}, fmt.Errorf("x86-64: indexed constructor cannot have empty reg")
	}
	if !slices.Contains([]int{1, 2, 4, 8}, scale) {
		return Mem{}, fmt.Errorf("x86-64: scale out of range (%d)", scale)
	}
	return Mem{
		Base:  reg,
		Index: index,
		Scale: scale,
		Disp:  disp,
		Size:  size,
	}, nil
}

func strLit(l string, n int) Wide {
	return Wide{Words: []Operand{Addr{Of: label(l, 8)}, Imm(n)}, Size: 16}
}

func (t *X64Target) memText(m Mem) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s [", sizeLabels[min(m.Size, 8)])

	if m.Label != "" {
		fmt.Fprintf(&b, "rel %s]", m.Label)
		return b.String()
	}

	if m.Base != x64context.NoReg {
		b.WriteString(t.ctx.GetRegister(m.Base, 8))
	}

	if m.Index != x64context.NoReg {
		fmt.Fprintf(&b, "+%s*%d", t.ctx.GetRegister(m.Index, 8), m.Scale)
	}

	if m.Disp != 0 {
		fmt.Fprintf(&b, "%+d", m.Disp)
	}

	b.WriteByte(']')
	return b.String()
}
