package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileIntLiteral(b *strings.Builder, lit *ast.IntLiteral) error {
	fmt.Fprintf(b, "\tmov rax, %d\n", lit.Value)
	return nil
}
