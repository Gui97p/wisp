package x64

import (
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
		return err
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
	case *ast.ImportDecl, *ast.ReexportDecl, *ast.ConstDecl:
	default:
		return fmt.Errorf("x86-64: unsupported declaration %T", d)
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
			t.data.printf("%d\n", value.Bits())
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
	}

	return fmt.Errorf("x86-64: unsupported global initializer")
}

func (t *X64Target) compileFunc(fd *ast.FuncDecl) error {
	fdName := fd.Name
	if fd.Receiver != nil {
		fdName = fmt.Sprintf("%s_%s", fd.Receiver.Type.Name, fd.Name)
	}
	name := t.funcSymbol(t.module, fdName)
	if fd.CExport != "" {
		name = fd.CExport
	}

	visible := fd.Exported
	if fd.Receiver != nil {
		visible = t.info.MethodVisible[fd]
	}
	if visible {
		t.text.printf("global %s\n", name)
	}

	t.text.printf("\n%s:\n", name)
	t.text.printlnt("push rbp")
	t.text.printlnt("mov rbp, rsp")

	types := []analyser.Type{}
	for _, sym := range t.info.VarSymbols[fd] {
		t.ctx.Set(sym, sizeOf(sym.Type))
		types = append(types, sym.Type)
	}
	t.collectVariables(fd.Body)
	t.text.printlnt("sub rsp, .frame\n")

	t.funcReturns = slices.Clone(t.info.FuncReturns[fd])
	t.fallible = fd.Fallible
	if fd.Fallible {
		t.funcReturns = append(t.funcReturns, analyser.ErrorType{})
	}
	sret := usesSret(t.funcReturns)
	if sret {
		t.hidden = slot(t.ctx.Reserve(8), 8)
		t.text.printft("mov %s, rdi\n", t.memText(t.hidden))
	}

	args := assignArgs(types, sret)
	for i, arg := range args {
		sym := t.info.VarSymbols[fd][i]
		offset, _ := t.ctx.Get(sym)
		size := sizeOf(sym.Type)
		for j, reg := range arg.Regs {
			t.storeWord(slot(offset, size).at(8*j, min(8, size-8*j)), Reg{reg, 8})
		}
	}

	for i, arg := range args {
		sym := t.info.VarSymbols[fd][i]
		offset, _ := t.ctx.Get(sym)
		size := sizeOf(sym.Type)
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
	if err := t.compileStatement(fd.Body); err != nil {
		return err
	}

	t.text.newLine()
	if fd.Fallible && len(t.funcReturns) == 1 {
		t.text.printlnt("xor eax, eax")
	}
	t.text.println(".return:")

	if t.isEntry && fd.Name == "main" {
		if fd.Fallible {
			reg := "rax"
			if len(fd.ReturnTypes) > 0 {
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
		} else if len(fd.ReturnTypes) == 0 {
			t.text.printlnt("xor eax, eax")
		}
	}

	t.text.printlnt("mov rsp, rbp")
	t.text.printlnt("pop rbp")
	t.text.printlnt("ret")
	t.text.printft(".frame equ %d\n", t.ctx.AlignTo(16))

	if t.ctx.Pushed() != 0 {
		return fmt.Errorf("x86-64: stack alignment failed")
	}

	return nil
}

func (t *X64Target) collectVariables(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		for _, stmt := range s.Statements {
			t.collectVariables(stmt)
		}
	case *ast.GroupStmt:
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
