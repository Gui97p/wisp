package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

const sumTagField = "$tag"

type optionalParts struct {
	value   Operand
	present Mem
	pointer bool
	element analyser.Type
}

func (t *X64Target) optionalOperand(expr ast.Expression) (optionalParts, error) {
	tp := t.info.Types[expr]
	element, ok := analyser.OptionalElement(tp)
	if !ok {
		return optionalParts{}, fmt.Errorf("x86-64: %s is not an optional", tp)
	}

	op, err := t.compileExpr(expr)
	if err != nil {
		return optionalParts{}, err
	}

	st, isStruct := analyser.OptionalStruct(tp)
	if !isStruct {
		return optionalParts{value: op, pointer: true, element: element}, nil
	}

	mem, ok := op.(Mem)
	if !ok {
		size := sizeOf(tp)
		mem = slot(t.ctx.Reserve(size), size)
		t.store(op, mem)
	}
	valueName, presentName := analyser.OptionalFields()
	layout := layoutOf(st)
	value := layout.Fields[valueName]
	present := layout.Fields[presentName]
	return optionalParts{
		value:   mem.at(value.Offset, value.Size),
		present: mem.at(present.Offset, present.Size),
		element: element,
	}, nil
}

func (t *X64Target) testPresent(parts optionalParts) (x64context.Register, string, int) {
	if parts.pointer {
		reg := t.materialize(parts.value, 8)
		return reg, t.ctx.GetRegister(reg, 8), 8
	}
	reg := t.ctx.AllocFreeRegister()
	if reg == x64context.NoReg {
		t.fail("x86-64: no available registers")
		return reg, "", 1
	}
	t.loadWord(reg, parts.present)
	return reg, t.ctx.GetRegister(reg, 1), 1
}

func (t *X64Target) compileOptionalUnwrap(expr *ast.PropagateExpr) ([]Operand, error) {
	line, col, endLine, endCol := t.enter(expr)
	defer t.leave(line, col, endLine, endCol)

	parts, err := t.optionalOperand(expr.Value)
	if err != nil {
		return nil, err
	}

	if parts.pointer {
		reg := t.materialize(parts.value, 8)
		if !expr.Unchecked {
			regStr := t.ctx.GetRegister(reg, 8)
			okLabel := t.newLabel("unwrap_ok")
			t.text.printft("test %s, %s\n", regStr, regStr)
			t.text.printft("jnz %s\n", okLabel)
			t.panic("unwrapped null")
			t.text.printf("%s:\n", okLabel)
		}
		return []Operand{Reg{reg, 8}}, nil
	}

	if !expr.Unchecked {
		reg, regStr, _ := t.testPresent(parts)
		okLabel := t.newLabel("unwrap_ok")
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("jnz %s\n", okLabel)
		t.panic("unwrapped null")
		t.text.printf("%s:\n", okLabel)
		t.ctx.FreeRegister(reg)
	}
	return []Operand{parts.value}, nil
}

func (t *X64Target) compileOptionalCoalesce(expr *ast.CoalesceExpr) ([]Operand, error) {
	parts, err := t.optionalOperand(expr.Left)
	if err != nil {
		return nil, err
	}

	size := sizeOf(parts.element)
	result := slot(t.ctx.Reserve(size), size)
	endLabel := t.newLabel("coalesce_end")

	if parts.pointer {
		reg := t.materialize(parts.value, 8)
		t.store(Reg{reg, 8}, result)
		regStr := t.ctx.GetRegister(reg, 8)
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("jnz %s\n", endLabel)
		t.ctx.FreeRegister(reg)
	} else {
		t.store(parts.value, result)
		reg, regStr, _ := t.testPresent(parts)
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("jnz %s\n", endLabel)
		t.ctx.FreeRegister(reg)
	}

	var lastYield ast.Statement
	if expr.Block != nil {
		block := expr.Block.(*ast.BlockStmt)
		l := len(block.Statements)
		if l > 0 && isYield(block.Statements[l-1]) {
			lastYield = block.Statements[l-1]
		}
	}

	t.collectVariables(expr.Block)
	t.coalesceFrames = append(t.coalesceFrames, coalesceFrame{slots: []Mem{result}, end: endLabel, deferDepth: len(t.deferScopes), lastYield: lastYield})
	err = t.compileStatement(expr.Block)
	t.coalesceFrames = t.coalesceFrames[:len(t.coalesceFrames)-1]
	if err != nil {
		return nil, err
	}

	t.text.printf("%s:\n", endLabel)
	return []Operand{result}, nil
}

