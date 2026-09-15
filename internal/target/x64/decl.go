package x64

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "%s:\n", fd.Name)
	b.WriteString("\tpush rbp\n")
	b.WriteString("\tmov rbp, rsp\n")

	if len(fd.ReturnTypes) == 0 {
		b.WriteString("\txor eax, eax\n")
	}
	b.WriteString("\n")

	for _, stmt := range fd.Body.Statements {
		if err := compileStmt(b, stmt); err != nil {
			return err
		}
	}

	b.WriteString("\n\tmov rsp, rbp\n")
	b.WriteString("\tpop rbp\n")
	b.WriteString("\tret\n\n")

	return nil
}
