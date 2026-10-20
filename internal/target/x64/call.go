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

func (t *X64Target) hiddenArg(tp analyser.Type, value Operand) *ast.IdentLiteral {
	size := sizeOf(tp)
	vsym := &analyser.Symbol{Kind: analyser.VAR, Type: tp}
	off := t.ctx.Set(vsym, size)
	t.store(value, slot(off, size))

	ident := &ast.IdentLiteral{Value: "__wisp_arg"}
	t.info.Idents[ident] = vsym
	t.info.Types[ident] = tp
	return ident
}

func (t *X64Target) boxArg(dst Mem, arg analyser.Type, op Operand) {
	layout := analyser.Layout(arg)

	var tag Operand = Imm(analyser.BoxCode(layout.Code))
	if layout.Named() {
		mod, key, _ := analyser.DescriptorOf(arg)
		tag = Addr{Of: t.typeLabel(mod, key)}
	}
	t.storeWord(dst, tag)

	str := false
	if tp, ok := underlying(arg).(analyser.PrimitiveType); ok && tp.Name == "string" {
		str = true
	}

	if str {
		t.store(op, dst.at(8, 16))
	} else {
		widenOp := t.widen(op, arg, 8)
		t.storeWord(dst.at(8, 8), widenOp)
		t.freeOp(widenOp)

		var w1 Operand = Imm(0)
		switch layout.Identity {
		case "", "void":
		default:
			if c := analyser.BoxCode(layout.Identity); c != 0 {
				w1 = Imm(c)
				break
			}

			mod, key, _ := analyser.DescriptorOf(arg)
			w1 = Addr{Of: t.typeLabel(mod, key)}
		}
		t.storeWord(dst.at(16, 8), w1)
	}
}

func (t *X64Target) compileVariadicCall(sym *analyser.Symbol, extern bool, expr *ast.CallExpr, ft *analyser.FuncType, rets []analyser.Type) ([]Operand, error) {
	fixed := len(ft.Params) - 1
	elem := ft.Params[fixed]
	esize := sizeOf(elem)
	extra := len(expr.Args) - fixed

	args := make([]ast.Expression, 0, fixed+1)
	for _, arg := range expr.Args[:fixed] {
		op, err := t.compileExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, t.hiddenArg(t.info.Types[arg], op))
	}

	if expr.Spread {
		extra--
		arg := expr.Args[len(expr.Args)-1]
		if len(expr.Args) == fixed+1 {
			op, err := t.compileExpr(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, t.hiddenArg(analyser.SpanType{Element: elem}, op))
			return t.compileCall(t.funcSymbol(sym.Module, sym.Name), extern, args, rets, false)
		}

		rspMem := slot(t.ctx.Reserve(8), 8)
		t.text.printft("mov %s, rsp\n", t.memText(rspMem))

		stride := esize
		if _, ok := t.info.Types[arg].(analyser.BoxedType); ok {
			stride = 24
		}

		op, err := t.compileExpr(arg)
		if err != nil {
			return nil, err
		}

		spanSlot := slot(t.ctx.Reserve(16), 16)
		t.store(op, spanSlot)
		lenReg := t.materialize(spanSlot.at(8, 8), 8)
		cReg := t.ctx.AllocFreeRegister()
		t.loadWord(cReg, Reg{lenReg, 8})
		cStr := t.ctx.GetRegister(cReg, 8)
		t.text.printft("add %s, %d\n", cStr, extra)
		t.text.printft("imul %s, %d\n", cStr, stride)
		t.text.printft("add %s, 15\nand %s, -16\n", cStr, cStr)
		t.text.printft("sub rsp, %s\n", cStr)
		t.ctx.FreeRegister(cReg)

		for i := range extra {
			op, err := t.compileExpr(expr.Args[fixed+i])
			if err != nil {
				return nil, err
			}
			if arg, boxed := t.info.Boxes[expr.Args[fixed+i]]; boxed {
				t.boxArg(Mem{Base: x64context.SP, Disp: stride * i, Size: 24}, arg, op)
			} else {
				t.store(op, Mem{Base: x64context.SP, Disp: i * esize, Size: esize})
			}
		}

		t.push(x64context.SI)
		t.push(x64context.DI)
		t.push(x64context.CX)

		t.loadWord(x64context.SI, spanSlot.at(0, 8))
		t.text.printft("lea rdi, [rsp + 24 + %d*%d]\n", extra, stride)
		t.loadWord(x64context.CX, spanSlot.at(8, 8))
		t.text.printft("imul rcx, %d\n", stride)
		t.text.printlnt("rep movsb")

		t.pop(x64context.CX)
		t.pop(x64context.DI)
		t.pop(x64context.SI)

		t.text.printft("add %s, %d\n", t.ctx.GetRegister(lenReg, 8), extra)
		args = append(args, t.hiddenArg(analyser.SpanType{Element: elem}, Wide{Words: []Operand{Addr{Of: Mem{Base: x64context.SP}}, Reg{lenReg, 8}}}))
		ops, err := t.compileCall(t.funcSymbol(sym.Module, sym.Name), extern, args, rets, false)
		if err != nil {
			return nil, err
		}
		t.text.printft("mov rsp, %s\n", t.memText(rspMem))
		return ops, nil
	}

	block := slot(t.ctx.Reserve((extra*esize+7)/8*8), extra*esize)
	for i := range extra {
		op, err := t.compileExpr(expr.Args[fixed+i])
		if err != nil {
			return nil, err
		}
		if arg, boxed := t.info.Boxes[expr.Args[fixed+i]]; boxed {
			t.boxArg(block.at(i*24, 24), arg, op)
		} else {
			t.store(op, block.at(i*esize, esize))
		}
	}

	var ptr Operand = Imm(0)
	if extra > 0 {
		ptr = Addr{Of: block}
	}

	args = append(args, t.hiddenArg(analyser.SpanType{Element: elem}, Wide{Words: []Operand{ptr, Imm(extra)}}))
	return t.compileCall(t.funcSymbol(sym.Module, sym.Name), extern, args, rets, false)
}

func (t *X64Target) compileCall(sym string, extern bool, args []ast.Expression, rets []analyser.Type, cVariadic bool) ([]Operand, error) {
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
				if _, ok := underlying(types[i]).(analyser.PrimitiveType); ok && size < 4 && size > 0 {
					op = t.widen(op, types[i], 4)
				}
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

	if cVariadic {
		t.text.printlnt("xor eax, eax")
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
		return t.compileCall(sym, mc.Module != t.module, args, t.info.CallReturns[expr], false)
	}

	sym, err := t.resolveCallee(expr.Name)
	if err != nil {
		return nil, err
	}
	if sym.CName != "" {
		cVariadic := false
		if ft, ok := sym.Type.(*analyser.FuncType); ok {
			cVariadic = ft.CVariadic
		}
		return t.compileCall(sym.CName, sym.Extern || sym.Module != t.module, expr.Args, t.info.CallReturns[expr], cVariadic)
	}
	if sym.Module == "" {
		return t.compileBuiltin(sym.Name, expr)
	}

	if ft, ok := sym.Type.(*analyser.FuncType); ok {
		if ft.Variadic {
			return t.compileVariadicCall(sym, sym.Module != t.module, expr, ft, t.info.CallReturns[expr])
		}
	}

	return t.compileCall(t.funcSymbol(sym.Module, sym.Name), sym.Module != t.module, expr.Args, t.info.CallReturns[expr], false)
}
