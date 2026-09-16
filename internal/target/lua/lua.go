package lua

import (
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type LuaTarget struct {
	program      *ast.Program
	info         *analyser.Info
	loopStack    []loopContext
	labelCounter uint64
}

func New(program *ast.Program, info *analyser.Info) *LuaTarget {
	return &LuaTarget{program: program, info: info}
}

func (*LuaTarget) Name() string {
	return "lua 5"
}

func (t *LuaTarget) Compile() (string, error) {
	var b strings.Builder

	if err := t.compileDeclarations(&b, t.program.Declarations); err != nil {
		return "", err
	}

	b.WriteString("__wisp_program_exit_code = main() or 0\n")
	b.WriteString("os.exit(__wisp_program_exit_code)\n")

	return b.String(), nil
}
