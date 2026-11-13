package x64

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileDeclaration(decl ast.Declaration) error {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Extern {
			return nil
		}
		t.file = t.files[decl]
		ctx := t.ctx
		t.ctx = x64context.NewContext()
		err := t.compileFunc(d)
		t.ctx = ctx
		if err != nil {
			return err
		}
		return t.drainLambdas()
	case *ast.VarDecl:
		syms := t.info.VarSymbols[d]
		for i := range d.Vars {
			sym := syms[i]
			l := symbolName(sym.Module, sym.Name)
			if d.Exported {
				t.data.printft("global %s\n", l)
			}
			t.data.printft("align 8\n%s:\n", l)

			var init ast.Expression
			if len(d.Values) > 0 {
				init = d.Values[i]
			}
			if err := t.emitData(init, sym.Type); err != nil {
				return err
			}
			t.drainLambdas()
		}
		return nil
	case *ast.StructDecl:
		key := analyser.TypeKey(t.module, d.Name)
		sym := "__wisp_type_" + re.ReplaceAllString(key, "_")
		t.data.printft("global %s\nalign 8\n%s:\n\tdq 1\n", sym, sym)
	case *ast.TypeDecl:
		key := analyser.TypeKey(t.module, d.Name)
		sym := "__wisp_type_" + re.ReplaceAllString(key, "_")
		t.data.printft("global %s\nalign 8\n%s:\n\tdq 1\n", sym, sym)
	case *ast.ImportDecl, *ast.ConstDecl:
	default:
		return fmt.Errorf("x86-64: unsupported declaration %T", d)
	}
	return nil
}

