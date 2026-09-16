package lua

import (
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type LuaTarget struct {
	info *analyser.Info
}

func New() *LuaTarget {
	return &LuaTarget{}
}

func (*LuaTarget) Name() string {
	return "lua 5"
}

func (t *LuaTarget) Compile(program *ast.Program, info *analyser.Info) (string, error) {
	t.info = info

	var b strings.Builder

	for _, decl := range program.Declarations {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if err := compileFunc(&b, fd); err != nil {
			return "", err
		}
	}

	b.WriteString("__wisp_program_exit_code = main() or 0\n")
	b.WriteString("os.exit(__wisp_program_exit_code)\n")

	return b.String(), nil
}
