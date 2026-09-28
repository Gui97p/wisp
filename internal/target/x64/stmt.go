package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileStatement(stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return t.compileReturn(s)
	default:
		return fmt.Errorf("x86-64: unsupported statement %T", stmt)
	}
}

func (t *X64Target) compileReturn(stmt *ast.ReturnStmt) error {
	if len(stmt.Values) != 1 {
		return fmt.Errorf("x86-64: only single-value return supported for now")
	}

	lit := stmt.Values[0]
	return t.compileExpression(lit)
}
