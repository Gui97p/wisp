package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileStatement(b *strings.Builder, stmt ast.Statement) error {
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

	lit := stmt.Values[0]
	return compileExpression(b, lit)
}
