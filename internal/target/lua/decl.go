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

func funcName(fd *ast.FuncDecl) string {
	if fd.Receiver != nil {
		return fd.Receiver.Type.Name + "_" + fd.Name
	}
	return fd.Name
}

func collectFuncNames(decls []ast.Declaration) []string {
	var names []string
	for _, d := range decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			names = append(names, funcName(fd))
		}
	}
	return names
}

func (t *LuaTarget) compileFuncDeclaration(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "%s = function(", funcName(fd))

	first := true
	if fd.Receiver != nil {
		recvName := fd.Receiver.Name
		if recvName == "" {
			recvName = "_"
		}
		b.WriteString(recvName)
		first = false
	}
	for _, v := range fd.Params {
		if !first {
			b.WriteByte(',')
		}
		b.WriteString(v.Name)
		first = false
	}
	b.WriteString(")\n")

	if err := t.compileStatement(b, fd.Body); err != nil {
		return err
	}
	b.WriteString("end\n")

	return nil
}

