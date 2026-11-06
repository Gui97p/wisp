package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileExpr(expr ast.Expression) (Operand, error) {
	line, col, endLine, endCol := t.enter(expr)
	defer t.leave(line, col, endLine, endCol)

	op, err := t.compileExprNode(expr)
	return op, t.locate(expr, err)
}

func (t *X64Target) compileExprNode(expr ast.Expression) (Operand, error) {
	if _, recorded := t.info.Folds[expr]; recorded {
		if value, ok := t.info.Folded(expr); ok {
			if value.Kind == analyser.ConstBool {
				return t.resolveBoolLiteral(value.Bool)
			}
			return t.resolveIntLiteral(value.Bits())
		}
	}
	switch e := expr.(type) {
	case *ast.NullLiteral:
		return Imm(0), nil
	case *ast.IntLiteral:
		tp := t.info.Types[e]
		if isFloat(tp) {
			return t.resolveFloatLiteral(float64(e.Value), sizeOf(tp))
		}
		return t.resolveIntLiteral(e.Value)
	case *ast.BoolLiteral:
		return t.resolveBoolLiteral(e.Value)
	case *ast.CharLiteral:
		return Imm(e.Value), nil
	case *ast.FloatLiteral:
		return t.resolveFloatLiteral(e.Value, sizeOf(t.info.Types[e]))
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
	case *ast.FuncLiteral:
		return t.compileFuncLiteral(e)
	case *ast.IdentLiteral:
		sym, ok := t.info.Idents[e]
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined identifier literal")
		}
		if orig, ok := t.info.Narrowed[sym]; ok {
			off, _ := t.ctx.Get(orig)
			return slot(off, 24).at(8, sizeOf(sym.Type)), nil
		}
		size := sizeOf(t.info.Types[e])

		switch {
		case sym.Global:
			return t.globalMem(sym, size), nil
		case sym.Kind == analyser.CONST:
			return t.constOperand(sym, t.info.Types[e])
		case sym.Kind == analyser.FUNC:
			return Addr{Of: label(t.funcLabel(sym), size)}, nil
		}

		offset, ok := t.ctx.Get(sym)
		if !ok {
			return nil, fmt.Errorf("x86-64: undefined literal")
		}

		return slot(offset, size), nil

	case *ast.BinaryExpr:
		return t.compileBinaryExpr(e)
	case *ast.UnaryExpr:
		return t.compileUnaryExpr(e)
	case *ast.DerefExpr:
		return t.compileDerefExpr(e)
	case *ast.CallExpr:
		return t.compileCallExpr(e)
	case *ast.CastExpr:
		return t.compileCastExpr(e)
	case *ast.ConvExpr:
		return t.compileExpr(e.Cast)
	case *ast.TernaryExpr:
		return t.compileTernaryExpr(e)
	case *ast.MemberExpr:
		if value, ok := t.info.EnumValues[e]; ok {
			return t.resolveIntLiteral(value.Value)
		}
		if info := t.info.Members[e]; info != nil && info.Kind == analyser.MemberModule {
			sym, err := t.resolveCallee(e)
			if err != nil {
				return nil, err
			}
			switch sym.Kind {
			case analyser.CONST:
				return t.constOperand(sym, t.info.Types[e])
			case analyser.FUNC:
				return Addr{Of: label(t.funcLabel(sym), 8)}, nil
			}
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
	case *ast.InExpr:
		return t.compileInExpr(e)
	case *ast.PropagateExpr:
		ops, err := t.compilePropagate(e)
		if err != nil {
			return nil, err
		}

		if len(ops) == 0 {
			return nil, nil
		}

		return ops[0], nil
	case *ast.CoalesceExpr:
		ops, err := t.compileCoalesce(e)
		if err != nil {
			return nil, err
		}

		if len(ops) == 0 {
			return nil, nil
		}

		return ops[0], nil
	case *ast.SwitchExpr:
		ops, err := t.compileSwitchValues(e)
		if err != nil {
			return nil, err
		}

		if len(ops) == 0 {
			return nil, nil
		}

		return ops[0], nil
	default:
		return nil, fmt.Errorf("%s is not supported yet", ast.Describe(expr))
	}
}