func (t *X64Target) drainLambdas() error {
	for len(t.lambdas) > 0 {
		l := t.lambdas[0]
		t.lambdas = t.lambdas[1:]
		ctx := t.ctx
		t.ctx = x64context.NewContext()
		err := t.compileFuncBody(l.name, l.display, false, t.info.VarSymbols[l.lit], t.info.Types[l.lit].(*analyser.FuncType).Returns, l.lit.Fallible, l.lit.Block)
		t.ctx = ctx
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *X64Target) emitData(expr ast.Expression, tp analyser.Type) error {
	size := sizeOf(tp)

	if expr == nil {
		t.data.printft("times %d db 0\n", size)
		return nil
	}

	switch u := underlying(tp).(type) {
	case analyser.PrimitiveType:
		value, ok := t.info.ConstValues[expr]
		if !ok {
			return fmt.Errorf("x86-64: unsupported global initializer")
		}
		if u.Name == "string" {
			dl, dn, err := t.stringData(value.Str)
			if err != nil {
				return err
			}
			t.data.printft("dq %s\n\tdq %d\n", dl, dn)
			return nil
		}
		t.data.printft("%s ", directive[size])
		switch value.Kind {
		case analyser.ConstBool:
			if value.Bool {
				t.data.println("1")
			} else {
				t.data.println("0")
			}
		case analyser.ConstInt:
			if isFloat(tp) {
				t.data.printf("%d\n", floatBits(float64(value.Bits()), size))
			} else {
				t.data.printf("%d\n", value.Bits())
			}
		case analyser.ConstFloat:
			t.data.printf("%d\n", floatBits(value.Float, size))
		}
		return nil
	case analyser.PointerType:
		t.data.printlnt("dq 0")
		return nil
	case analyser.ArrayType:
		lit, ok := expr.(*ast.ArrayLiteral)
		if !ok {
			return fmt.Errorf("x86-64: unsupported global initializer")
		}
		for _, element := range lit.Elements {
			if err := t.emitData(element, u.Element); err != nil {
				return err
			}
		}
		if missing := int(u.Size) - len(lit.Elements); missing > 0 {
			t.data.printft("times %d db 0\n", missing*sizeOf(u.Element))
		}
		return nil
	case *analyser.StructType:
		lit, ok := expr.(*ast.StructLiteral)
		if !ok {
			return fmt.Errorf("x86-64: unsupported global initializer")
		}
		st := structOf(u)
		layout := layoutOf(st)

		values := map[string]ast.Expression{}
		for i, key := range lit.Keys {
			values[key] = lit.Values[i]
		}

		pos := 0
		for _, field := range st.Order {
			lField := layout.Fields[field]
			if lField.Offset > pos {
				t.data.printft("times %d db 0\n", lField.Offset-pos)
			}
			if err := t.emitData(values[field], lField.Type); err != nil {
				return err
			}
			pos = lField.Offset + lField.Size
		}
		if layout.Size > pos {
			t.data.printft("times %d db 0\n", layout.Size-pos)
		}
		return nil
	case *analyser.FuncType:
		switch e := expr.(type) {
		case *ast.NullLiteral:
			t.data.printlnt("dq 0")
		case *ast.FuncLiteral:
			t.data.printft("dq %s\n", t.queueLambda(e))
		case *ast.IdentLiteral:
			t.data.printft("dq %s\n", t.funcLabel(t.info.Idents[e]))
		case *ast.MemberExpr:
			sym, err := t.resolveCallee(e)
			if err != nil {
				return err
			}
			t.data.printft("dq %s\n", t.funcLabel(sym))
		default:
			return fmt.Errorf("x86-64: unsupported global initializer")
		}
		return nil
	}

	return fmt.Errorf("x86-64: unsupported global initializer")
}

const minRegisterWeight = 8

func classOf(tp analyser.Type) x64context.Class {
	switch t := underlying(tp).(type) {
	case analyser.PrimitiveType:
		switch t.Name {
		case "float", "float32", "float64":
			return x64context.SSE
		case "string":
		default:
			return x64context.Int
		}
	case *analyser.FuncType, analyser.ImplicitEnumType, analyser.PointerType:
		return x64context.Int
	}
	return x64context.Memory
}

func (t *X64Target) compileFuncBody(name, display string, entry bool, params []*analyser.Symbol, returns []analyser.Type, fallible bool, body *ast.BlockStmt) error {
	t.funcName = display
	t.sites = nil
	t.candidates = nil
	t.loopSyms = map[*ast.ForStmt]*loopState{}
	desc, err := newDesc(t.format, name, name, true, 32)
	if err != nil {
		return err
	}
	t.text.printf("\nsection .text%s\n", desc)
	t.text.printf("%s:\n", name)

	if t.trace {
		t.ctx.Reserve(8)
	}

	types := []analyser.Type{}
	for _, sym := range params {
		types = append(types, sym.Type)
	}

	t.funcReturns = slices.Clone(returns)
	t.fallible = fallible
	if fallible {
		t.funcReturns = append(t.funcReturns, analyser.ErrorType{})
	}

	sret := usesSret(t.funcReturns)
	args := assignArgs(types, sret)

	stackSize := t.ctx.Size()

	n := 0
	if t.optimize && !t.trace && !entry && !fallible && !sret {
		ps := map[*analyser.Symbol]bool{}
		ok := len(args) == len(params)
		for i, arg := range args {
			if classOf(params[i].Type) != x64context.Int || len(arg.Regs) != 1 {
				ok = false
			} else {
				ps[params[i]] = true
			}
		}
		if ok {
			n = t.framelessPrefix(body, ps)
		}
		if n == len(body.Statements) {
			n = 0
		}
	}
	if n > 0 {
		t.frameless = true
		for i, arg := range args {
			t.ctx.Bind(params[i], arg.Regs[0])
			t.ctx.AllocRegister(arg.Regs[0])
		}
		for _, s := range body.Statements[:n] {
			if err := t.compileStatement(s); err != nil {
				return err
			}
		}
		t.frameless = false
		for i, arg := range args {
			t.ctx.Unbind(params[i])
			t.ctx.FreeRegister(arg.Regs[0])
		}
	}

	if t.ctx.Size() != stackSize {
		return fmt.Errorf("x86-64: frameless prefix used the stack")
	}

	for _, sym := range params {
		t.ctx.Set(sym, sizeOf(sym.Type))
	}

	t.text.printlnt("push rbp")
	t.text.printlnt("mov rbp, rsp")

	if sret {
		t.hidden = slot(t.ctx.Reserve(8), 8)
		t.text.printft("mov %s, rdi\n", t.memText(t.hidden))
	}

	t.collectVariables(body)
	if t.optimize {
		weight := map[*analyser.Symbol]int{}
		barred := map[*analyser.Symbol]bool{}
		t.weigh(&ast.BlockStmt{Statements: body.Statements[n:]}, 0, weight, barred)

		for sym, w := range weight {
			if barred[sym] || w < minRegisterWeight {
				continue
			}
			class := classOf(sym.Type)
			if class == x64context.Memory {
				continue
			}
			t.candidates = append(t.candidates, candidate{sym: sym, weight: w, class: class})
		}
		slices.SortFunc(t.candidates, func(a, b candidate) int {
			return cmp.Or(b.weight-a.weight, a.sym.Line-b.sym.Line, a.sym.Col-b.sym.Col)
		})

		ni, nx := 0, 0
		for _, cand := range t.candidates {
			switch cand.class {
			case x64context.Int:
				if ni < len(x64context.SavedOrder) {
					t.ctx.Bind(cand.sym, x64context.SavedOrder[ni])
					ni++
				}
			case x64context.SSE:
				if nx < len(x64context.XMMSavedOrder) {
					t.ctx.Bind(cand.sym, x64context.XMMSavedOrder[nx])
					nx++
				}
			}
		}
	}
	t.text.printlnt("sub rsp, .frame\n")
	if t.trace {
		t.text.printft("lea r11, [rel desc_%s]\n", name)
		t.text.printlnt("mov [rbp-8], r11")
	}

	t.savedRegs = nil
	for _, reg := range t.ctx.SavedRegisters() {
		off := t.ctx.Reserve(8)
		t.text.printft("mov %s, %s\n", t.memText(slot(off, 8)), t.ctx.GetRegister(reg, 8))
		t.savedRegs = append(t.savedRegs, savedReg{reg, off})
	}

	for i, arg := range args {
		sym := params[i]
		loc := t.local(sym)
		size := sizeOf(sym.Type)
		for j, reg := range arg.Regs {
			if home, ok := loc.(Reg); ok {
				t.loadWord(home.Reg, Reg{reg, home.Size})
			} else {
				wordSize := min(8, size-8*j)
				t.storeWord(loc.(Mem).at(8*j, wordSize), Reg{reg, wordSize})
			}
		}
	}

	for i, arg := range args {
		sym := params[i]
		offset, _ := t.ctx.Get(sym)
		size := sizeOf(sym.Type)
		if home, ok := t.local(sym).(Reg); ok && len(arg.Regs) == 0 && size > 0 {
			t.loadWord(home.Reg, deref(x64context.BP, 16+arg.Stack, size))
			continue
		}
		if len(arg.Regs) == 0 && size > 0 {
			for k := range (size + 7) / 8 {
				reg := t.ctx.AllocFreeRegister()
				if reg == x64context.NoReg {
					return fmt.Errorf("x86-64: no available registers")
				}
				t.loadWord(reg, deref(x64context.BP, 16+arg.Stack+8*k, 8))
				t.storeWord(slot(offset, size).at(8*k, min(8, size-8*k)), Reg{reg, 8})
				t.ctx.FreeRegister(reg)
			}
		}
	}

	t.deferScopes = nil
	l := len(body.Statements)
	t.endsInReturn = l > 0 && isReturn(body.Statements[l-1])
	if t.endsInReturn {
		t.lastReturn = body.Statements[l-1]
	}
	if err := t.compileStatement(&ast.BlockStmt{Statements: body.Statements[n:]}); err != nil {
		return err
	}

	t.text.newLine()
	if fallible && len(t.funcReturns) == 1 && !t.endsInReturn {
		t.text.printlnt("xor eax, eax")
	}
	t.text.println(".return:")

	if entry {
		if fallible {
			reg := "rax"
			if len(returns) > 0 {
				reg = "rdx"
			}

			t.prelude.define("__wisp_error_emit")
			t.text.printft(`test %s, %s
		jz .ok
		push %s
		mov rdi, %s
		call __wisp_error_emit
		pop rax
		mov rax, [rax+16]
	
		.ok:
	`, reg, reg, reg, reg)
		} else if len(returns) == 0 {
			t.text.printlnt("xor eax, eax")
		}
	}

	for _, saved := range t.savedRegs {
		t.text.printft("mov %s, %s\n", t.ctx.GetRegister(saved.reg, 8), t.memText(slot(saved.offset, 8)))
	}

	t.text.printlnt("mov rsp, rbp")
	t.text.printlnt("pop rbp")
	t.text.printlnt("ret")
	t.text.printft(".frame equ %d\n", t.ctx.AlignTo(16))

	if t.trace {
		t.emitDescriptor(name)
	}
	tText, err := textOf(t.format)
	if err != nil {
		return err
	}
	if tText != "" {
		tText = " " + tText
	}
	t.text.printf("section .text%s\n", tText)

	if t.ctx.Pushed() != 0 {
		return fmt.Errorf("x86-64: stack alignment failed")
	}

	t.funcName = ""
	return nil
}

func isReturn(stmt ast.Statement) bool {
	_, ok := stmt.(*ast.ReturnStmt)
	return ok
}

func isYield(stmt ast.Statement) bool {
	_, ok := stmt.(*ast.YieldStmt)
	return ok
}

const descriptorMagic = 0x57495350

func (t *X64Target) emitDescriptor(name string) {
	desc, err := newDesc(t.format, name, "desc_"+name, false, 8)
	if err != nil {
		t.fail("%s", err.Error())
		return
	}
	t.text.printf("section .rodata%s\n", desc)
	t.text.printf("desc_%s:\n", name)
	t.text.printft("dq 0x%x\n", descriptorMagic)
	t.text.printft("dq %s\n", name)
	t.text.printft("dq %d\n", len(t.sites))
	for i, site := range t.sites {
		t.text.printft("dq %s%s - %s, .s%d, %d\n", name, site.label, name, i, len(site.text))
	}
	for i, site := range t.sites {
		data := bytesToAsm([]byte(site.text))
		if data == "" {
			data = "0"
		}
		t.text.printf(".s%d: db %s\n", i, data)
	}
}

func (t *X64Target) compileFunc(fd *ast.FuncDecl) error {
	if fd.Inline && fd.HasConstParams() {
		return nil
	}

	fdName := fd.Name
	if fd.Receiver != nil {
		fdName = fmt.Sprintf("%s_%s", fd.Receiver.Type.Name, fd.Name)
	}
	name := t.funcSymbol(t.module, fdName)
	display := t.module + "."
	if fd.CExport != "" {
		name = fd.CExport
		display = fd.CExport
	}

	visible := fd.Exported
	if fd.Receiver != nil {
		visible = t.info.MethodVisible[fd]
		display += fd.Receiver.Type.Name + "."
	}
	if visible {
		t.text.printf("global %s\n", name)
	}

	return t.compileFuncBody(name, display+fd.Name, t.isEntry && fd.Name == "main", t.info.VarSymbols[fd], t.info.FuncReturns[fd], fd.Fallible, fd.Body)
}

func (t *X64Target) synthetic(name string, tp analyser.Type, at ast.Node, order int) *analyser.Symbol {
	line, col := at.Position()
	return &analyser.Symbol{Name: name, Kind: analyser.VAR, Type: tp, Line: line, Col: col + order}
}

func (t *X64Target) weigh(root ast.Node, depth int, weight map[*analyser.Symbol]int, barred map[*analyser.Symbol]bool) {
	ast.Inspect(root, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ForStmt:
			t.weigh(n.Body, depth+1, weight, barred)
			t.weigh(n.Start, depth, weight, barred)
			t.weigh(n.End, depth+1, weight, barred)
			t.weigh(n.Step, depth+1, weight, barred)
			t.weigh(n.Range, depth, weight, barred)
			vs := t.info.VarSymbols[n]
			if n.Var != "" {
				weight[vs[0]] += 2 << (3 * min(depth+1, 8))
			}
			if n.Var2 != "" {
				weight[vs[1]] += 2 << (3 * min(depth+1, 8))
			}
			if n.Range != nil {
				intTp := analyser.PrimitiveType{Name: "int"}
				st := &loopState{
					i: t.synthetic("__wisp_for_i", intTp, n, 2),
				}
				t.loopSyms[n] = st
				b := 1 << (3 * min(depth+1, 8))
				weight[st.i] += 3 * b
				tp := t.info.Types[n.Range]
				if _, ok := tp.(analyser.ArrayType); !ok {
					st.n = t.synthetic("__wisp_for_n", intTp, n, 1)
					weight[st.n] += b
				}
				if n.Var2 != "" {
					st.p = t.synthetic("__wisp_for_p", intTp, n, 0)
					weight[st.p] += b
				}
			} else if n.Start == nil {
				st := &loopState{count: t.synthetic("__wisp_for_count", t.info.Types[n.End], n, 0)}
				t.loopSyms[n] = st
				weight[st.count] += 2 << (3 * min(depth+1, 8))
			}
			return false
		case *ast.LoopStmt:
			t.weigh(n.Body, depth+1, weight, barred)
			if n.Condition != nil {
				t.weigh(n.Condition, depth+1, weight, barred)
			}
			if n.UntilCondition != nil {
				t.weigh(n.UntilCondition, depth+1, weight, barred)
			}
			return false
		case *ast.IdentLiteral:
			sym := t.info.Idents[n]
			if sym != nil && !sym.Global && (sym.Kind == analyser.VAR || sym.Kind == analyser.PARAM) {
				weight[sym] += 1 << (3 * min(depth, 8))
			}
			return false
		case *ast.UnaryExpr:
			if ident, ok := n.Value.(*ast.IdentLiteral); ok && n.Operator == "&" {
				barred[t.info.Idents[ident]] = true
			}
			return true
		case *ast.CallExpr:
			if call, ok := t.info.Inlines[n]; ok {
				t.weigh(call.Body, depth, weight, barred)
			}
			if mc, ok := t.info.MethodCalls[n]; ok && mc.PointerReceiver && !mc.ObjectIsPointer {
				if ident, ok := mc.Receiver.(*ast.IdentLiteral); ok {
					barred[t.info.Idents[ident]] = true
				}
			}
			return true
		case *ast.NativeStmt:
			for _, bind := range n.Bindings {
				if bind.Var != nil {
					barred[t.info.Idents[bind.Var]] = true
				}
			}
			return false
		case *ast.FuncLiteral:
			return false
		}
		return true
	})
}

func (t *X64Target) collectVariables(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, stmt := range s.Statements {
			t.collectVariables(stmt)
		}
	case *ast.VarStmt:
		for i := range s.Vars {
			t.ctx.Set(t.info.VarSymbols[s][i], sizeOf(t.info.VarTypes[s][i]))
		}
	case *ast.IfStmt:
		t.collectVariables(s.Then)
		if s.Else != nil {
			t.collectVariables(s.Else)
		}
	case *ast.ForStmt:
		if s.Var != "" {
			t.ctx.Set(t.info.VarSymbols[s][0], sizeOf(t.info.VarTypes[s][0]))
		}
		if s.Var2 != "" {
			t.ctx.Set(t.info.VarSymbols[s][1], sizeOf(t.info.VarTypes[s][1]))
		}
		t.collectVariables(s.Body)
	case *ast.LoopStmt:
		t.collectVariables(s.Body)
	case *ast.DeferStmt:
		t.collectVariables(s.Body)
	}
}
