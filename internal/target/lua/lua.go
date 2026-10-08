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

	loopStack    []loopContext
	labelCounter uint64

	currentFallible bool
	currentPayloads []analyser.Type

	pending []string
}

func New(program *ast.Program, info *analyser.Info, isEntry bool) *LuaTarget {
	return &LuaTarget{program: program, info: info, isEntry: isEntry}
}

func (*LuaTarget) Name() string {
	return "lua"
}

func (t *LuaTarget) Compile() (string, error) {
	var b strings.Builder

	for _, d := range t.program.Declarations {
		if imp, ok := d.(*ast.ImportDecl); ok {
			fmt.Fprintf(&b, "local %s = require(%q)\n", imp.Alias, ModuleID(imp.Path))
		}
	}

	names := collectFuncNames(t.program.Declarations)
	if len(names) > 0 {
		fmt.Fprintf(&b, "local %s\n", strings.Join(names, ", "))
	}

	if err := t.compileDeclarations(&b, t.program.Declarations); err != nil {
		return "", err
	}

	if t.isEntry {
		b.WriteString(t.entryErrorCheck())
	} else {
		b.WriteString("local __wisp_module = {}\n")
		t.writeExports(&b)
		b.WriteString("return __wisp_module\n")
	}

	body := b.String()
	return preludeFor(body) + body, nil
}

var luaBuiltins = map[string]bool{
	"math": true, "string": true, "table": true, "os": true, "io": true,
	"coroutine": true, "utf8": true, "debug": true, "package": true, "bit32": true,
}

func ModuleID(path string) string {
	path = strings.TrimSuffix(path, ".wsp")
	dotted := strings.ReplaceAll(path, "/", ".")
	if luaBuiltins[dotted] {
		return "wisp." + dotted
	}
	return dotted
}

func Build(luaSource, outputPath string) error {
	if err := os.WriteFile(outputPath, []byte(luaSource), 0644); err != nil {
		return fmt.Errorf("failed to write lua file: %w", err)
	}

	return nil
}

func (t *LuaTarget) writeExports(b *strings.Builder) {
	for _, d := range t.program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Receiver != nil {
				if t.info.MethodVisible[decl] {
					fmt.Fprintf(b, "__wisp_module.%s = %s\n", funcName(decl), funcName(decl))
				}
				continue
			}
			if !decl.Exported {
				continue
			}
			fmt.Fprintf(b, "__wisp_module.%s = %s\n", decl.Name, decl.Name)
		case *ast.ConstDecl:
			if !decl.Exported {
				continue
			}
			for _, v := range decl.Vars {
				fmt.Fprintf(b, "__wisp_module.%s = %s\n", v.Name, v.Name)
			}
		case *ast.ReexportDecl:
			if decl.Alias != "" {
				fmt.Fprintf(b, "__wisp_module.%s = %s\n", decl.Alias, decl.Name)
				continue
			}
			for _, name := range t.info.Reexports[decl] {
				fmt.Fprintf(b, "__wisp_module.%s = %s.%s\n", name, decl.Name, name)
			}
		}
	}
}

func (t *LuaTarget) importAlias(path string) (string, bool) {
	for _, d := range t.program.Declarations {
		if imp, ok := d.(*ast.ImportDecl); ok && imp.Path == path {
			return imp.Alias, true
		}
	}
	return "", false
}
