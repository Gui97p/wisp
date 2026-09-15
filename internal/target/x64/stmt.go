package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileStmt(b *strings.Builder, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return compileReturn(b, s)
	default:
		return fmt.Errorf("x86-64: unsupported statement %T", stmt)
	}
}

func compileReturn(b *strings.Builder, stmt *ast.ReturnStmt) error {
	if len(stmt.Values) != 1 {
		return fmt.Errorf("x86-64: only single-value return supported for now")
	}
	lit, ok := stmt.Values[0].(*ast.IntLiteral)
	if !ok {
		return fmt.Errorf("x86-64: only integer literal return supported for now")
	}
	fmt.Fprintf(b, "\tmov eax, %d\n", lit.Value)
	return nil
}
