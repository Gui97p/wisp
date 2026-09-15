package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "local function %s(", fd.Name)
	for k, v := range fd.Params {
		b.WriteString(v.Name)
		if k != len(fd.Params)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteString(")\n")

	for _, stmt := range fd.Body.Statements {
		if err := compileStatement(b, stmt); err != nil {
			return err
		}
	}

	b.WriteString("end\n")

	return nil
}
