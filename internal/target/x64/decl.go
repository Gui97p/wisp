package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/target"
	x64context "github.com/Gui97p/wisp/internal/target/x64/context"
)

func (t *X64Target) compileDeclaration(decl ast.Declaration) error {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		ctx := t.ctx
		t.ctx = x64context.NewContext()
		err := t.compileFunc(d)
		t.ctx = ctx
		return err
	case *ast.ImportDecl, *ast.ReexportDecl:
		return nil
	default:
		return fmt.Errorf("x86-64: unsupported declaration %T", d)
	}
}

func (t *X64Target) compileFunc(fd *ast.FuncDecl) error {
	if fd.Body != nil {
		if err := target.CheckNative(t.target, ast.NativeBackends(fd.Body)); err != nil {
			return fmt.Errorf("x86-64: function %s %w", fd.Name, err)
		}
	}

	name := t.funcSymbol(t.module, fd.Name)

	if fd.Exported {
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

	t.funcReturns = t.info.FuncReturns[fd]
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

	if err := t.compileStatement(fd.Body); err != nil {
		return err
	}

	t.text.newLine()
	t.text.println(".return:")
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
		t.collectVariables(s.Body)
	case *ast.LoopStmt:
		t.collectVariables(s.Body)
	}
}
