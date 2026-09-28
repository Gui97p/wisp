package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileDeclaration(decl ast.Declaration) error {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return t.compileFunc(d)
	default:
		return fmt.Errorf("x86-64: unsupported declaration %T", d)
	}
}

func (t *X64Target) compileFunc(fd *ast.FuncDecl) error {
	t.text.printf("\n%s:\n", fd.Name)
	t.text.printlnt("push rbp")
	t.text.printlnt("mov rbp, rsp\n")

	t.text.printlnt("mov rsp, rbp")
	t.text.printlnt("pop rbp")
	t.text.printlnt("ret")

	return nil
}
