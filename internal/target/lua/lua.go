package lua

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

type coalesceFrame struct {
	results    []string
	label      string
	deferDepth int
}

type LuaTarget struct {
	coalesceFrames []coalesceFrame

	program *ast.Program
	info    *analyser.Info
	isEntry bool
	target  string

	loopStack    []loopContext
	labelCounter uint64

	currentFallible bool
	currentPayloads []analyser.Type

	deferScopes []*deferScope

	pending []string

	files map[ast.Declaration]string
	id    string

	extraModules map[string]string
	boxed        map[*analyser.Symbol]bool

	aggregates     map[string]*aggregateFns
	aggregateOrder []*aggregateFns
}

func (t *LuaTarget) extraModule(path string) string {
	id := ModuleID(path)
	alias := "__wisp_m_" + strings.NewReplacer(".", "_", "/", "_").Replace(id)
	if t.extraModules == nil {
		t.extraModules = map[string]string{}
	}
	t.extraModules[alias] = id
	return alias
}

func (t *LuaTarget) SetID(id string) {
	t.id = id
}

func (t *LuaTarget) SetTarget(name string) {
	t.target = name
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
	var head, b strings.Builder
	t.collectBoxed()

	if !t.isEntry {
		fmt.Fprintf(&head, "local __wisp_module = {}\n__lua_package.loaded[%q] = __wisp_module\n", t.id)
	}

	for _, d := range t.program.Declarations {
		if imp, ok := d.(*ast.ImportDecl); ok {
			fmt.Fprintf(&head, "local %s = require(%q)\n", luaIdent(imp.Alias), ModuleID(imp.Path))
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

	extras := make([]string, 0, len(t.extraModules))
	for alias := range t.extraModules {
		extras = append(extras, alias)
	}
	sort.Strings(extras)
	for _, alias := range extras {
		fmt.Fprintf(&head, "local %s = require(%q)\n", alias, t.extraModules[alias])
	}

	head.WriteString(t.aggregateCode())

	body := head.String() + b.String()
	return luaGlobals + preludeFor(body) + body, nil
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
		return fmt.Errorf("failed to write the Lua file (%s)", diag.Reason(err))
	}

	return nil
}

func (t *LuaTarget) writeExports(b *strings.Builder) {
	for _, d := range t.program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Receiver != nil {
				if t.info.MethodVisible[decl] {
					fmt.Fprintf(b, "__wisp_module%s = %s\n", fieldAccess(funcName(decl)), funcName(decl))
				}
				continue
			}
			if !decl.Exported {
				continue
			}
			fmt.Fprintf(b, "__wisp_module%s = %s\n", fieldAccess(decl.Name), luaIdent(decl.Name))
		case *ast.ConstDecl:
			if !decl.Exported {
				continue
			}
			for _, v := range decl.Vars {
				fmt.Fprintf(b, "__wisp_module%s = %s\n", fieldAccess(v.Name), luaIdent(v.Name))
			}
		case *ast.VarDecl:
		}
	}

	parts := make([]string, 0, len(t.info.PartExports))
	for part := range t.info.PartExports {
		parts = append(parts, part)
	}
	sort.Strings(parts)
	for i, part := range parts {
		fmt.Fprintf(b, "local __part%d = require(%q)\n", i, ModuleID(part))
		for _, name := range t.info.PartExports[part] {
			fmt.Fprintf(b, "__wisp_module%s = __part%d%s\n", fieldAccess(name), i, fieldAccess(name))
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
	type exportVar struct {
		name  string
		boxed bool
	}

	var vars []exportVar
	for _, d := range t.program.Declarations {
		if vd, ok := d.(*ast.VarDecl); ok && vd.Exported {
			for k, v := range vd.Vars {
				boxed := false
				if syms := t.info.VarSymbols[vd]; k < len(syms) {
					boxed = t.isBoxed(syms[k])
				}
				vars = append(vars, exportVar{v.Name, boxed})
			}
		}
	}
	if len(vars) == 0 {
		return ""
	}

	ref := func(v exportVar) string {
		if v.boxed {
			return luaIdent(v.name) + "[0]"
		}
		return luaIdent(v.name)
	}

	var b strings.Builder
	b.WriteString("setmetatable(__wisp_module, {\n")
	b.WriteString("  __index = function(_, k)\n")
	for _, v := range vars {
		fmt.Fprintf(&b, "    if k == %q then return %s end\n", v.name, ref(v))
	}
	b.WriteString("  end,\n")
	b.WriteString("  __newindex = function(_, k, v)\n")
	for _, v := range vars {
		fmt.Fprintf(&b, "    if k == %q then %s = v return end\n", v.name, ref(v))
	}
	b.WriteString("    rawset(_, k, v)\n")
	b.WriteString("  end,\n")
	b.WriteString("})\n")
	return b.String()
}
