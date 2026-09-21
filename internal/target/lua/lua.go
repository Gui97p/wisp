package lua

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type LuaTarget struct {
	program      *ast.Program
	info         *analyser.Info
	loopStack    []loopContext
	labelCounter uint64

	currentFallibleIndex int
	currentReturnCount   int

	pending       []string
	coalesceStack []string
}

func New(program *ast.Program, info *analyser.Info) *LuaTarget {
	return &LuaTarget{program: program, info: info}
}

func (*LuaTarget) Name() string {
	return "lua 5"
}

func (t *LuaTarget) Compile() (string, error) {
	var b strings.Builder
	b.WriteString(runtimePrelude)

	names := collectFuncNames(t.program.Declarations)
	if len(names) > 0 {
		fmt.Fprintf(&b, "local %s\n", strings.Join(names, ", "))
	}

	if err := t.compileDeclarations(&b, t.program.Declarations); err != nil {
		return "", err
	}

	b.WriteString("main()\n")

	return b.String(), nil
}

func Build(luaSource, outputPath string) error {
	if err := os.WriteFile(outputPath, []byte(luaSource), 0644); err != nil {
		return fmt.Errorf("failed to write asm file: %w", err)
	}

	return nil
}