func (t *X64Target) funcLabel(sym *analyser.Symbol) string {
	symName := t.funcSymbol(sym.Module, sym.Name)
	if sym.CName != "" {
		symName = sym.CName
	}
	if sym.Module != t.module || sym.Extern {
		t.prelude.define(symName)
	}
	return symName
}

func (t *X64Target) globalMem(sym *analyser.Symbol, size int) Mem {
	l := symbolName(sym.Module, sym.Name)
	if sym.Module != t.module {
		t.prelude.define(l)
	}
	return label(l, size)
}

func (t *X64Target) constOperand(sym *analyser.Symbol, tp analyser.Type) (Operand, error) {
	if sym.Const == nil {
		return nil, fmt.Errorf("x86-64: null constant")
	}
	switch sym.Const.Kind {
	case analyser.ConstInt:
		if isFloat(tp) {
			return t.resolveFloatLiteral(float64(sym.Const.Bits()), sizeOf(tp))
		}
		return t.resolveIntLiteral(sym.Const.Bits())
	case analyser.ConstFloat:
		return t.resolveFloatLiteral(sym.Const.Float, sizeOf(tp))
	case analyser.ConstBool:
		return t.resolveBoolLiteral(sym.Const.Bool)
	case analyser.ConstString:
		return t.resolveStringLiteral(sym.Const.Str)
	default:
		return nil, fmt.Errorf("x86-64: invalid constant type")
	}
}

func (t *X64Target) checkNotNull(reg x64context.Register) {
	ok := t.newLabel("not_null")
	regStr := t.ctx.GetRegister(reg, 8)
	t.text.printft("test %s, %s\n", regStr, regStr)
	t.text.printft("jnz %s\n", ok)
	t.panic("null pointer dereference")
	t.text.printf("%s:\n", ok)
}

