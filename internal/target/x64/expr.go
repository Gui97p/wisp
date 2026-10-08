package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileExpr(expr ast.Expression) (Operand, error) {
	switch e := expr.(type) {
	case *ast.NullLiteral:
		return Imm(0), nil
	case *ast.IntLiteral:
		return t.resolveIntLiteral(e.Value)
	case *ast.BoolLiteral:
		return t.resolveBoolLiteral(e.Value)
	case *ast.CharLiteral:
		return Imm(e.Value), nil
	case *ast.StringLiteral:
		return t.resolveStringLiteral(e.Value)
	case *ast.StructLiteral, *ast.ArrayLiteral:
		tp := t.info.Types[e]
		size := sizeOf(tp)
		offset := t.ctx.Reserve(size)

		memSlot := slot(offset, size)
		if err := t.compileInto(e, memSlot, tp); err != nil {
			return nil, err
		}
		return memSlot, nil
	case *ast.IdentLiteral:
		sym, ok := t.info.Idents[e]
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined identifier literal")
		}
		if sym.Kind == analyser.CONST {
			if sym.Const == nil {
				return nil, fmt.Errorf("x86-64: null constant")
			}
			switch sym.Const.Kind {
			case analyser.ConstInt:
				return t.resolveIntLiteral(sym.Const.Bits())
			case analyser.ConstBool:
				return t.resolveBoolLiteral(sym.Const.Bool)
			case analyser.ConstString:
				return t.resolveStringLiteral(sym.Const.Str)
			default:
				return nil, fmt.Errorf("x86-64: invalid constant type")
			}
		}

		offset, ok := t.ctx.Get(sym)
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined literal")
		}
		size := sizeOf(t.info.Types[e])

		return slot(offset, size), nil

	case *ast.BinaryExpr:
		return t.compileBinaryExpr(e)
	case *ast.UnaryExpr:
		return t.compileUnaryExpr(e)
	case *ast.CallExpr:
		return t.compileCallExpr(e)
	case *ast.CastExpr:
		return t.compileCastExpr(e)
	case *ast.TernaryExpr:
		return t.compileTernaryExpr(e)
	case *ast.MemberExpr:
		if value, ok := t.info.EnumValues[e]; ok {
			return t.resolveIntLiteral(value.Value)
		}
		return t.compileLValue(e)
	case *ast.DotIdent:
		value, ok := t.info.EnumValues[e]
		if !ok {
			return nil, fmt.Errorf("x86-64: unresolved enum member .%s", e.Name)
		}
		return t.resolveIntLiteral(value.Value)
	case *ast.IndexExpr:
		return t.compileLValue(e)
	case *ast.SliceExpr:
		return t.compileSliceExpr(e)
	default:
		return nil, fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}

func (t *X64Target) compileLValue(expr ast.Expression) (Mem, error) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		offset, ok := t.ctx.Get(t.info.Idents[e])
		if !ok {
			return Mem{}, fmt.Errorf("x86-64: undeclared variable %s", e.Value)
		}
		size := sizeOf(t.info.Types[e])
		return slot(offset, size), nil
	case *ast.MemberExpr:
		op, err := t.compileExpr(e.Object)
		if err != nil {
			return Mem{}, err
		}
		info := t.info.Members[e]

		var base Mem
		if info.ByPointer {
			reg := t.materialize(op, 8)
			base = deref(reg, 0, sizeOf(info.Struct))
		} else {
			memSlot, ok := op.(Mem)
			if !ok {
				return Mem{}, fmt.Errorf("x86-64: operator is not a Mem")
			}
			base = memSlot
		}

		f := layoutOf(info.Struct).Fields[e.Field]
		return base.at(f.Offset, f.Size), nil
	case *ast.IndexExpr:
		return t.compileIndex(e)
	case *ast.UnaryExpr:
		if e.Operator != "*" {
			return Mem{}, fmt.Errorf("x86-64: unsupported assignment target %T", expr)
		}
		value, err := t.compileExpr(e.Value)
		if err != nil {
			return Mem{}, err
		}

		size := sizeOf(t.info.Types[expr])
		reg := t.materialize(value, size)
		return Mem{Base: reg, Size: size}, nil
	default:
		return Mem{}, fmt.Errorf("x86-64: unsupported assignment target %T", expr)
	}
}

