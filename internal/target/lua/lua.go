package lua

import (
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type LuaTarget struct{}

func New() *LuaTarget {
	return &LuaTarget{}
}

func (*LuaTarget) Name() string {
	return "lua 5"
}

func (*LuaTarget) Compile(program *ast.Program, info *analyser.Info) (string, error) {
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

	b.WriteString("_PROGRAM_EXIT_CODE = main() or 0\n")
	b.WriteString("print(\"exit code: \".._PROGRAM_EXIT_CODE)\n")

	return b.String(), nil
}
