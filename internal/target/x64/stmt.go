package x64

import (
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileStatement(stmt ast.Statement) error {
	line, col, endLine, endCol := t.enter(stmt)
	defer t.leave(line, col, endLine, endCol)

	return t.locate(stmt, t.compileStatementNode(stmt))
}

func (t *X64Target) compileStatementNode(stmt ast.Statement) error {
	var err error
	liveBefore := len(t.ctx.AllocatedRegisters())

	switch s := stmt.(type) {
	case *ast.NativeStmt:
		err = t.compileNativeStmt(s)
	case *ast.ExpressionStmt:
		var op Operand
		op, err = t.compileExpr(s.Expr)
		if op != nil {
			t.freeOp(op)
		}
		return err
	case *ast.BlockStmt:
		t.pushDeferScope()
		var last ast.Statement
		for _, stmt := range s.Statements {
			err = t.compileStatement(stmt)
			if err != nil {
				break
			}
			last = stmt
		}
		if last != nil && err == nil {
			switch last.(type) {
			case *ast.ReturnStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.YieldStmt:
			default:
				err = t.emitDefers(len(t.deferScopes)-1, false)
			}
		}
		t.popDeferScope()
	case *ast.VarStmt:
		err = t.compileVarStmt(s)
	case *ast.ReturnStmt:
		err = t.compileReturnStmt(s)
	case *ast.YieldStmt:
		err = t.compileYieldStmt(s)
	case *ast.AssignStmt:
		err = t.compileAssignStmt(s)
	case *ast.IncDecStmt:
		err = t.compileIncDecStmt(s)
	case *ast.IfStmt:
		err = t.compileIfStmt(s)
	case *ast.ForStmt:
		err = t.compileForStmt(s)
	case *ast.BreakStmt:
		err = t.compileBreakStmt(s)
	case *ast.ContinueStmt:
		err = t.compileContinueStmt(s)
	case *ast.LoopStmt:
		err = t.compileLoopStmt(s)
	case *ast.DeferStmt:
		t.registerDefer(s)
	case *ast.ConstStmt:
	default:
		return fmt.Errorf("%s is not supported yet", ast.Describe(stmt))
	}

	if t.err != nil {
		return t.err
	}

	if err == nil && len(t.ctx.AllocatedRegisters()) != liveBefore {
		line, col := stmt.Position()
		endLine, endCol := stmt.EndPosition()
		return &diag.Error{Diagnostic: diag.Diagnostic{
			Message: "a register was left allocated after this statement",
			Origin:  "x86-64",
			Line:    line, Col: col, EndLine: endLine, EndCol: endCol,
			Help: "this is a bug in the compiler, so please report it along with the code that triggers it",
		}}
	}
	return err
}

func (t *X64Target) compileVarStmt(stmt *ast.VarStmt) error {
	if len(stmt.Values) == 0 {
		for i := range stmt.Vars {
			sym := t.info.VarSymbols[stmt][i]
			if _, ok := t.ctx.Get(sym); !ok {
				return fmt.Errorf("x86-64: undeclared variable %s", sym.Name)
			}
			t.zero(t.local(sym))
		}
		return nil
	}
	return t.eachValue(stmt.Values, func(i int, op Operand) error {
		sym := t.info.VarSymbols[stmt][i]
		if _, ok := t.ctx.Get(sym); !ok {
			return fmt.Errorf("x86-64: undeclared variable %s", sym.Name)
		}
		t.store(op, t.local(sym))
		return nil
	})
}

func (t *X64Target) compileInlineReturn(stmt *ast.ReturnStmt) error {
	frame := t.inlineFrames[len(t.inlineFrames)-1]
	isError := t.info.ErrorReturns[stmt]
	last := frame.results[len(frame.results)-1].(Mem)

	if isError {
		op, err := t.compileExpr(stmt.Values[0])
		if err != nil {
			return err
		}
		for i := range frame.call.Returns {
			t.zero(frame.results[i].(Mem))
		}
		t.store(op, last)
	} else {
		var ops []Operand
		if err := t.eachValue(stmt.Values, func(i int, op Operand) error {
			ops = append(ops, op)
			return nil
		}); err != nil {
			return err
		}
		for i, op := range ops {
			t.store(op, frame.results[i].(Mem))
		}
		if frame.call.Fallible {
			t.store(Imm(0), last)
		}
	}

	if err := t.emitDefers(frame.deferDepth, isError); err != nil {
		return err
	}
	if stmt != frame.lastReturn {
		t.text.printft("jmp %s\n", frame.label)
	}
	return nil
}

func (t *X64Target) compileReturnStmt(stmt *ast.ReturnStmt) error {
	if _, ok := t.info.InlineReturns[stmt]; ok {
		return t.compileInlineReturn(stmt)
	}

	if t.info.ErrorReturns[stmt] {
		op, err := t.compileExpr(stmt.Values[0])
		if err != nil {
			return err
		}
		return t.returnError(op)
	}

	ops := []Operand{}
	if err := t.eachValue(stmt.Values, func(i int, op Operand) error {
		ops = append(ops, op)
		return nil
	}); err != nil {
		return err
	}

	if t.fallible {
		ops = append(ops, Imm(0))
	}

	if t.hasDefers(0, false) {
		tmps := []Operand{}

		for i, op := range ops {
			if _, ok := op.(Imm); ok {
				tmps = append(tmps, op)
				continue
			}
			size := sizeOf(t.funcReturns[i])
			tmp := slot(t.ctx.Reserve(size), size)
			t.store(op, tmp)
			t.freeOp(op)
			tmps = append(tmps, tmp)
		}

		if err := t.emitDefers(0, false); err != nil {
			return err
		}

		ops = tmps
	}

	t.returnValue(ops, t.funcReturns, t.hidden)
	if stmt != t.lastReturn {
		t.text.printlnt("jmp .return")
	}

	return nil
}

func (t *X64Target) compileAssignStmt(stmt *ast.AssignStmt) error {
	if len(stmt.Targets) == 1 {
		dst, err := t.compileAssignable(stmt.Targets[0])
		if err != nil {
			return err
		}
		targetType := t.info.Types[stmt.Targets[0]]
		valueType := t.info.Types[stmt.Values[0]]

		if stmt.Op == "=" {
			switch s := stmt.Values[0].(type) {
			case *ast.BinaryExpr:
				if s.Operator != "/" && s.Operator != "%" {
					var ident *ast.IdentLiteral
					var expr ast.Expression
					if targetIdent, ok := stmt.Targets[0].(*ast.IdentLiteral); ok {
						targetSym := t.info.Idents[targetIdent]
						if l, ok := s.Left.(*ast.IdentLiteral); ok && t.info.Idents[l] == targetSym {
							ident = l
							expr = s.Right
						} else if r, ok := s.Right.(*ast.IdentLiteral); ok && t.info.Idents[r] == targetSym {
							switch s.Operator {
							case "+", "*", "&", "|", "^":
								ident = r
								expr = s.Left
							}
						}

						if ident != nil {
							value, err := t.compileExpr(expr)
							if err != nil {
								return err
							}
							if r, ok := dst.(Reg); ok && t.ctx.IsHome(r.Reg) {
								t.inPlace = r.Reg
								defer func() { t.inPlace = x64context.NoReg }()
							}
							op, err := t.emitBinary(s.Operator, dst, value, targetType, valueType, targetType, nil)
							if err != nil {
								return err
							}
							t.store(op, dst)
							return nil
						}
					}
				}
			}
			op, err := t.compileExpr(stmt.Values[0])
			if err != nil {
				return err
			}
			t.store(op, dst)
		} else {
			op, err := t.compileExpr(stmt.Values[0])
			if err != nil {
				return err
			}
			cOp := stmt.Op[:len(stmt.Op)-1]
			if r, ok := dst.(Reg); ok && t.ctx.IsHome(r.Reg) && cOp != "/" && cOp != "%" {
				t.inPlace = r.Reg
				defer func() { t.inPlace = x64context.NoReg }()
			}
			res, err := t.emitBinary(cOp, dst, op, targetType, valueType, targetType, nil)
			if err != nil {
				return err
			}
			t.store(res, dst)
		}
		t.freeOp(dst)

		return nil
	}

	err := t.eachValue(stmt.Values, func(i int, op Operand) error {
		t.pushValue(op)
		return nil
	})
	if err != nil {
		return err
	}

	for _, target := range slices.Backward(stmt.Targets) {
		dst, err := t.compileAssignable(target)
		if err != nil {
			return err
		}
		size := sizeOf(t.info.Types[target])
		for w := (size+7)/8 - 1; w >= 0; w-- {
			reg := t.ctx.AllocFreeRegister()

			if reg == x64context.NoReg {
				return fmt.Errorf("x86-64: no available registers")
			}
			t.pop(reg)
			if home, ok := dst.(Reg); ok {
				t.loadWord(home.Reg, Reg{reg, 8})
			} else {
				t.storeWord(dst.(Mem).at(8*w, min(8, size-8*w)), Reg{reg, 8})
			}
			t.ctx.FreeRegister(reg)
		}
		t.freeOp(dst)
	}

	return nil
}

func (t *X64Target) compileIncDecStmt(stmt *ast.IncDecStmt) error {
	memSlot, err := t.compileAssignable(stmt.Target)
	if err != nil {
		return err
	}
	var inst, op string

	switch tp := t.info.Types[stmt.Target].(type) {
	case analyser.PointerType:
		inst = "add"
		if stmt.Op == "--" {
			inst = "sub"
		}
		op = fmt.Sprintf("%s, %d", t.opText(memSlot, 8), sizeOf(tp.Element))
	default:
		inst = incDecOperators[stmt.Op]
		op = t.opText(memSlot, sizeOf(t.info.Types[stmt.Target]))
	}
	t.text.printft("%s %s\n", inst, op)
	t.freeOp(memSlot)
	return nil
}

func (t *X64Target) compileIfStmt(stmt *ast.IfStmt) error {
	elseLabel := t.newLabel("else")
	endLabel := t.newLabel("end")

	if stmt.IsSwitch {
		ctx := x64context.LoopContext{
			BreakLabel:       endLabel,
			SupportsContinue: false,
		}
		t.pushLoop(ctx)
	}

	c, ok := t.info.Folded(stmt.Condition)
	folded := ok && c.Kind == analyser.ConstBool
	if !folded {
		t.branch(stmt.Condition, elseLabel, false)
	}

	if c.Bool || !folded {
		endsInReturn := false
		l := len(stmt.Then.Statements)
		if l > 0 && isReturn(stmt.Then.Statements[l-1]) {
			endsInReturn = true
		}
		if err := t.compileStatement(stmt.Then); err != nil {
			return err
		}
		if !(folded || endsInReturn) {
			t.text.printft("jmp %s\n", endLabel)
		}
	}

	if !c.Bool || !folded {
		if !folded {
			t.text.printf("%s:\n", elseLabel)
		}
		if stmt.Else != nil {
			if err := t.compileStatement(stmt.Else); err != nil {
				return err
			}
		}
	}

	t.text.printf("%s:\n", endLabel)
	return nil
}

func (t *X64Target) compileForStmt(stmt *ast.ForStmt) error {
	switch {
	case stmt.Range != nil:
		return t.compileForRange(stmt)
	case stmt.Start != nil:
		return t.compileForNumeric(stmt)

	default:
		return t.compileForCount(stmt)
	}
}

func (t *X64Target) loopCondition(cond ast.Expression, breakLabel string, until bool) error {
	if cond == nil {
		return nil
	}
	return t.branch(cond, breakLabel, until)
}

func (t *X64Target) compileLoopStmt(stmt *ast.LoopStmt) error {
	condLabel := t.newLabel("cond")

	ctx := x64context.LoopContext{
		Label:            stmt.Label,
		ContinueLabel:    t.newLabel("loop_continue"),
		BreakLabel:       t.newLabel("loop_break"),
		SupportsContinue: true,
	}

	t.pushLoop(ctx)
	defer t.ctx.PopLoop()

	t.text.printf("%s:\n", condLabel)

	if err := t.loopCondition(stmt.Condition, ctx.BreakLabel, false); err != nil {
		return err
	}

	if err := t.compileStatement(stmt.Body); err != nil {
		return err
	}

	t.text.printf("%s:\n", ctx.ContinueLabel)

	if err := t.loopCondition(stmt.UntilCondition, ctx.BreakLabel, true); err != nil {
		return err
	}

	t.text.printft("jmp %s\n", condLabel)
	t.text.printf("%s:\n", ctx.BreakLabel)

	return nil
}

func (t *X64Target) compileBreakStmt(stmt *ast.BreakStmt) error {
	var loop *x64context.LoopContext

	if stmt.Label == "" {
		loop = t.ctx.CurrentLoop()
	} else {
		loop = t.ctx.FindLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("break: no matching loop")
	}

	if err := t.emitDefers(loop.DeferDepth, false); err != nil {
		return err
	}
	t.text.printft("jmp %s\n", loop.BreakLabel)

	return nil
}

func (t *X64Target) compileContinueStmt(stmt *ast.ContinueStmt) error {
	var loop *x64context.LoopContext

	if stmt.Label == "" {
		loop = t.ctx.CurrentContinuable()
	} else {
		loop = t.ctx.FindLoop(stmt.Label)
	}

	if loop == nil {
		return fmt.Errorf("continue: no matching loop")
	}

	if err := t.emitDefers(loop.DeferDepth, false); err != nil {
		return err
	}
	t.text.printft("jmp %s\n", loop.ContinueLabel)

	return nil
}

func (t *X64Target) compileYieldStmt(stmt *ast.YieldStmt) error {
	if len(t.coalesceFrames) == 0 {
		return fmt.Errorf("x86-64: yield outside a ?? handler")
	}
	frame := t.coalesceFrames[len(t.coalesceFrames)-1]

	err := t.eachValue(stmt.Values, func(i int, op Operand) error {
		t.store(op, frame.slots[i])
		return nil
	})
	if err != nil {
		return err
	}

	if err := t.emitDefers(frame.deferDepth, false); err != nil {
		return err
	}
	t.text.printft("jmp %s\n", frame.end)
	return nil
}
