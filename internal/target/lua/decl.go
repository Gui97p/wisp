package lua

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func (t *LuaTarget) compileFunc(b *strings.Builder, fd *ast.FuncDecl) error {
	fmt.Fprintf(b, "local function %s(", fd.Name)
	for k, v := range fd.Params {
		b.WriteString(v.Name)
		if k != len(fd.Params)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteString(")\n")

	t.compileStatement(b, fd.Body)
	b.WriteString("end\n")

	return nil
}