func (t *X64Target) compileLValue(expr ast.Expression) (Mem, error) {
	switch e := expr.(type) {
	case *ast.IdentLiteral:
		size := sizeOf(t.info.Types[e])
		sym := t.info.Idents[e]
		if sym.Global {
			return t.globalMem(sym, size), nil
		}
		offset, ok := t.ctx.Get(sym)
		if !ok {
			return Mem{}, fmt.Errorf("x86-64: undeclared variable %s", e.Value)
		}

		return slot(offset, size), nil
	case *ast.MemberExpr:
		if t.info.Members[e].Kind == analyser.MemberModule {
			sym, err := t.resolveCallee(e)
			if err != nil {
				return Mem{}, err
			}
			return t.globalMem(sym, sizeOf(t.info.Types[e])), nil
		}
		op, err := t.compileExpr(e.Object)
		if err != nil {
			return Mem{}, err
		}
		info := t.info.Members[e]
		if info.Kind == analyser.MemberErrorField {
			reg := t.materialize(op, 8)
			t.checkNotNull(reg)
			return deref(reg, errFieldOffset[e.Field], sizeOf(t.info.Types[e])), nil
		}

		var base Mem
		if info.ByPointer {
			reg := t.materialize(op, 8)
			t.checkNotNull(reg)
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
	case *ast.DerefExpr:
		value, err := t.compileExpr(e.Value)
		if err != nil {
			return Mem{}, err
		}

		size := sizeOf(t.info.Types[expr])
		reg := t.materialize(value, size)
		t.checkNotNull(reg)
		return Mem{Base: reg, Size: size}, nil
	default:
		return Mem{}, fmt.Errorf("x86-64: unsupported assignment target %T", expr)
	}
}

func (t *X64Target) errorMember(expr *ast.BinaryExpr) (errExpr, member ast.Expression, enum analyser.NamedType, ok bool) {
	try := func(e, m ast.Expression) bool {
		if _, isErr := t.info.Types[e].(analyser.ErrorType); !isErr {
			return false
		}
		nt, isNamed := t.info.Types[m].(analyser.NamedType)
		if !isNamed || !nt.Enum {
			return false
		}
		errExpr, member, enum = e, m, nt
		return true
	}
	if try(expr.Left, expr.Right) || try(expr.Right, expr.Left) {
		return errExpr, member, enum, true
	}
	return nil, nil, analyser.NamedType{}, false
}

func (t *X64Target) compileBinaryExpr(expr *ast.BinaryExpr) (Operand, error) {
	if tc, ok := t.info.TypeCases[expr]; ok {
		return t.compileTypeCase(expr, tc)
	}
	if isString(t.info.Types[expr.Left]) || isString(t.info.Types[expr.Right]) {
		return t.compileStringBinary(expr)
	}
	if errExpr, member, ev, ok := t.errorMember(expr); ok {
		return t.compileErrorBinary(expr, errExpr, member, ev)
	}
	return t.compileScalarBinary(expr)
}

func (t *X64Target) compileTypeCase(expr *ast.BinaryExpr, tc *analyser.TypeCase) (Operand, error) {
	op, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	sub := op.(Mem)

	reg := t.ctx.AllocFreeRegister()
	regStr := t.ctx.GetRegister(reg, 1)
	t.text.printft("xor %s, %s\n", t.ctx.GetRegister(reg, 4), t.ctx.GetRegister(reg, 4))

	tmp := t.ctx.AllocFreeRegister()
	tmpStr := t.ctx.GetRegister(tmp, 8)
	t.loadWord(tmp, sub.at(0, 8))

	if tc.Group {
		var lo, hi int
		switch tp := tc.Type.(type) {
		case analyser.PrimitiveType:
			if strings.Contains(tp.Name, "uint") {
				lo = analyser.BoxCode("uint")
				hi = analyser.BoxCode("uint64")
			} else {
				lo = analyser.BoxCode("int")
				hi = analyser.BoxCode("int64")
			}
		}
		t.text.printft("sub %s, %d\ncmp %s, %d\n", tmpStr, lo, tmpStr, hi-lo)
		t.text.printft("setbe %s\n", regStr)
	} else {
		layout := analyser.Layout(tc.Type)
		w := t.ctx.AllocFreeRegister()
		var expected Operand

		if layout.Named() {
			mod, key, _ := analyser.DescriptorOf(tc.Type)
			t.loadWord(w, Addr{Of: t.typeLabel(mod, key)})
			expected = Reg{w, 8}
		} else {
			expected = Imm(analyser.BoxCode(layout.Code))
		}
		t.text.printft("cmp %s, %s\n", tmpStr, t.opText(expected, 8))
		t.text.printft("sete %s\n", regStr)

		if layout.Identity != "" && layout.Identity != "void" {
			t.loadWord(tmp, sub.at(16, 8))
			if c := analyser.BoxCode(layout.Identity); c != 0 {
				expected = Imm(c)
			} else {
				mod, key, _ := analyser.DescriptorOf(tc.Type)
				t.loadWord(w, Addr{Of: t.typeLabel(mod, key)})
				expected = Reg{w, 8}
			}
			t.text.printft("cmp %s, %s\n", tmpStr, t.opText(expected, 8))
			tmpStr1 := t.ctx.GetRegister(tmp, 1)
			t.text.printft("sete %s\n", tmpStr1)
			t.text.printft("and %s, %s\n", regStr, tmpStr1)
		}
		t.ctx.FreeRegister(w)
	}

	t.ctx.FreeRegister(tmp)
	t.freeOp(sub)

	return Reg{reg, 1}, nil
}

type signMask struct{ size int }

func (t *X64Target) compileUnaryExpr(expr *ast.UnaryExpr) (Operand, error) {
	if expr.Cast != nil {
		return t.compileExpr(expr.Cast)
	}

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

	tp := t.info.Types[expr]
	size := sizeOf(tp)
	if isFloat(tp) && expr.Operator == "-" {
		reg := t.materializeXmm(value)
		mask := fmt.Sprintf("0x%0*x", size*2, uint64(1)<<(size*8-1))
		l, ok := t.createData(fmt.Sprintf("_fmask_%d", size), signMask{size: size})
		if !ok {
			t.rodata.printlnt("align 16")
			t.rodata.printf("%s:\n", l)
			df := "dq"
			ext := ", 0"
			if size == 4 {
				df = "dd"
				ext = ", 0, 0, 0"
			}
			t.rodata.printft("%s %s%s\n", df, mask, ext)
		}
		t.text.printft("xorps %s, [rel %s]\n", t.ctx.GetRegister(reg, size), l)
		return Reg{reg, size}, nil
	}

	reg := t.materialize(value, size)
	regStr := t.ctx.GetRegister(reg, size)

	switch expr.Operator {
	case "!":
		t.text.printft("test %s, %s\n", regStr, regStr)
		t.text.printft("sete %s\n", regStr)
	default:
		inst, ok := unaryOperators[expr.Operator]
		if !ok {
			return nil, fmt.Errorf("x86-64: unsupported unary operator %s", expr.Operator)
		}
		t.text.printft("%s %s\n", inst, regStr)
	}

	return Reg{reg, size}, nil
}

func (t *X64Target) compileDerefExpr(expr *ast.DerefExpr) (Operand, error) {
	value, err := t.compileExpr(expr.Value)
	if err != nil {
		return nil, err
	}

	size := sizeOf(t.info.Types[expr])
	reg := t.materialize(value, size)
	t.checkNotNull(reg)
	return Mem{Base: reg, Size: size}, nil
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

	if underlying(srcType).Equals(underlying(dstType)) {
		return t.compileExpr(expr.Value)
	}

	srcSize := sizeOf(srcType)
	dstSize := sizeOf(dstType)

	switch {
	case isChar(srcType) && isString(dstType):
		ptr, err := t.baseIndex(expr.Value)
		if err != nil {
			return nil, err
		}

		reg := t.ctx.AllocFreeRegister()
		regStr := t.ctx.GetRegister(reg, 8)
		t.text.printft("lea %s, [rel %s]\n", regStr, t.useChars())
		t.text.printft("add %s, %s\n", t.ctx.GetRegister(ptr.Reg, 8), regStr)
		t.ctx.FreeRegister(reg)

		return Wide{Words: []Operand{Reg{ptr.Reg, 8}, Imm(1)}, Size: 16}, nil
	case isSpan(srcType) && isString(dstType):
		return t.compileExpr(expr.Value)
	case (isString(srcType) || isSpan(srcType)) && isPointer(dstType):
		op, err := t.compileExpr(expr.Value)
		if err != nil {
			return nil, err
		}
		return t.field(op, 0), nil
	case isArray(srcType) && isPointer(dstType):
		op, err := t.compileLValue(expr.Value)
		if err != nil {
			return nil, err
		}
		return Addr{Of: op}, nil
	case isFloat(srcType) && isInteger(dstType):
		op, err := t.compileExpr(expr.Value)
		if err != nil {
			return nil, err
		}
		reg := t.ctx.AllocFreeRegister()
		regStr := t.ctx.GetRegister(reg, 8)
		inst := "sd"
		if isFloat32(srcType) {
			inst = "ss"
		}
		if isUint64(dstType) {
			xmm := t.materializeXmm(op)
			xmmStr := t.ctx.GetRegister(xmm, srcSize)
			tmp := t.ctx.AllocFreeRegister()
			tmpStr := t.ctx.GetRegister(tmp, 8)

			bigLabel := t.newLabel("cast_big")
			doneLabel := t.newLabel("cast_done")

			mem, err := t.resolveFloatLiteral(9223372036854775808.0, srcSize)
			if err != nil {
				return nil, err
			}
			memText := t.opText(mem, srcSize)

			t.text.printft("ucomi%s %s, %s\n", inst, xmmStr, memText)
			t.text.printft("jae %s\n", bigLabel)
			t.text.printft("cvtt%s2si %s, %s\n", inst, regStr, xmmStr)
			t.text.printft("jmp %s\n", doneLabel)
			t.text.printf("%s:\n", bigLabel)
			t.text.printft("sub%s %s, %s\n", inst, xmmStr, memText)
			t.text.printft("cvtt%s2si %s, %s\n", inst, regStr, xmmStr)
			t.text.printft("mov %s, 0x8000000000000000\n", tmpStr)
			t.text.printft("xor %s, %s\n", regStr, tmpStr)
			t.text.printf("%s:\n", doneLabel)

			t.ctx.FreeRegister(xmm)
			t.ctx.FreeRegister(tmp)
		} else {
			t.text.printft("cvtt%s2si %s, %s\n", inst, regStr, t.opText(op, srcSize))
		}
		t.freeOp(op)
		return Reg{reg, dstSize}, nil
	case isInteger(srcType) && isFloat(dstType):
		op, err := t.compileExpr(expr.Value)
		if err != nil {
			return nil, err
		}
		reg := t.materialize(t.widen(op, srcType, 8), 8)
		regStr := t.ctx.GetRegister(reg, 8)
		xmm := t.ctx.AllocFreeXmm()
		xmmStr := t.ctx.GetRegister(xmm, 8)
		inst := "sd"
		if isFloat32(dstType) {
			inst = "ss"
		}
		if isUint64(srcType) {
			bigLabel := t.newLabel("cast_big")
			doneLabel := t.newLabel("cast_done")

			t.text.printft("test %s, %s\n", regStr, regStr)
			t.text.printft("js %s\n", bigLabel)
			t.text.printft("cvtsi2%s %s, %s\n", inst, xmmStr, regStr)
			t.text.printft("jmp %s\n", doneLabel)

			tmp := t.ctx.AllocFreeRegister()
			tmpStr := t.ctx.GetRegister(tmp, 8)

			t.text.printf("%s:\n", bigLabel)
			t.text.printft("mov %s, %s\n", tmpStr, regStr)
			t.text.printft("shr %s, 1\n", tmpStr)
			t.text.printft("and %s, 1\n", t.ctx.GetRegister(reg, 4))
			t.text.printft("or %s, %s\n", tmpStr, regStr)
			t.text.printft("cvtsi2%s %s, %s\n", inst, xmmStr, tmpStr)
			t.text.printft("add%s %s, %s\n", inst, xmmStr, xmmStr)
			t.text.printf("%s:\n", doneLabel)

			t.ctx.FreeRegister(tmp)
		} else {
			t.text.printft("cvtsi2%s %s, %s\n", inst, xmmStr, regStr)
		}
		t.freeOp(op)
		t.ctx.FreeRegister(reg)
		return Reg{xmm, dstSize}, nil
	case isFloat(srcType) && isFloat(dstType):
		if srcSize == dstSize {
			return t.compileExpr(expr.Value)
		}
		op, err := t.compileExpr(expr.Value)
		if err != nil {
			return nil, err
		}
		xmm := t.ctx.AllocFreeXmm()

		inst := "cvtss2sd"
		if dstSize == 4 {
			inst = "cvtsd2ss"
		}
		t.text.printft("%s %s, %s\n", inst, t.ctx.GetRegister(xmm, dstSize), t.opText(op, srcSize))
		t.freeOp(op)
		return Reg{xmm, dstSize}, nil
	default:
		op, err := t.compileExpr(expr.Value)
		if err != nil {
			return nil, err
		}

		reg := t.materialize(t.widen(op, srcType, dstSize), dstSize)
		return Reg{reg, dstSize}, nil
	}
}

func (t *X64Target) compileTernaryExpr(expr *ast.TernaryExpr) (Operand, error) {
	if c, ok := t.info.Folded(expr.Condition); ok && c.Kind == analyser.ConstBool {
		if c.Bool {
			return t.compileExpr(expr.Then)
		}
		return t.compileExpr(expr.Else)
	}

	elseLabel := t.newLabel("ternary_else")
	endLabel := t.newLabel("ternary_end")

	tp := t.info.Types[expr]
	size := sizeOf(tp)

	offset := t.ctx.Reserve(size)
	dest := slot(offset, size)

	if err := t.branch(expr.Condition, elseLabel, false); err != nil {
		return nil, err
	}

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

	if limit != nil {
		t.checkIndexOk(end, limit, "jbe", "slice out of range")
		t.freeOp(limit)
	}
	t.checkIndexOk(start, end, "jbe", "slice out of range")

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

func (t *X64Target) compileInExpr(expr *ast.InExpr) (Operand, error) {
	lTp := t.info.Types[expr.Left]
	rTp := t.info.Types[expr.Right]

	switch l := lTp.(type) {
	case analyser.PrimitiveType:
		if l.Name != "string" {
			break
		}

		b := analyser.PrimitiveType{Name: "bool"}

		switch r := rTp.(type) {
		case analyser.PrimitiveType:
			if r.Name != "string" {
				break
			}

			ops, err := t.compileCall(t.useStrfind(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{b}, false)
			if err != nil {
				return nil, err
			}
			return ops[0], nil
		case analyser.ArrayType:
			st, ok := r.Element.(analyser.PrimitiveType)
			if !ok || st.Name != "string" {
				break
			}

			syn := &ast.SliceExpr{Array: expr.Right}
			t.info.Types[syn] = analyser.SpanType{Element: r.Element}

			ops, err := t.compileCall(t.useStrinstrings(), false, []ast.Expression{expr.Left, syn}, []analyser.Type{b}, false)
			if err != nil {
				return nil, err
			}
			return ops[0], nil
		case analyser.SpanType:
			st, ok := r.Element.(analyser.PrimitiveType)
			if !ok || st.Name != "string" {
				break
			}

			ops, err := t.compileCall(t.useStrinstrings(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{b}, false)
			if err != nil {
				return nil, err
			}
			return ops[0], nil
		}
	}

	op, err := t.compileExpr(expr.Left)
	if err != nil {
		return nil, err
	}

	lSize := sizeOf(lTp)
	if lSize > 8 {
		return nil, fmt.Errorf("x86-64: unsupported in expression")
	}
	l := t.materialize(op, lSize)
	lS := t.ctx.GetRegister(l, lSize)

	base, limit, err := t.baseMem(expr.Right)
	if err != nil {
		return nil, err
	}

	p := t.ctx.AllocFreeRegister()
	pS := t.ctx.GetRegister(p, 8)
	t.loadWord(p, Addr{Of: base})

	n := t.ctx.AllocFreeRegister()
	nS := t.ctx.GetRegister(n, 8)
	t.loadWord(n, limit)

	t.freeWord(base)
	t.freeOp(limit)

	r := t.ctx.AllocFreeRegister()
	rS := t.ctx.GetRegister(r, 1)
	t.text.printft("xor %s, %s\n", rS, rS)

	loop := t.newLabel("in_loop")
	found := t.newLabel("in_found")
	end := t.newLabel("in_end")

	t.text.printf("%s:\n", loop)
	t.text.printft("test %s, %s\n", nS, nS)
	t.text.printft("jz %s\n", end)
	t.text.printft("cmp %s, %s\n", lS, t.memText(deref(p, 0, lSize)))
	t.text.printft("je %s\n", found)
	t.text.printft("add %s, %d\n", pS, lSize)
	t.text.printft("dec %s\n", nS)
	t.text.printft("jmp %s\n", loop)
	t.text.printf("%s:\n", found)
	t.text.printft("mov %s, 1\n", rS)
	t.text.printf("%s:\n", end)

	t.ctx.FreeRegister(l)
	t.ctx.FreeRegister(p)
	t.ctx.FreeRegister(n)

	return Reg{r, 1}, nil
}

func (t *X64Target) returnError(errOp Operand) error {
	if t.hasDefers(0, true) {
		errSlot := slot(t.ctx.Reserve(8), 8)
		t.store(errOp, errSlot)
		t.freeOp(errOp)

		if err := t.withFreeRegisters(func() error {
			return t.emitDefers(0, true, errSlot)
		}); err != nil {
			return err
		}

		errOp = errSlot
	}

	ops := []Operand{}
	for _, tp := range t.funcReturns[:len(t.funcReturns)-1] {
		size := sizeOf(tp)
		if size <= 8 {
			ops = append(ops, Imm(0))
			continue
		}
		m := slot(t.ctx.Reserve(size), size)
		t.zero(m)
		ops = append(ops, m)
	}
	ops = append(ops, errOp)
	t.returnValue(ops, t.funcReturns, t.hidden)
	t.text.printlnt("jmp .return")
	return nil
}

func (t *X64Target) compilePropagate(expr *ast.PropagateExpr) ([]Operand, error) {
	line, col, endLine, endCol := t.enter(expr)
	defer t.leave(line, col, endLine, endCol)
	ops, err := t.compileCallValues(expr.Value.(*ast.CallExpr))
	if err != nil {
		return nil, err
	}
	errOp := ops[len(ops)-1]

	okLabel := t.newLabel("propagate_ok")

	reg := t.materialize(errOp, 8)
	regStr := t.ctx.GetRegister(reg, 8)

	t.text.printft("test %s, %s\n", regStr, regStr)
	t.text.printft("jz %s\n", okLabel)

	if expr.Assert {
		if reg != x64context.DI {
			t.text.printft("mov rdi, %s\n", regStr)
		}
		t.prelude.define("__wisp_error_emit")
		t.text.printlnt("call __wisp_error_emit")
		t.panic("unwrapped error")
	} else {
		if err := t.returnError(Reg{reg, 8}); err != nil {
			return nil, err
		}
	}

	t.text.printf("%s:\n", okLabel)
	t.ctx.FreeRegister(reg)
	return ops[:len(ops)-1], nil
}

func (t *X64Target) compileCoalesce(expr *ast.CoalesceExpr) ([]Operand, error) {
	ops, err := t.compileCallValues(expr.Left.(*ast.CallExpr))
	if err != nil {
		return nil, err
	}

	tps := t.info.Types[expr.Left].(analyser.FallibleType).Values
	slots := make([]Mem, len(tps))
	for i, tp := range tps {
		size := sizeOf(tp)
		slots[i] = slot(t.ctx.Reserve(size), size)
		t.store(ops[i], slots[i])
	}

	endLabel := t.newLabel("coalesce_end")

	errOp := ops[len(tps)]
	reg := t.materialize(errOp, 8)
	regStr := t.ctx.GetRegister(reg, 8)

	if expr.ErrorBind != "" {
		offset := t.ctx.Set(t.info.VarSymbols[expr][0], 8)
		t.store(Reg{reg, 8}, slot(offset, 8))
	}

	t.text.printft("test %s, %s\n", regStr, regStr)
	t.text.printft("jz %s\n", endLabel)
	t.ctx.FreeRegister(reg)

	t.collectVariables(expr.Block)
	t.coalesceFrames = append(t.coalesceFrames, coalesceFrame{slots: slots, end: endLabel, deferDepth: len(t.deferScopes)})
	err = t.compileStatement(expr.Block)
	t.coalesceFrames = t.coalesceFrames[:len(t.coalesceFrames)-1]
	if err != nil {
		return nil, err
	}

	t.text.printf("%s:\n", endLabel)

	res := make([]Operand, len(slots))
	for i, s := range slots {
		res[i] = s
	}
	return res, nil
}

func (t *X64Target) compileSwitchValues(expr *ast.SwitchExpr) ([]Operand, error) {
	tps := t.info.SwitchValues[expr]
	slots := make([]Mem, len(tps))
	for i, tp := range tps {
		size := sizeOf(tp)
		slots[i] = slot(t.ctx.Reserve(size), size)
	}

	for _, stmt := range expr.Prelude {
		t.collectVariables(stmt)
		if err := t.compileStatement(stmt); err != nil {
			return nil, err
		}
	}

	arms := make([]*ast.SwitchArm, 0, len(expr.Arms)+1)
	for i := range expr.Arms {
		arms = append(arms, &expr.Arms[i])
	}
	arms = append(arms, expr.Default)

	endLabel := t.newLabel("switch_end")
	for _, arm := range arms {
		nextLabel := ""
		if arm.Condition != nil {
			nextLabel = t.newLabel("switch_next")
			cond, err := t.compileExpr(arm.Condition)
			if err != nil {
				return nil, err
			}
			size := sizeOf(t.info.Types[arm.Condition])
			reg := t.materialize(cond, size)
			regStr := t.ctx.GetRegister(reg, size)
			t.text.printft("test %s, %s\n", regStr, regStr)
			t.ctx.FreeRegister(reg)
			t.text.printft("jz %s\n", nextLabel)
		}

		if arm.Exit != nil {
			if err := t.compileStatement(arm.Exit); err != nil {
				return nil, err
			}
		} else {
			err := t.eachValue(arm.Values, func(i int, op Operand) error {
				t.store(op, slots[i])
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		if arm.Condition != nil {
			t.text.printft("jmp %s\n", endLabel)
			t.text.printf("%s:\n", nextLabel)
		}
	}
	t.text.printf("%s:\n", endLabel)

	ops := make([]Operand, len(slots))
	for i, s := range slots {
		ops[i] = s
	}
	return ops, nil
}