func (t *X64Target) boxOptional(dst Mem, tp, element analyser.Type, op Operand) {
	absent := t.newLabel("box_absent")
	end := t.newLabel("box_end")

	if st, isStruct := analyser.OptionalStruct(tp); isStruct {
		mem, ok := op.(Mem)
		if !ok {
			size := sizeOf(tp)
			mem = slot(t.ctx.Reserve(size), size)
			t.store(op, mem)
		}
		valueName, presentName := analyser.OptionalFields()
		layout := layoutOf(st)
		value := layout.Fields[valueName]
		present := layout.Fields[presentName]

		reg := t.materialize(mem.at(present.Offset, present.Size), 1)
		regStr := t.ctx.GetRegister(reg, 1)
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("jz %s\n", absent)
		t.ctx.FreeRegister(reg)
		t.boxArg(dst, element, mem.at(value.Offset, value.Size))
	} else {
		reg := t.materialize(op, 8)
		regStr := t.ctx.GetRegister(reg, 8)
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("jz %s\n", absent)
		t.boxArg(dst, element, Reg{reg, 8})
	}

	t.text.printft("jmp %s\n", end)
	t.text.printf("%s:\n", absent)
	t.storeWord(dst, Imm(analyser.BoxCode("null")))
	t.storeWord(dst.at(8, 8), Imm(0))
	t.storeWord(dst.at(16, 8), Imm(0))
	t.text.printf("%s:\n", end)
}

func (t *X64Target) compileBlockExpr(expr *ast.BlockExpr) (Operand, error) {
	tp := t.info.Types[expr]
	size := sizeOf(tp)
	result := slot(t.ctx.Reserve(size), size)
	endLabel := t.newLabel("block_end")

	var lastYield ast.Statement
	if l := len(expr.Block.Statements); l > 0 && isYield(expr.Block.Statements[l-1]) {
		lastYield = expr.Block.Statements[l-1]
	}

	t.collectVariables(expr.Block)
	t.coalesceFrames = append(t.coalesceFrames, coalesceFrame{slots: []Mem{result}, end: endLabel, deferDepth: len(t.deferScopes), lastYield: lastYield, label: expr.Label})
	err := t.compileStatement(expr.Block)
	t.coalesceFrames = t.coalesceFrames[:len(t.coalesceFrames)-1]
	if err != nil {
		return nil, err
	}

	t.text.printf("%s:\n", endLabel)
	return result, nil
}

func (t *X64Target) compileVariantValue(variant *analyser.VariantInfo) Operand {
	size := sizeOf(variant.Sum)
	dst := slot(t.ctx.Reserve(size), size)
	t.storeVariantTag(dst, variant)
	return dst
}

func (t *X64Target) storeVariantTag(dst Mem, variant *analyser.VariantInfo) {
	layout := layoutOf(variant.Sum)
	t.zero(dst)
	tag := layout.Fields[sumTagField]
	t.storeWord(dst.at(tag.Offset, tag.Size), Imm(variant.Variant.Tag))
}

func (t *X64Target) compileVariant(lit *ast.StructLiteral, dst Mem, variant *analyser.VariantInfo) error {
	t.storeVariantTag(dst, variant)
	if variant.Variant.Payload == nil {
		return nil
	}

	layout := layoutOf(variant.Sum)
	payload := layout.Fields[variant.Variant.Field]
	region := dst.at(payload.Offset, payload.Size)
	inner := layoutOf(variant.Variant.Payload)
	for _, field := range variant.Variant.Payload.Order {
		f := inner.Fields[field]
		mem := region.at(f.Offset, f.Size)
		index := slices.Index(lit.Keys, field)
		if index == -1 {
			t.zero(mem)
			continue
		}
		if err := t.compileInto(lit.Values[index], mem, f.Type); err != nil {
			return err
		}
	}
	return nil
}
