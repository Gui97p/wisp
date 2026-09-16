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
		case *ast.StructDecl:
			if err := t.compileStructDeclaration(b, d); err != nil {
				return err
			}
		case *ast.ConstDecl:
			if err := t.compileConstDeclaration(b, d); err != nil {
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

func (t *LuaTarget) compileStructDeclaration(b *strings.Builder, sd *ast.StructDecl) error {
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

func (t *LuaTarget) compileConstDeclaration(b *strings.Builder, cd *ast.ConstDecl) error {
	b.WriteString("local ")
	for k, variable := range cd.Vars {
		if k > 0 {
			b.WriteString(", ")
		}
		b.WriteString(variable.Name)
	}

	b.WriteString(" = ")
	if len(cd.Values) > 0 {
		for k, value := range cd.Values {
			if k > 0 {
				b.WriteString(", ")
			}

			if err := t.compileExpression(b, value); err != nil {
				return err
			}
		}
	} else {
		types := t.info.VarTypes[cd]
		for k := range cd.Vars {
			if k > 0 {
				b.WriteString(", ")
			}

			value, err := zeroValue(types[k])
			if err != nil {
				return err
			}

			b.WriteString(value)
		}
	}
	b.WriteByte('\n')
	return nil
}
