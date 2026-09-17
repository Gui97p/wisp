package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileDeclarations(b *strings.Builder, decls []ast.Declaration) error {
	for _, cd := range decls {
		switch d := cd.(type) {
		case *ast.FuncDecl:
			if err := t.compileFuncDeclaration(b, d); err != nil {
				return err
			}
		case *ast.ConstDecl:
			if err := t.compileDefinition(b, d, d.Vars, d.Values); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *LuaTarget) compileFuncDeclaration(b *strings.Builder, fd *ast.FuncDecl) error {
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

