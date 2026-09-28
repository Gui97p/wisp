package x64

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *X64Target) compileExpression(expr ast.Expression) error {
	switch expr.(type) {
	default:
		return fmt.Errorf("x86-64: unsupported expression %T", expr)
	}
}
