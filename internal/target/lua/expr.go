package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileExpression(b *strings.Builder, expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		compileExpression(b, e.Left)
		b.WriteString(e.Operator)
		compileExpression(b, e.Right)
	case *ast.IntLiteral:
		fmt.Fprintf(b, "%d", e.Value)
	default:
		return fmt.Errorf("lua: unsupported expression %T", expr)
	}
	return nil
}
