package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) pop(reg x64context.Register) {
	t.text.printft("pop %s\n", t.ctx.GetRegister(reg, 8))
	t.ctx.AddPushed(-1)
}

func (t *X64Target) push(reg x64context.Register) {
	t.text.printft("push %s\n", t.ctx.GetRegister(reg, 8))
	t.ctx.AddPushed(1)
}

func (t *X64Target) panic(message string) {
	t.prelude.define("__wisp_panic")
	label, ok := t.createData("str", message)
	if !ok {
		t.rodata.printf("%s db %s\n", label, bytesToAsm(append([]byte(message), 0)))
	}
	t.text.printft("lea rdi, [rel %s]\n", label)
	t.text.printft("mov esi, %d\n", len(message))
	t.text.printlnt("call __wisp_panic")
}

func (t *X64Target) pieces(size int) []int {
	var pieces []int

	for size > 0 {
		piece := 1
		for piece*2 <= size && piece < 8 {
			piece *= 2
		}

		pieces = append(pieces, piece)
		size -= piece
	}

	return pieces
}

func (t *X64Target) zero(dst Mem) {
	off := 0
	for _, p := range t.pieces(dst.Size) {
		t.store(Imm(0), dst.at(off, p))
		off += p
	}
}

func (t *X64Target) words(op Operand) []Operand {
	switch o := op.(type) {
	case Wide:
		return o.Words
	case Mem:
		ops := []Operand{}
		off := 0
		for _, p := range t.pieces(o.Size) {
			ops = append(ops, o.at(off, p))
			off += p
		}
		return ops
	default:
		return []Operand{op}
	}
}

func (t *X64Target) store(op Operand, dst Mem) {
	off := 0
	for _, w := range t.words(op) {
		size := min(8, dst.Size-off)
		switch wo := w.(type) {
		case Reg:
			size = wo.Size
		case Mem:
			size = wo.Size
		}
		size = min(size, dst.Size-off)
		t.storeWord(dst.at(off, size), w)
		off += size
		t.freeWord(w)
	}
}

func (t *X64Target) pushValue(op Operand) int {
	ws := t.words(op)
	for _, w := range ws {
		t.pushWord(w)
		t.freeWord(w)
	}
	return len(ws)
}

func (t *X64Target) field(op Operand, i int) Operand {
	var r Operand
	for j, w := range t.words(op) {
		if i == j {
			r = w
			continue
		}
		t.freeWord(w)
	}
	if r == nil {
		t.fail("x86-64: operand index out of range")
	}
	return r
}

func (t *X64Target) freeOp(op Operand) {
	for _, w := range t.words(op) {
		t.freeWord(w)
	}
}

func (t *X64Target) materialize(op Operand, size int) x64context.Register {
	switch o := op.(type) {
	case Imm:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return reg
		}
		t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, size), o)
		return reg
	case Reg:
		return o.Reg
	case Mem:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return reg
		}
		t.loadWord(reg, o)
		t.freeWord(o)
		return reg
	case Addr:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no available registers")
			return reg
		}
		t.loadWord(reg, o)
		return reg
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
	return x64context.NoReg
}

func (t *X64Target) compileInto(expr ast.Expression, dst Mem, tp analyser.Type) error {
	switch e := expr.(type) {
	case *ast.StructLiteral:
		st := structOf(tp)
		layout := layoutOf(st)
		for _, field := range st.Order {
			f := layout.Fields[field]
			mem := dst.at(f.Offset, f.Size)
			index := slices.Index(e.Keys, field)
			if index != -1 {
				if err := t.compileInto(e.Values[index], mem, f.Type); err != nil {
					return err
				}
			} else {
				t.zero(mem)
			}
		}
	case *ast.ArrayLiteral:
		ar, ok := arrayOf(tp)
		if !ok {
			return fmt.Errorf("x86-64: not an array")
		}
		size := sizeOf(ar.Element)
		for n := range ar.Size {
			mem := dst.at(int(n)*size, size)
			if int(n) < len(e.Elements) {
				if err := t.compileInto(e.Elements[n], mem, ar.Element); err != nil {
					return err
				}
			} else {
				t.zero(mem)
			}
		}
	default:
		op, err := t.compileExpr(expr)
		if err != nil {
			return err
		}
		t.store(op, dst)
	}
	return nil
}

func (t *X64Target) opText(op Operand, size int) string {
	switch o := op.(type) {
	case Imm:
		return fmt.Sprint(o)
	case Reg:
		return t.ctx.GetRegister(o.Reg, size)
	case Mem:
		return t.memText(o)
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
	return ""
}

func (t *X64Target) spill(op Operand) Operand {
	switch o := op.(type) {
	case Imm:
		return op
	case Mem:
		if slices.Contains(x64context.RegisterOrder, o.Base) || slices.Contains(x64context.RegisterOrder, o.Index) {
			offset := t.ctx.Reserve(o.Size)
			memSlot := slot(offset, o.Size)
			t.store(op, memSlot)
			return memSlot
		}
		return op
	case Reg:
		offset := t.ctx.Reserve(o.Size)
		memSlot := slot(offset, o.Size)
		t.store(op, memSlot)
		return memSlot
	case Addr:
		memSlot := slot(t.ctx.Reserve(8), 8)
		t.store(op, memSlot)
		return memSlot
	case Wide:
		offset := t.ctx.Reserve(o.Size)
		memSlot := slot(offset, o.Size)
		t.store(op, memSlot)
		return memSlot
	}
	t.fail("spill failed: invalid operand %s", op)
	return nil
}

func (t *X64Target) valueTypes(exprs []ast.Expression) []analyser.Type {
	types := []analyser.Type{}

	for _, expr := range exprs {
		switch e := expr.(type) {
		case *ast.CallExpr:
			types = append(types, t.info.CallReturns[e]...)
		default:
			types = append(types, t.info.Types[e])
		}
	}

	return types
}

func (t *X64Target) eachValue(exprs []ast.Expression, callback func(i int, op Operand) error) error {
	i := 0
	for _, expr := range exprs {
		var ops []Operand
		var err error
		switch e := expr.(type) {
		case *ast.CallExpr:
			ops, err = t.compileCallValues(e)
			if err != nil {
				return err
			}
		default:
			op, err := t.compileExpr(e)
			if err != nil {
				return err
			}
			ops = []Operand{op}
		}

		for _, op := range ops {
			if err := callback(i, op); err != nil {
				return err
			}
			i++
		}
	}

	return nil
}
