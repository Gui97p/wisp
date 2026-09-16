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
		}
	}
	return nil
}

func (t *LuaTarget) compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "local function %s(", fd.Name)
	for k, v := range fd.Params {
		b.WriteString(v.Name)
		if k != len(fd.Params)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteString(")\n")

	if err := t.compileStatement(b, fd.Body); err != nil {
		return err
	}
	b.WriteString("end\n")

	return nil
}
