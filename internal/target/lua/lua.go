package lua

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
)

type LuaTarget struct {
	program *ast.Program
	info    *analyser.Info
	isEntry bool
	modPath string

	loopStack    []loopContext
	labelCounter uint64

	currentFallibleIndex int
	currentReturnCount   int

	pending       []string
	coalesceStack []string
}

func New(program *ast.Program, info *analyser.Info, isEntry bool, modPath string) *LuaTarget {
	return &LuaTarget{program: program, info: info, isEntry: isEntry, modPath: modPath}
}

func (*LuaTarget) Name() string {
	return "lua"
}

func (t *LuaTarget) Compile() (string, error) {
	var b strings.Builder
	b.WriteString(runtimePrelude)

	for _, d := range t.program.Declarations {
		if imp, ok := d.(*ast.ImportDecl); ok {
			fmt.Fprintf(&b, "local %s = require(%q)\n", imp.Alias, luaRequirePath(imp.Path))
		}
	}

	names := t.collectFuncNames(t.program.Declarations)
	if len(names) > 0 {
		fmt.Fprintf(&b, "local %s\n", strings.Join(names, ", "))
	}

	if err := t.compileDeclarations(&b, t.program.Declarations); err != nil {
		return "", err
	}

	if t.isEntry {
		b.WriteString("main()\n")
	} else {
		b.WriteString("local __wisp_module = {}\n")
		for _, name := range t.exportedNames() {
			fmt.Fprintf(&b, "__wisp_module.%s = %s\n", name, name)
		}
		b.WriteString("return __wisp_module\n")
	}

	return b.String(), nil
}

func luaRequirePath(path string) string {
	path = strings.TrimSuffix(path, ".wsp")
	return "__wisp_modules." + strings.ReplaceAll(path, "/", ".")
}

func Build(luaSource, outputPath string) error {
	if err := os.WriteFile(outputPath, []byte(luaSource), 0644); err != nil {
		return fmt.Errorf("failed to write asm file: %w", err)
	}

	return nil
}

func (t *LuaTarget) exportedNames() []string {
	var names []string

	for _, d := range t.program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if !decl.Exported || decl.Receiver != nil {
				continue
			}
			names = append(names, decl.Name)
		case *ast.ConstDecl:
			if !decl.Exported {
				continue
			}
			for _, v := range decl.Vars {
				names = append(names, v.Name)
			}
		}
	}

	return names
}
