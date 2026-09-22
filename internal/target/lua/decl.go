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

func (t *LuaTarget) collectFuncNames(decls []ast.Declaration) []string {
	var names []string
	for _, d := range decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Body != nil || t.nativeAlias(fd.Name) != "" {
				names = append(names, funcName(fd))
			}
		}
	}
	return names
}

func (t *LuaTarget) compileFuncDeclaration(b *strings.Builder, fd *ast.FuncDecl) error {
	if fd.Body == nil {
		if alias := t.nativeAlias(fd.Name); alias != "" {
			fmt.Fprintf(b, "%s = %s\n", funcName(fd), alias)
		}
		return nil
	}

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
	prevCount, prevIdx := t.currentReturnCount, t.currentFallibleIndex
	t.currentReturnCount = len(fd.ReturnTypes)
	t.currentFallibleIndex = -1
	for i, r := range fd.ReturnTypes {
		if r.Fallible {
			t.currentFallibleIndex = i
			break
		}
	}
	defer func() {
		t.currentReturnCount = prevCount
		t.currentFallibleIndex = prevIdx
	}()
	b.WriteString(")\n")

	if err := t.compileStatement(b, fd.Body); err != nil {
		return err
	}
	b.WriteString("end\n")

	return nil
}