func (t *X64Target) compileBinaryExpr(expr *ast.BinaryExpr) (Operand, error) {
	if p, ok := underlying(t.info.Types[expr.Left]).(analyser.PrimitiveType); ok && p.Name == "string" {
		t.useStreq()
		ops, err := t.compileCall("__wisp_streq", false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{analyser.PrimitiveType{Name: "bool"}})
		if err != nil {
			return nil, err
		}
		switch expr.Operator {
		case "==":
		case "!=":
			t.text.printft("xor %s, 1\n", t.opText(ops[0], 8))
		default:
			return nil, fmt.Errorf("x86-64: string ordering not supported")
		}
		return ops[0], nil
	}

	left, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}

	var right Operand
	if expr.Operator != "&&" && expr.Operator != "||" {
		if regsNeeded(expr.Right) > len(t.ctx.AllocatedOrderRegisters()) {
			left = t.spill(left)
		}

		right, err = t.compileExpr(expr.Right)
		if err != nil {
			return nil, err
		}
	}

	return t.emitBinary(expr.Operator, left, right, t.info.Types[expr.Left], t.info.Types[expr.Right], t.info.Types[expr], expr.Right)
}

func (t *X64Target) compileUnaryExpr(expr *ast.UnaryExpr) (Operand, error) {
	switch expr.Operator {
	case "&":
		mem, err := t.compileLValue(expr.Value)
		if err != nil {
			return nil, err
		}
		return Addr{Of: mem}, nil
	}

	value, err := t.compileExpr(expr.Value)
	if err != nil {
		return nil, err
	}

	size := sizeOf(t.info.Types[expr])
	reg := t.materialize(value, size)
	regStr := t.ctx.GetRegister(reg, size)

	switch expr.Operator {
	case "!":
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("sete %s\n", regStr)
	case "*":
		return Mem{Base: reg, Size: size}, nil
	default:
		inst, ok := unaryOperators[expr.Operator]
		if !ok {
			return nil, fmt.Errorf("x86-64: unsupported unary operator %s", expr.Operator)
		}
		t.text.printft("%s %s\n", inst, regStr)
	}

	return Reg{reg, size}, nil
}

func (t *X64Target) resolveCallee(expr ast.Expression) (*analyser.Symbol, error) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		return t.info.Idents[e], nil
	case *ast.MemberExpr:
		c, err := t.resolveCallee(e.Object)
		if err != nil {
			return nil, err
		}
		tp, ok := c.Type.(*analyser.ModuleType)
		if !ok {
			return c, nil
		}
		return tp.Exports[e.Field], nil
	default:
		return nil, fmt.Errorf("x86-64: invalid callee %s", e)
	}
}

func (t *X64Target) compileCallExpr(expr *ast.CallExpr) (Operand, error) {
	ops, err := t.compileCallValues(expr)
	if err != nil {
		return nil, err
	}

	if len(ops) == 1 {
		return ops[0], nil
	}

	for _, op := range ops {
		t.freeOp(op)
	}
	return nil, nil
}

