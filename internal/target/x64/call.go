package x64

import (
	"github.com/Gui97p/wisp/internal/analyser"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func usesSret(types []analyser.Type) bool {
	total := 0
	for _, tp := range types {
		classes := classify(tp)
		length := len(classes)

		if length != 0 && classes[0] == x64context.Memory {
			return true
		}

		total += length
	}

	return total > 2
}

func retOffsets(types []analyser.Type) []int {
	offsets := []int{}

	offset := 0
	for _, tp := range types {
		offsets = append(offsets, offset)
		offset += (sizeOf(tp) + 7) / 8 * 8
	}

	return offsets
}

func classify(tp analyser.Type) []x64context.Class {
	size := sizeOf(tp)

	if size == 0 {
		return []x64context.Class{}
	}

	if size <= 8 {
		return []x64context.Class{x64context.Int}
	}

	if size <= 16 {
		return []x64context.Class{x64context.Int, x64context.Int}
	}

	return []x64context.Class{x64context.Memory}
}

func assignArgs(types []analyser.Type, sret bool) []x64context.ArgLoc {
	argLocs := []x64context.ArgLoc{}

	pi := 0
	if sret {
		pi = 1
	}
	stack := 0

	for _, tp := range types {
		classes := classify(tp)
		if len(classes) == 0 {
			argLocs = append(argLocs, x64context.ArgLoc{})
			continue
		} else if classes[0] == x64context.Memory {
			argLocs = append(argLocs, x64context.ArgLoc{
				Stack: stack,
			})
			stack += (sizeOf(tp) + 7) / 8 * 8
		} else {
			need := len(classes)
			if pi+need <= len(x64context.ParamOrder) {
				argLoc := x64context.ArgLoc{}
				for i := 0; i < need; i++ {
					argLoc.Regs = append(argLoc.Regs, x64context.ParamOrder[pi+i])
				}
				argLocs = append(argLocs, argLoc)
				pi += need
			} else {
				argLocs = append(argLocs, x64context.ArgLoc{
					Stack: stack,
				})
				stack += (sizeOf(tp) + 7) / 8 * 8
			}
		}
	}

	return argLocs
}

func (t *X64Target) captureReturn(types []analyser.Type, slot Mem) []Operand {
	if len(types) == 0 {
		return nil
	}

	ops := []Operand{}

	if usesSret(types) {
		offsets := retOffsets(types)
		for i, tp := range types {
			ops = append(ops, slot.at(offsets[i], sizeOf(tp)))
		}
	} else {
		n := 0
		for _, tp := range types {
			n += len(classify(tp))
		}

		rs := []x64context.Register{}
		for range n {
			rs = append(rs, t.ctx.AllocFreeRegister())
		}

		if len(rs) == 1 {
			if rs[0] != x64context.AX {
				t.text.printft("mov %s, rax\n", t.ctx.GetRegister(rs[0], 8))
			}
		} else if n > 0 {
			r1, r2 := rs[0], rs[1]

			if r1 == x64context.DX {
				if r2 == x64context.AX {
					t.text.printlnt("xchg rax, rdx")
				} else {
					t.text.printft("mov %s, rdx\n", t.ctx.GetRegister(r2, 8))
					t.text.printlnt("mov rdx, rax")
				}
			} else {
				if r1 != x64context.AX {
					t.text.printft("mov %s, rax\n", t.ctx.GetRegister(r1, 8))
				}

				if r2 != x64context.DX {
					t.text.printft("mov %s, rdx\n", t.ctx.GetRegister(r2, 8))
				}
			}
		}

		k := 0
		for _, tp := range types {
			w := len(classify(tp))
			if w == 1 {
				ops = append(ops, Reg{Reg: rs[k], Size: sizeOf(tp)})
			} else if w == 2 {
				ops = append(ops, Wide{Words: []Operand{Reg{rs[k], 8}, Reg{rs[k+1], 8}}, Size: 16})
			}
			k += w
		}
	}

	return ops
}

func (t *X64Target) returnValue(ops []Operand, types []analyser.Type, hidden Mem) {
	if len(ops) == 0 {
		return
	}

	if usesSret(types) {
		p := t.ctx.AllocFreeRegister()
		t.loadWord(p, hidden)

		offsets := retOffsets(types)
		for i := range ops {
			t.store(ops[i], deref(p, offsets[i], sizeOf(types[i])))
		}
		t.ctx.FreeRegister(p)
		t.loadWord(x64context.AX, hidden)
	} else {
		n := 0
		for _, op := range ops {
			n += t.pushValue(op)
		}

		if n == 2 {
			t.pop(x64context.DX)
			n--
		}

		if n == 1 {
			t.pop(x64context.AX)
		}
	}
}
