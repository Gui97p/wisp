package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
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
	default:
		return fmt.Errorf("x86-64: unsupported declaration %T", d)
	}
}

func (t *X64Target) compileFunc(fd *ast.FuncDecl) error {
	if fd.Exported {
		t.text.printf("global %s\n", fd.Name)
	}

	t.text.printf("\n%s:\n", fd.Name)
	t.text.printlnt("push rbp")
	t.text.printlnt("mov rbp, rsp")

	for i := range fd.Params {
		t.ctx.Set(t.info.VarSymbols[fd][i], t.sizeOf(t.info.VarTypes[fd][i]))
	}
	t.collectVariables(fd.Body)
	t.text.printft("sub rsp, %d\n\n", t.ctx.AlignTo(16))

	regIdx := 0
	for paramIdx := range fd.Params {
		offset, ok := t.ctx.Get(t.info.VarSymbols[fd][paramIdx])
		if !ok {
			return fmt.Errorf("x86-64: error on allocating parameter offset")
		}

		size := t.sizeOf(t.info.VarTypes[fd][paramIdx])
		regsNeeded := (size + 7) / 8

		if regIdx+regsNeeded > len(x64context.ParamOrder) {
			return fmt.Errorf("x84-64: max parameter size reached")
		}

		for j := range regsNeeded {
			reg := x64context.ParamOrder[regIdx]
			t.text.printft("mov [rbp-%d], %s\n", offset-j*8, t.ctx.GetRegister(reg, min(size, 8)))
			t.ctx.FreeRegister(reg)
			regIdx++
			size -= 8
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
			t.ctx.Set(t.info.VarSymbols[s][i], t.sizeOf(t.info.VarTypes[s][i]))
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
