package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileDeclarations(b *strings.Builder, decls []ast.Declaration) error {
	for _, decl := range decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if err := t.compileFunc(b, d); err != nil {
				return err
			}
		case *ast.StructDecl:
			if err := t.compileStruct(b, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *LuaTarget) compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "local function %s(", fd.Name)
	for k, v := range fd.Params {
		if k > 0 {
			b.WriteByte(',')
		}
		b.WriteString(v.Name)
	}
	b.WriteString(")\n")

	if err := t.compileStatement(b, fd.Body); err != nil {
		return err
	}
	b.WriteString("end\n")

	return nil
}

func (t *LuaTarget) compileStruct(b *strings.Builder, sd *ast.StructDecl) error {
	fmt.Fprintf(b, "local %s = {}\n", sd.Name)
	fmt.Fprintf(b, "function %s.New(", sd.Name)
	for k, v := range sd.Members {
		if k > 0 {
			b.WriteByte(',')
		}
		b.WriteString(v.Name)
	}
	fmt.Fprintln(b, ")")
	fmt.Fprintln(b, "return {")

	for _, v := range sd.Members {
		fmt.Fprintf(b, "%s = %s;\n", v.Name, v.Name)
	}
	fmt.Fprintln(b, "}\nend")

	return nil
}
