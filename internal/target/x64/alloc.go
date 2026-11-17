package x64

import (
	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileAllocation(expr *ast.CallExpr) ([]Operand, error) {
	if resize, isResize := t.info.Resizes[expr]; isResize {
		return t.compileResize(expr, resize)
	}
	make, isMake := t.info.Makes[expr]
	if !isMake {
		return t.compileCallValues(expr.Lowered)
	}

	intType := analyser.PrimitiveType{Name: "int"}
	count, err := t.compileExpr(make.Count)
	if err != nil {
		return nil, err
	}
	held := t.hiddenArg(intType, count)

	bytes := &ast.BinaryExpr{Left: held, Operator: "*", Right: make.Size}
	t.info.Types[bytes] = intType

	original := expr.Lowered.Args[0]
	expr.Lowered.Args[0] = bytes
	ops, err := t.compileCallValues(expr.Lowered)
	expr.Lowered.Args[0] = original
	if err != nil {
		return nil, err
	}

	return t.sliceResult(ops, held)
}

func (t *X64Target) sliceResult(ops []Operand, counted *ast.IdentLiteral) ([]Operand, error) {
	failure := slot(t.ctx.Reserve(8), 8)
	t.store(ops[1], failure)
	length := slot(t.ctx.Reserve(8), 8)
	count, err := t.compileExpr(counted)
	if err != nil {
		return nil, err
	}
	t.store(count, length)

	kept := t.newLabel("slice_ok")
	t.text.printft("cmp %s, 0\n", t.opText(failure, 8))
	t.text.printft("je %s\n", kept)
	t.zero(length)
	t.text.printf("%s:\n", kept)
	return []Operand{Wide{Words: []Operand{ops[0], length}, Size: 16}, failure}, nil
}

func (t *X64Target) compileResize(expr *ast.CallExpr, resize *analyser.ResizeInfo) ([]Operand, error) {
	intType := analyser.PrimitiveType{Name: "int"}
	pointerType := analyser.PointerType{Element: analyser.VoidType{}}
	size := sizeOf(resize.Elem)

	sliceOp, err := t.compileExpr(resize.Slice)
	if err != nil {
		return nil, err
	}
	held := t.hiddenArg(t.info.Types[resize.Slice], sliceOp)
	heldMem, err := t.compileExpr(held)
	if err != nil {
		return nil, err
	}
	mem := heldMem.(Mem)
	pointer := t.hiddenArg(pointerType, mem.at(0, 8))

	oldBytes, err := t.emitBinary("*", mem.at(8, 8), Imm(size), intType, intType, intType, nil)
	if err != nil {
		return nil, err
	}
	old := t.hiddenArg(intType, oldBytes)

	countOp, err := t.compileExpr(resize.Count)
	if err != nil {
		return nil, err
	}
	counted := t.hiddenArg(intType, countOp)
	countMem, err := t.compileExpr(counted)
	if err != nil {
		return nil, err
	}
	newBytes, err := t.emitBinary("*", countMem, Imm(size), intType, intType, intType, nil)
	if err != nil {
		return nil, err
	}
	next := t.hiddenArg(intType, newBytes)

	original := expr.Lowered.Args
	expr.Lowered.Args = []ast.Expression{pointer, old, next}
	ops, err := t.compileCallValues(expr.Lowered)
	expr.Lowered.Args = original
	if err != nil {
		return nil, err
	}
	return t.sliceResult(ops, counted)
}
