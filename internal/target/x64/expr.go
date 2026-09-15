package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileExpression(b *strings.Builder, expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return compileBinaryExpression(b, e)
	case *ast.IntLiteral:
		return compileIntLiteral(b, e)
	default:
		return fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}
