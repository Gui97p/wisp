package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) baseMem(expr ast.Expression) (Mem, Operand, error) {
	var base Mem
	var limit Operand

	info := t.info.Types[expr]
	switch tp := info.(type) {
	case analyser.ArrayType:
		mem, err := t.compileLValue(expr)
		if err != nil {
			return Mem{}, nil, err
		}
		base = mem
		limit = Imm(tp.Size)
	case analyser.SpanType:
		op, err := t.compileExpr(expr)
		if err != nil {
			return Mem{}, nil, err
		}
		words := t.words(op)
		reg := t.ctx.AllocFreeRegister()
		t.loadWord(reg, words[0])
		base = Mem{Base: reg}
		limit = words[1]
	case analyser.PrimitiveType:
		if tp.Name != "string" {
			return Mem{}, nil, fmt.Errorf("x86-64: unsupported index expression")
		}
		op, err := t.compileExpr(expr)
		if err != nil {
			return Mem{}, nil, err
		}
		words := t.words(op)
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			return Mem{}, nil, fmt.Errorf("x86-64: no available registers")
		}
		t.loadWord(reg, words[0])
		base = Mem{Base: reg}
		limit = words[1]
	default:
		return Mem{}, nil, fmt.Errorf("x86-64: unsupported index expression")
	}

	return base, limit, nil
}

func (t *X64Target) baseIndex(expr ast.Expression) (Reg, error) {
	index, err := t.compileExpr(expr)
	if err != nil {
		return Reg{}, err
	}
	indexTp := t.info.Types[expr]
	indexSize := sizeOf(indexTp)
	reg := t.materialize(index, indexSize)
	dstSize := 8
	indexReg := t.ctx.GetRegister(reg, dstSize)
	if indexSize < 8 {
		inst := ucastOperators[indexSize][dstSize]
		if isSigned(indexTp) {
			inst = castOperators[indexSize][dstSize]
		} else if indexSize == 4 {
			dstSize = 4
		}

		t.text.printft("%s %s, %s\n", inst, indexReg, t.ctx.GetRegister(reg, indexSize))
	}

	return Reg{reg, dstSize}, nil
}

func (t *X64Target) checkIndexOk(reg Reg, limit Operand, inst, message string) {
	okLabel := t.newLabel("index_ok")
	t.text.printft("cmp %s, %s\n", t.ctx.GetRegister(reg.Reg, reg.Size), t.opText(limit, reg.Size))
	t.text.printft("%s %s\n", inst, okLabel)
	t.panic(message)
	t.text.printf("%s:\n", okLabel)
}

func (t *X64Target) elementAt(base Mem, index Reg, elemSize int) (Mem, error) {
	scale := elemSize
	if !slices.Contains([]int{1, 2, 4, 8}, elemSize) {
		indexReg := t.ctx.GetRegister(index.Reg, index.Size)
		t.text.printft("imul %s, %s, %d\n", indexReg, indexReg, elemSize)
		scale = 1
	}

	return indexed(base.Base, index.Reg, scale, base.Disp, elemSize)
}

func (t *X64Target) compileIndex(expr *ast.IndexExpr) (Mem, error) {
	base, limit, err := t.baseMem(expr.Array)
	if err != nil {
		return Mem{}, err
	}

	info := t.info.Types[expr.Array]
	ar, isArray := info.(analyser.ArrayType)
	if intIndex, ok := expr.Index.(*ast.IntLiteral); ok && isArray {
		size := sizeOf(ar.Element)
		mem := base.at(int(intIndex.Value)*size, size)
		return mem, nil
	}

	elemSize := sizeOf(t.info.Types[expr])

	index, err := t.baseIndex(expr.Index)
	if err != nil {
		return Mem{}, err
	}

	t.checkIndexOk(index, limit, "jb", "index out of range")
	t.freeOp(limit)
	return t.elementAt(base, index, elemSize)
}
