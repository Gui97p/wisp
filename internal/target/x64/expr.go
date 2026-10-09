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
		if info.Kind == analyser.MemberErrorField {
			reg := t.materialize(op, 8)
			return deref(reg, errFieldOffset[e.Field], sizeOf(t.info.Types[e])), nil
		}

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
	if isString(t.info.Types[expr.Left]) || isString(t.info.Types[expr.Right]) {
		return t.compileStringBinary(expr)
	}
	if errExpr, member, ev, ok := t.errorMember(expr); ok {
		return t.compileErrorBinary(expr, errExpr, member, ev)
	}
	return t.compileScalarBinary(expr)
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

	if c, ok := srcType.(analyser.PrimitiveType); ok && c.Name == "char" {
		if s, ok := dstType.(analyser.PrimitiveType); ok && s.Name == "string" {
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
		}
	}

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

			ops, err := t.compileCall(t.useStrfind(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{b})
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

			ops, err := t.compileCall(t.useStrinstrings(), false, []ast.Expression{expr.Left, syn}, []analyser.Type{b})
			if err != nil {
				return nil, err
			}
			return ops[0], nil
		case analyser.SpanType:
			st, ok := r.Element.(analyser.PrimitiveType)
			if !ok || st.Name != "string" {
				break
			}

			ops, err := t.compileCall(t.useStrinstrings(), false, []ast.Expression{expr.Left, expr.Right}, []analyser.Type{b})
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

func (t *X64Target) returnError(errOp Operand) {
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
}

func (t *X64Target) compilePropagate(expr *ast.PropagateExpr) ([]Operand, error) {
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

	t.returnError(Reg{reg, 8})

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

	if expr.Block != nil {
		t.collectVariables(expr.Block)
		if err := t.compileStatement(expr.Block); err != nil {
			return nil, err
		}
	} else {
		defaults := []ast.Expression{expr.Default}
		if e, ok := expr.Default.(*ast.TupleExpr); ok {
			defaults = e.Elements
		}
		for i, def := range defaults {
			op, err := t.compileExpr(def)
			if err != nil {
				return nil, err
			}
			t.store(op, slots[i])
		}
	}

	t.text.printf("%s:\n", endLabel)

	res := make([]Operand, len(slots))
	for i, s := range slots {
		res[i] = s
	}
	return res, nil
}
