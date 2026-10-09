package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
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

func (t *X64Target) captureReturn(types []analyser.Type, memSlot Mem) []Operand {
	if len(types) == 0 {
		return nil
	}

	ops := []Operand{}

	if usesSret(types) {
		offsets := retOffsets(types)
		for i, tp := range types {
			ops = append(ops, memSlot.at(offsets[i], sizeOf(tp)))
		}
	} else {
		n := 0
		for _, tp := range types {
			n += len(classify(tp))
		}

		rs := []x64context.Register{}
		for range n {
			reg := t.ctx.AllocFreeRegister()
			if reg == x64context.NoReg {
				t.fail("x86-64: no available registers")
				return nil
			}
			rs = append(rs, reg)
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
			switch w {
			case 1:
				ops = append(ops, Reg{Reg: rs[k], Size: sizeOf(tp)})
			case 2:
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
		if p == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return
		}
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

func (t *X64Target) compileCall(sym string, extern bool, args []ast.Expression, rets []analyser.Type) ([]Operand, error) {
	pushed := t.ctx.Pushed()

	types := t.valueTypes(args)
	sret := usesSret(rets)
	argLocs := assignArgs(types, sret)

	regs := t.ctx.AllocatedRegisters()
	for _, reg := range regs {
		t.pushWord(Reg{reg, 8})
	}

	stackBytes := 0
	for i, arg := range argLocs {
		size := sizeOf(types[i])
		if len(arg.Regs) == 0 && size > 0 {
			stackBytes += (size + 7) / 8 * 8
		}
	}

	pad := 0
	if (t.ctx.Pushed()+stackBytes/8)%2 == 1 {
		pad = 1
	}
	t.alignStack(-(stackBytes + 8*pad))
	base := t.ctx.Pushed()

	err := t.eachValue(args, func(i int, op Operand) error {
		size := sizeOf(types[i])
		if size > 0 {
			if len(argLocs[i].Regs) == 0 {
				t.store(op, Mem{Base: x64context.SP, Disp: 8*(t.ctx.Pushed()-base) + argLocs[i].Stack, Size: size})
			} else {
				t.pushValue(op)
			}
		} else {
			t.freeOp(op)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, arg := range slices.Backward(argLocs) {
		for _, reg := range slices.Backward(arg.Regs) {
			t.pop(reg)
		}
	}

	var memSlot Mem
	if sret {
		lastIndex := len(rets) - 1
		offsets := retOffsets(rets)
		total := offsets[lastIndex] + (sizeOf(rets[lastIndex])+7)/8*8
		off := t.ctx.Reserve(total)

		memSlot = slot(off, total)
		t.loadWord(x64context.DI, Addr{Of: memSlot})
	}

	if extern {
		t.prelude.define(sym)
	}

	t.text.printft("call %s\n", sym)

	ops := t.captureReturn(rets, memSlot)
	t.alignStack(stackBytes + 8*pad)

	for _, reg := range slices.Backward(regs) {
		t.pop(reg)
	}

	if t.ctx.Pushed() != pushed {
		return nil, fmt.Errorf("x86-64: call stack unaligned")
	}

	return ops, nil
}

func (t *X64Target) compileCallValues(expr *ast.CallExpr) ([]Operand, error) {
	if e, ok := t.info.ErrorLiterals[expr]; ok {
		op, err := t.compileErrorLiteral(e)
		if err != nil {
			return nil, err
		}
		return []Operand{op}, nil
	}

	if expr.Cast != nil {
		op, err := t.compileExpr(expr.Cast)
		if err != nil {
			return nil, err
		}
		return []Operand{op}, nil
	}

	if mc, ok := t.info.MethodCalls[expr]; ok {
		sym := t.funcSymbol(mc.Module, fmt.Sprintf("%s_%s", mc.Owner, expr.Name.(*ast.MemberExpr).Field))
		args := append([]ast.Expression{mc.Receiver}, expr.Args...)
		return t.compileCall(sym, mc.Module != t.module, args, t.info.CallReturns[expr])
	}

	sym, err := t.resolveCallee(expr.Name)
	if err != nil {
		return nil, err
	}
	if sym.Module == "" {
		op, err := t.compileBuiltin(sym.Name, expr)
		if err != nil {
			return nil, err
		}
		if op == nil {
			return nil, nil
		}
		return []Operand{op}, nil
	}

	return t.compileCall(t.funcSymbol(sym.Module, sym.Name), sym.Module != t.module, expr.Args, t.info.CallReturns[expr])
}
