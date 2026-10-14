package lua

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
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

	files map[ast.Declaration]string
}

func (t *LuaTarget) SetFiles(files map[ast.Declaration]string) {
	t.files = files
}

func (t *LuaTarget) locate(node ast.Node, err error) error {
	if err == nil {
		return nil
	}
	line, col := node.Position()
	endLine, endCol := node.EndPosition()
	return diag.Locate("lua", err, line, col, endLine, endCol)
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

	for _, hoisted := range []bool{true, false} {
		for _, d := range t.program.Declarations {
			if isDataDecl(d) != hoisted {
				continue
			}
			if err := t.compileDeclarations(&b, []ast.Declaration{d}); err != nil {
				return "", diag.InFile(err, t.files[d])
			}
		}
	}

	if t.isEntry {
		b.WriteString(t.entryErrorCheck())
	} else {
		b.WriteString(t.moduleTable())
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
		case *ast.VarDecl:
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

func isDataDecl(d ast.Declaration) bool {
	switch d.(type) {
	case *ast.ConstDecl, *ast.VarDecl:
		return true
	}
	return false
}

func (t *LuaTarget) moduleTable() string {
	var names []string
	for _, d := range t.program.Declarations {
		if vd, ok := d.(*ast.VarDecl); ok && vd.Exported {
			for _, v := range vd.Vars {
				names = append(names, v.Name)
			}
		}
	}
	if len(names) == 0 {
		return "local __wisp_module = {}\n"
	}

	var b strings.Builder
	b.WriteString("local __wisp_module = setmetatable({}, {\n")
	b.WriteString("  __index = function(_, k)\n")
	for _, name := range names {
		fmt.Fprintf(&b, "    if k == %q then return %s end\n", name, name)
	}
	b.WriteString("  end,\n")
	b.WriteString("  __newindex = function(_, k, v)\n")
	for _, name := range names {
		fmt.Fprintf(&b, "    if k == %q then %s = v return end\n", name, name)
	}
	b.WriteString("    rawset(_, k, v)\n")
	b.WriteString("  end,\n")
	b.WriteString("})\n")
	return b.String()
}
