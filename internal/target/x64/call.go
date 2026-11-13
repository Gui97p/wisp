package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func usesSret(types []analyser.Type) bool {
	totalInts, totalSSE := 0, 0
	for _, tp := range types {
		classes := classify(tp)
		length := len(classes)

		if length != 0 && classes[0] == x64context.Memory {
			return true
		}

		for _, class := range classes {
			if class == x64context.SSE {
				totalSSE++
			} else {
				totalInts++
			}
		}
	}

	return totalInts > 2 || totalSSE > 2
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

func markInts(tp analyser.Type, base int, classes []x64context.Class) {
	switch {
	case isFloat(tp):
	case isStruct(tp):
		st := structOf(tp)
		layout := layoutOf(st)

		for _, field := range layout.Fields {
			markInts(field.Type, base+field.Offset, classes)
		}
	case isArray(tp):
		at, _ := arrayOf(tp)
		size := sizeOf(at.Element)
		for i := range int(at.Size) {
			markInts(at.Element, base+i*size, classes)
		}
	default:
		size := sizeOf(tp)
		if size == 0 {
			return
		}
		start := base / 8
		end := (base + size - 1) / 8

		for i := start; i <= end; i++ {
			classes[i] = x64context.Int
		}
	}
}

func classify(tp analyser.Type) []x64context.Class {
	size := sizeOf(tp)

	if size == 0 {
		return []x64context.Class{}
	}

	if isFloat(tp) {
		return []x64context.Class{x64context.SSE}
	}

	if size > 16 {
		return []x64context.Class{x64context.Memory}
	}

	if !isStruct(tp) && !isArray(tp) {
		words := (size + 7) / 8
		classes := make([]x64context.Class, words)

		for i := range classes {
			classes[i] = x64context.Int
		}

		return classes
	}

	words := (size + 7) / 8
	classes := make([]x64context.Class, words)

	for i := range classes {
		classes[i] = x64context.SSE
	}

	markInts(tp, 0, classes)

	return classes
}

func assignArgs(types []analyser.Type, sret bool) []x64context.ArgLoc {
	argLocs := []x64context.ArgLoc{}

	pi, xi := 0, 0
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
			ints, sses := 0, 0
			for _, class := range classes {
				if class == x64context.SSE {
					sses++
				} else {
					ints++
				}
			}
			need := len(classes)
			if pi+ints <= len(x64context.ParamOrder) && xi+sses <= len(x64context.XMMParamOrder) {
				argLoc := x64context.ArgLoc{}
				ii, ss := 0, 0
				for i := range need {
					var reg x64context.Register
					if classes[i] == x64context.SSE {
						reg = x64context.XMMParamOrder[xi+ss]
						ss++
					} else {
						reg = x64context.ParamOrder[pi+ii]
						ii++
					}
					argLoc.Regs = append(argLoc.Regs, reg)
				}
				pi += ints
				xi += sses
				argLocs = append(argLocs, argLoc)
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

func (t *X64Target) assignWin64Args(types []analyser.Type) []x64context.ArgLoc {
	argLocs := []x64context.ArgLoc{}
	var i int
	for _, tp := range types {
		classes := classify(tp)
		if len(classes) == 0 {
			argLocs = append(argLocs, x64context.ArgLoc{})
			continue
		} else {
			for _, class := range classes {
				argLoc := x64context.ArgLoc{}
				if i < 4 {
					var reg x64context.Register
					if class == x64context.SSE {
						reg = x64context.XMMWin64ParamOrder[i]
					} else {
						reg = x64context.Win64ParamOrder[i]
					}
					argLoc.Regs = append(argLoc.Regs, reg)
				} else {
					argLoc.Stack = (32 + 8*(i-4))
				}
				argLocs = append(argLocs, argLoc)
				i++
			}
		}
	}
	return argLocs
}

func (t *X64Target) emitAdapter(a pendingAdapter) error {
	ft, ok := underlying(a.sym.Type).(*analyser.FuncType)
	if !ok {
		return fmt.Errorf("x86-64: %s is not a function", a.sym.Name)
	}
	if len(ft.Returns) > 1 || usesSret(ft.Returns) {
		return fmt.Errorf("x86-64: a C function returns at most one scalar")
	}
	for _, tp := range ft.Params {
		if sizeOf(tp) > 8 || classOf(tp) == x64context.Memory {
			return fmt.Errorf("x86-64: a C function takes only scalars and pointers")
		}
	}

	desc, err := newDesc(t.format, a.name, a.name, true, 32)
	if err != nil {
		return err
	}
	t.text.printf("\nsection .text%s\n", desc)
	t.text.printf("%s:\n", a.name)
	if a.sym.Extern || a.sym.Module != t.module {
		t.prelude.define(a.sym.CName)
	}

	from := assignArgs(ft.Params, false)
	to := t.assignWin64Args(ft.Params)

	t.text.printlnt("push rbp")
	t.text.printlnt("mov rbp, rsp")
	t.text.printlnt("sub rsp, .frame")

	saved := make([]int, len(ft.Params))
	for i, tp := range ft.Params {
		size := sizeOf(tp)
		if size == 0 || len(from[i].Regs) == 0 {
			continue
		}
		saved[i] = t.ctx.Reserve(8)
		t.storeWord(slot(saved[i], size), Reg{from[i].Regs[0], size})
	}
	t.ctx.Reserve(32 + 8*max(0, len(ft.Params)-4))

	for i, tp := range ft.Params {
		size := sizeOf(tp)
		if size == 0 {
			continue
		}
		var src Operand
		if len(from[i].Regs) > 0 {
			src = slot(saved[i], size)
		} else {
			src = deref(x64context.BP, 16+from[i].Stack, size)
		}
		if len(to[i].Regs) > 0 {
			t.loadWord(to[i].Regs[0], src)
			continue
		}
		t.loadWord(x64context.AX, src)
		t.storeWord(Mem{Base: x64context.SP, Disp: to[i].Stack, Size: size}, Reg{x64context.AX, size})
	}

	t.text.printft("call %s\n", a.sym.CName)
	t.text.printlnt("mov rsp, rbp")
	t.text.printlnt("pop rbp")
	t.text.printlnt("ret")
	t.text.printft(".frame equ %d\n", t.ctx.AlignTo(16))

	tText, err := textOf(t.format)
	if err != nil {
		return err
	}
	if tText != "" {
		tText = " " + tText
	}
	t.text.printf("section .text%s\n", tText)
	return nil
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
		rs := []x64context.Register{}
		srcs := []x64context.Register{}
		nr, nx := 0, 0
		for _, tp := range types {
			for _, class := range classify(tp) {
				var reg x64context.Register
				if class == x64context.SSE {
					reg = t.ctx.AllocFreeXmm()
					srcs = append(srcs, x64context.XMMReturnOrder[nx])
					nx++
				} else {
					reg = t.ctx.AllocFreeRegister()
					srcs = append(srcs, x64context.ReturnOrder[nr])
					nr++
				}
				if reg == x64context.NoReg {
					t.fail("x86-64: no available registers")
					return nil
				}
				rs = append(rs, reg)
				n++
			}
		}

		if len(rs) == 1 {
			t.loadWord(rs[0], Reg{srcs[0], 8})
		} else if n > 0 {
			for _, r := range srcs {
				t.push(r)
			}
			for _, r := range slices.Backward(rs) {
				t.pop(r)
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
		if t.registerReturn(types) {
			t.moveTo(x64context.ReturnOrder[:len(ops)], ops, nil)
			return
		}

		for _, op := range ops {
			t.pushValue(op)
		}

		nreg, nxmm := 0, 0
		rs := []x64context.Register{}
		for _, tp := range types {
			for _, word := range classify(tp) {
				if word == x64context.SSE {
					rs = append(rs, x64context.XMMReturnOrder[nxmm])
					nxmm++
				} else {
					rs = append(rs, x64context.ReturnOrder[nreg])
					nreg++
				}
			}
		}

		for _, r := range slices.Backward(rs) {
			t.pop(r)
		}
	}
}

func (t *X64Target) registerReturn(types []analyser.Type) bool {
	if len(types) > len(x64context.ReturnOrder) {
		return false
	}
	for _, tp := range types {
		classes := classify(tp)
		if len(classes) != 1 || classes[0] != x64context.Int {
			return false
		}
	}
	return true
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
		if isFloat(arg) {
			size := sizeOf(arg)
			t.storeWord(dst.at(8, size), op)
			if size == 4 {
				t.storeWord(dst.at(12, 4), Imm(0))
			}
			t.freeOp(op)
		} else {
			widenOp := t.widen(op, arg, 8)
			t.storeWord(dst.at(8, 8), widenOp)
			t.freeOp(widenOp)
		}

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

func (t *X64Target) compileVariadicCall(sym string, extern bool, receiver ast.Expression, expr *ast.CallExpr, ft *analyser.FuncType, rets []analyser.Type) ([]Operand, error) {
	fixed := len(ft.Params) - 1
	elem := ft.Params[fixed]
	esize := sizeOf(elem)
	extra := len(expr.Args) - fixed

	args := make([]ast.Expression, 0, fixed+1)
	if receiver != nil {
		op, err := t.compileExpr(receiver)
		if err != nil {
			return nil, err
		}
		args = append(args, t.hiddenArg(t.info.Types[receiver], op))
	}

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
			return t.compileCall(sym, extern, args, rets, false, false)
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
		ops, err := t.compileCall(sym, extern, args, rets, false, false)
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
	return t.compileCall(sym, extern, args, rets, false, false)
}

func (t *X64Target) compileCall(sym string, extern bool, args []ast.Expression, rets []analyser.Type, cVariadic, cWin64 bool) ([]Operand, error) {
	pushed := t.ctx.Pushed()

	types := t.valueTypes(args)
	sret := usesSret(rets)
	var argLocs []x64context.ArgLoc
	if cWin64 {
		argLocs = t.assignWin64Args(types)
	} else {
		argLocs = assignArgs(types, sret)
	}

	fast := t.registerOnly(args, types, argLocs, sret, cVariadic)
	held := make([]Operand, len(args))
	pushedOrder := make([]int, 0, len(args))

	regs := t.ctx.AllocatedRegisters()
	for _, reg := range regs {
		t.pushWord(Reg{reg, 8})
	}

	stackBytes := 0
	if cWin64 {
		stackBytes += 32
	}
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
				if fast {
					size := sizeOf(types[i])
					if size == 0 {
						t.freeOp(op)
						return nil
					}
					if _, ok := underlying(types[i]).(analyser.PrimitiveType); ok && size < 4 {
						op = t.widen(op, types[i], 4)
					}
					if t.mustPush(op, args[i+1:]) {
						t.pushValue(op)
						pushedOrder = append(pushedOrder, i)
					} else {
						held[i] = op
					}
					return nil
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

	if fast {
		t.moveArgs(argLocs, held, pushedOrder)
	} else {
		for _, arg := range slices.Backward(argLocs) {
			for _, reg := range slices.Backward(arg.Regs) {
				t.pop(reg)
			}
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
		xi := 0
		for i, arg := range argLocs {
			for _, reg := range arg.Regs {
				if reg.Class() == x64context.SSE {
					if cWin64 {
						r := x64context.Win64ParamOrder[i]
						t.text.printft("movq %s, %s\n", t.ctx.GetRegister(r, 8), t.ctx.GetRegister(reg, 8))
					}
					xi++
				}
			}
		}
		if !cWin64 {
			if xi == 0 {
				t.text.printlnt("xor eax, eax")
			} else {
				t.text.printft("mov eax, %d\n", xi)
			}
		}
	}
	xmms := t.ctx.XmmBindings()
	for _, b := range xmms {
		if _, ok := t.ctx.Get(b.Symbol); ok {
			t.storeWord(t.localMem(b.Symbol), Reg{b.Register, sizeOf(b.Symbol.Type)})
		}
	}
	t.text.printft("call %s\n", sym)
	for _, b := range xmms {
		if _, ok := t.ctx.Get(b.Symbol); ok {
			t.loadWord(b.Register, t.localMem(b.Symbol))
		}
	}
	if t.trace {
		label := fmt.Sprintf(".site_%d", len(t.sites))
		t.text.printf("%s:\n", label)
		text := diag.FrameBlock(t.file, t.sources[t.file], t.line, t.col, t.endLine, t.endCol, t.funcName, t.snippets)
		t.sites = append(t.sites, callSite{label, text})
	}

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

func (t *X64Target) compileIndirectCall(expr *ast.CallExpr) ([]Operand, error) {
	op, err := t.compileExpr(expr.Name)
	if err != nil {
		return nil, err
	}
	off := t.ctx.Reserve(8)
	s := slot(off, 8)
	t.store(op, s)
	t.freeOp(op)
	reg := t.ctx.AllocFreeRegister()
	t.loadWord(reg, s)
	t.checkNotNull(reg)
	t.ctx.FreeRegister(reg)
	return t.compileCall(t.memText(s), false, expr.Args, t.info.CallReturns[expr], false, false)
}

func (t *X64Target) compileCallValues(expr *ast.CallExpr) ([]Operand, error) {
	line, col, endLine, endCol := t.enter(expr)
	defer t.leave(line, col, endLine, endCol)
	if e, ok := t.info.ErrorLiterals[expr]; ok {
		op, err := t.compileErrorLiteral(e)
		if err != nil {
			return nil, err
		}
		return []Operand{op}, nil
	}

	if call, ok := t.info.Inlines[expr]; ok {
		if call.Receiver != nil {
			op, err := t.compileExpr(t.info.MethodCalls[expr].Receiver)
			if err != nil {
				return nil, err
			}
			t.store(op, t.inlineLocal(call.Receiver))
		}
		for i, ai := range call.ArgIndex {
			op, err := t.compileExpr(expr.Args[ai])
			if err != nil {
				return nil, err
			}
			t.store(op, t.inlineLocal(call.Params[i]))
		}
		var result x64context.Register
		if !call.Fallible && len(call.Returns) == 1 && classOf(call.Returns[0]) == x64context.Int {
			result = t.ctx.AllocFreeRegisterAmong(x64context.R8, x64context.R9, x64context.R10, x64context.R11)
		}
		var slots []Operand
		if result != x64context.NoReg {
			slots = []Operand{Reg{result, sizeOf(call.Returns[0])}}
		} else {
			slots = make([]Operand, 0, len(call.Returns))
			for _, r := range call.Returns {
				size := sizeOf(r)
				slots = append(slots, slot(t.ctx.Reserve(size), size))
			}
		}
		if call.Fallible {
			s := slot(t.ctx.Reserve(8), 8)
			t.zero(s)
			slots = append(slots, s)
		}

		funcReturns := t.funcReturns
		fallible := t.fallible
		t.funcReturns = call.Returns
		t.fallible = call.Fallible
		var lastReturn ast.Statement
		l := len(call.Body.Statements)
		if l > 0 && isReturn(call.Body.Statements[l-1]) {
			lastReturn = call.Body.Statements[l-1]
		}
		frame := inlineFrame{call: call, results: slots, label: t.newLabel("inline_end"), deferDepth: len(t.deferScopes), lastReturn: lastReturn}
		t.inlineFrames = append(t.inlineFrames, frame)
		t.collectVariables(call.Body)
		if err := t.compileStatement(call.Body); err != nil {
			return nil, err
		}
		t.inlineFrames = t.inlineFrames[:len(t.inlineFrames)-1]
		t.text.printf("%s:\n", frame.label)
		t.funcReturns = funcReturns
		t.fallible = fallible
		return slots, nil
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
		rets := t.info.CallReturns[expr]
		if mc.Func.Variadic {
			return t.compileVariadicCall(sym, mc.Module != t.module, mc.Receiver, expr, mc.Func, rets)
		}
		args := append([]ast.Expression{mc.Receiver}, expr.Args...)
		return t.compileCall(sym, mc.Module != t.module, args, rets, false, false)
	}

	switch n := expr.Name.(type) {
	case *ast.CallExpr, *ast.IndexExpr:
		return t.compileIndirectCall(expr)
	case *ast.MemberExpr:
		if t.info.Members[n].Kind == analyser.MemberField {
			return t.compileIndirectCall(expr)
		}
	}

	sym, err := t.resolveCallee(expr.Name)
	if err != nil {
		return nil, err
	}
	if _, ok := underlying(sym.Type).(*analyser.FuncType); ok && (sym.Kind == analyser.VAR || sym.Kind == analyser.PARAM) {
		op, err := t.compileExpr(expr.Name)
		if err != nil {
			return nil, err
		}
		reg := t.ctx.AllocFreeRegister()
		t.loadWord(reg, op)
		t.checkNotNull(reg)
		t.ctx.FreeRegister(reg)
		return t.compileCall(t.opText(op, 8), false, expr.Args, t.info.CallReturns[expr], false, false)
	}
	if sym.CName != "" {
		cVariadic := false
		if ft, ok := underlying(sym.Type).(*analyser.FuncType); ok {
			cVariadic = ft.CVariadic
		}
		return t.compileCall(sym.CName, sym.Extern || sym.Module != t.module, expr.Args, t.info.CallReturns[expr], cVariadic, t.abi == "win64")
	}
	if sym.Module == "" {
		return t.compileBuiltin(sym.Name, expr)
	}

	symbol := t.funcSymbol(sym.Module, sym.Name)
	if ft, ok := underlying(sym.Type).(*analyser.FuncType); ok {
		if ft.Variadic {
			return t.compileVariadicCall(symbol, sym.Module != t.module, nil, expr, ft, t.info.CallReturns[expr])
		}
	}

	return t.compileCall(symbol, sym.Module != t.module, expr.Args, t.info.CallReturns[expr], false, false)
}

func (t *X64Target) moveArgs(argLocs []x64context.ArgLoc, held []Operand, order []int) {
	dsts := make([]x64context.Register, len(argLocs))
	for i, loc := range argLocs {
		if len(loc.Regs) == 1 {
			dsts[i] = loc.Regs[0]
		}
	}
	t.moveTo(dsts, held, order)
}

func (t *X64Target) moveTo(dsts []x64context.Register, held []Operand, order []int) {
	target := map[x64context.Register]int{}
	for i, loc := range dsts {
		target[loc] = i
	}

	for i, op := range held {
		if op == nil {
			continue
		}
		for _, reg := range operandRegs(op) {
			if owner, isTarget := target[reg]; isTarget && owner != i {
				t.pushValue(op)
				order = append(order, i)
				held[i] = nil
				break
			}
		}
	}

	for i, op := range held {
		if op == nil {
			continue
		}
		dst := dsts[i]
		if r, same := op.(Reg); !same || r.Reg != dst {
			t.loadWord(dst, op)
		}
		t.freeOp(op)
	}

	for _, i := range slices.Backward(order) {
		t.pop(dsts[i])
	}
}

func operandRegs(op Operand) []x64context.Register {
	switch o := op.(type) {
	case Reg:
		return []x64context.Register{o.Reg}
	case Mem:
		return memRegs(o)
	case Addr:
		return memRegs(o.Of)
	}
	return nil
}

func memRegs(m Mem) []x64context.Register {
	var regs []x64context.Register
	for _, r := range []x64context.Register{m.Base, m.Index} {
		if r != x64context.NoReg && r != x64context.BP && r != x64context.SP {
			regs = append(regs, r)
		}
	}
	return regs
}

func (t *X64Target) registerOnly(args []ast.Expression, types []analyser.Type, argLocs []x64context.ArgLoc, sret, cVariadic bool) bool {
	if sret || cVariadic || len(types) != len(args) {
		return false
	}
	for i, loc := range argLocs {
		if sizeOf(types[i]) == 0 {
			continue
		}
		if len(loc.Regs) != 1 || loc.Regs[0].Class() == x64context.SSE {
			return false
		}
	}
	return true
}

func (t *X64Target) mustPush(op Operand, later []ast.Expression) bool {
	free := len(x64context.RegisterOrder) - len(t.ctx.AllocatedOrderRegisters())
	need := 0
	for _, expr := range later {
		need = max(need, regsNeeded(expr))
	}
	if need > free {
		return true
	}
	if _, isMem := op.(Mem); isMem && hasCall(later) {
		return true
	}
	return false
}

func hasCall(exprs []ast.Expression) bool {
	for _, expr := range exprs {
		switch e := expr.(type) {
		case *ast.CallExpr, *ast.PropagateExpr, *ast.CoalesceExpr, *ast.SwitchExpr:
			return true
		case *ast.BinaryExpr:
			if hasCall([]ast.Expression{e.Left, e.Right}) {
				return true
			}
		case *ast.UnaryExpr:
			return hasCall([]ast.Expression{e.Value})
		case *ast.CastExpr:
			return hasCall([]ast.Expression{e.Value})
		case *ast.ConvExpr:
			return hasCall([]ast.Expression{e.Cast.Value})
		case *ast.DerefExpr:
			return hasCall([]ast.Expression{e.Value})
		case *ast.MemberExpr:
			return hasCall([]ast.Expression{e.Object})
		case *ast.IndexExpr:
			if hasCall([]ast.Expression{e.Array, e.Index}) {
				return true
			}
		case *ast.SliceExpr:
			if hasCall([]ast.Expression{e.Array, e.Start, e.End}) {
				return true
			}
		case *ast.TernaryExpr:
			if hasCall([]ast.Expression{e.Condition, e.Then, e.Else}) {
				return true
			}
		case *ast.InExpr:
			if hasCall([]ast.Expression{e.Left, e.Right}) {
				return true
			}
		case *ast.ArrayLiteral:
			return hasCall(e.Elements)
		case *ast.StructLiteral:
			return hasCall(e.Values)
		}
	}
	return false
}
