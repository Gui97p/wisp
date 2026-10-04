package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) pop(reg x64context.Register) {
	t.text.printft("pop %s\n", t.ctx.GetRegister(reg, 8))
	t.ctx.AddPushed(-1)
}

func (t *X64Target) words(op Operand) []Operand {
	switch o := op.(type) {
	case Wide:
		return o.Words
	case Mem:
		if o.Size <= 8 {
			return []Operand{op}
		} else {
			ops := []Operand{}
			for i := 0; i < (o.Size+7)/8; i++ {
				ops = append(ops, o.at(8*i, min(8, o.Size-8*i)))
			}
			return ops
		}
	default:
		return []Operand{op}
	}
}

func (t *X64Target) store(op Operand, dst Mem) {
	for i, w := range t.words(op) {
		t.storeWord(dst.at(8*i, min(8, dst.Size-8*i)), w)
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
			t.fail("x86-64: no avaiable registers")
			return reg
		}
		t.text.printft("mov %s, %d\n", t.ctx.GetRegister(reg, size), o)
		return reg
	case Reg:
		return o.Reg
	case Mem:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no avaiable registers")
			return reg
		}
		t.loadWord(reg, o)
		t.freeWord(o)
		return reg
	case Addr:
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			t.fail("x86-64: no avaiable registers")
			return reg
		}
		t.loadWord(reg, o)
		return reg
	default:
		t.fail("x86-64: unsupported operand %s", o)
	}
	return x64context.NoReg
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