func (t *X64Target) compileCastExpr(expr *ast.CastExpr) (Operand, error) {
	srcType := t.info.Types[expr.Value]
	dstType := t.info.Types[expr]

	if underlying(srcType) == underlying(dstType) {
		return t.compileExpr(expr.Value)
	}

	srcSize := sizeOf(srcType)
	dstSize := sizeOf(dstType)

	op, err := t.compileExpr(expr.Value)
	if err != nil {
		return nil, err
	}

	if dstSize <= srcSize {
		reg := t.materialize(op, dstSize)
		return Reg{reg, dstSize}, nil
	}

	reg := t.ctx.AllocFreeRegister()
	srcReg := t.materialize(op, srcSize)

	inst := ucastOperators[srcSize][dstSize]
	if isSigned(srcType) {
		inst = castOperators[srcSize][dstSize]
	} else if dstSize == 8 && srcSize == 4 {
		dstSize = 4
	}

	t.text.printft("%s %s, %s\n", inst, t.ctx.GetRegister(reg, dstSize), t.ctx.GetRegister(srcReg, srcSize))
	t.freeOp(op)
	t.ctx.FreeRegister(srcReg)

	return Reg{reg, dstSize}, nil
}

func (t *X64Target) compileTernaryExpr(expr *ast.TernaryExpr) (Operand, error) {
	elseLabel := t.newLabel("ternary_else")
	endLabel := t.newLabel("ternary_end")

	tp := t.info.Types[expr]
	size := sizeOf(tp)

	offset := t.ctx.Reserve(size)
	dest := slot(offset, size)

	cond, err := t.compileExpr(expr.Condition)
	if err != nil {
		return nil, err
	}

	condSize := sizeOf(t.info.Types[expr.Condition])
	reg := t.materialize(cond, condSize)
	regStr := t.ctx.GetRegister(reg, condSize)
	t.text.printft("test %s, %s\n", regStr, regStr)
	t.ctx.FreeRegister(reg)
	t.text.printft("jz %s\n", elseLabel)

	thenOp, err := t.compileExpr(expr.Then)
	if err != nil {
		return nil, err
	}
	t.store(thenOp, dest)
	t.text.printft("jmp %s\n", endLabel)

	t.text.printf("%s:\n", elseLabel)
	elseOp, err := t.compileExpr(expr.Else)
	if err != nil {
		return nil, err
	}
	t.store(elseOp, dest)

	t.text.printf("%s:\n", endLabel)
	return dest, nil
}

func (t *X64Target) compileSliceExpr(expr *ast.SliceExpr) (Operand, error) {
	base, limit, err := t.baseMem(expr.Array)
	if err != nil {
		return nil, err
	}

	var start Reg
	if expr.Start != nil {
		start, err = t.baseIndex(expr.Start)
		if err != nil {
			return nil, err
		}
	} else {
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			return nil, fmt.Errorf("x86-64: no available registers")
		}
		reg4 := t.ctx.GetRegister(reg, 4)
		t.text.printft("xor %s, %s\n", reg4, reg4)
		start = Reg{reg, 8}
	}

	var end Reg
	if expr.End != nil {
		end, err = t.baseIndex(expr.End)
		if err != nil {
			return nil, err
		}
	} else {
		reg := t.ctx.AllocFreeRegister()
		if reg == x64context.NoReg {
			return nil, fmt.Errorf("x86-64: no available registers")
		}
		t.text.printft("mov %s, %s\n", t.ctx.GetRegister(reg, 8), t.opText(limit, 8))
		end = Reg{reg, 8}
	}

	t.checkIndexOk(end, limit, "jbe", "slice out of range")
	t.checkIndexOk(start, end, "jbe", "slice out of range")
	t.freeOp(limit)

	elemSize := 1
	switch tp := t.info.Types[expr].(type) {
	case analyser.SpanType:
		elemSize = sizeOf(tp.Element)
	case analyser.PrimitiveType:
		if tp.Name != "string" {
			return nil, fmt.Errorf("x86-64: unsupported slice expression")
		}
	default:
		return nil, fmt.Errorf("x86-64: unsupported slice expression")
	}
	t.text.printft("sub %s, %s\n", t.opText(end, end.Size), t.opText(start, start.Size))
	mem, err := t.elementAt(base, start, elemSize)
	if err != nil {
		return nil, err
	}

	return Wide{Words: []Operand{Addr{Of: mem}, Reg{end.Reg, 8}}, Size: 16}, nil
}
