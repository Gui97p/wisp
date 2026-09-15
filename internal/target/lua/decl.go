package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "local function %s()\n", fd.Name)

	for _, stmt := range fd.Body.Statements {
		if err := compileStatement(b, stmt); err != nil {
			return err
		}
	}

	b.WriteString("end\n")

	return nil
}
