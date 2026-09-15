package lua

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
		return fmt.Errorf("lua: unsupported statement %T", stmt)
	}
}

func compileReturn(b *strings.Builder, stmt *ast.ReturnStmt) error {
	if len(stmt.Values) != 1 {
		return fmt.Errorf("lua: only single-value return supported for now")
	}

	lit := stmt.Values[0]
	b.WriteString("return ")
	if err := compileExpression(b, lit); err != nil {
		return err
	}
	b.WriteRune('\n')
	return nil
}
