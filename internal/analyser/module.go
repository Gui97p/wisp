package analyser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

type ModuleInfo struct {
	Exports map[string]*Symbol
}

func (a *Analyser) Exports() map[string]*Symbol {
	exports := map[string]*Symbol{}

	reach := ast.ReachNone
	add := func(node ast.Node, name string, sym *Symbol) {
		if reach != ast.ReachNone {
			sym.ScopeDir = reachDir(reach, sym.File)
		}
		if _, exists := exports[name]; exists {
			a.errorf(node, "%s already declared in exported scope", name)
			return
		}
		exports[name] = sym
	}

	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		reach = declReach(d)
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Receiver != nil || !decl.Exported {
				continue
			}
			if sym, ok := a.scope.Resolve(decl.Name); ok {
				add(decl, decl.Name, sym)
			}
		case *ast.StructDecl:
			if !decl.Exported {
				continue
			}
			if sym, ok := a.scope.Resolve(decl.Name); ok {
				add(decl, decl.Name, sym)
			}
		case *ast.TypeDecl:
			if !decl.Exported {
				continue
			}
			if sym, ok := a.scope.Resolve(decl.Name); ok {
				add(decl, decl.Name, sym)
			}
		case *ast.ConstDecl:
			if !decl.Exported {
				continue
			}
			for _, v := range decl.Vars {
				if sym, ok := a.scope.Resolve(v.Name); ok {
					add(decl, v.Name, sym)
				}
			}
		case *ast.VarDecl:
			if !decl.Exported {
				continue
			}
			for _, v := range decl.Vars {
				if sym, ok := a.scope.Resolve(v.Name); ok {
					add(decl, v.Name, sym)
				}
			}
		case *ast.ReexportDecl:
			if decl.Reach != ast.ReachNone {
				a.errorf(decl, "%s cannot be reexported with a restricted reach", decl.Name)
				continue
			}
			sym, ok := a.scope.Resolve(decl.Name)
			if !ok {
				a.errorf(decl, "identifier %s not declared in this scope", decl.Name)
				continue
			}
			if _, ok := sym.Type.(*ModuleType); !ok {
				a.errorf(decl, "%s is not an imported module", decl.Name)
				continue
			}
			if decl.Alias != "" {
				add(decl, decl.Alias, sym)
				continue
			}
			mt := sym.Type.(*ModuleType)
			for name, exp := range mt.Exports {
				add(decl, name, exp)
				a.info.Reexports[decl] = append(a.info.Reexports[decl], name)
			}
		}
	}

	return exports
}

func declReach(d ast.Declaration) int {
	switch decl := d.(type) {
	case *ast.FuncDecl:
		return decl.Reach
	case *ast.StructDecl:
		return decl.Reach
	case *ast.TypeDecl:
		return decl.Reach
	case *ast.ConstDecl:
		return decl.Reach
	case *ast.VarDecl:
		return decl.Reach
	}
	return ast.ReachNone
}

func reachDir(reach int, file string) string {
	if reach == ast.ReachNone || file == "" {
		return ""
	}
	dir := filepath.Dir(file)
	if reach == ast.ReachShared {
		dir = filepath.Dir(dir)
	}
	return dir
}

func insideDir(file, dir string) bool {
	if file == "" || dir == "" {
		return true
	}
	rel, err := filepath.Rel(dir, filepath.Dir(file))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func visibleExports(exports map[string]*Symbol, importer string) (map[string]*Symbol, map[string]string) {
	var hidden map[string]string
	var visible map[string]*Symbol
	for name, sym := range exports {
		if sym.ScopeDir != "" && !insideDir(importer, sym.ScopeDir) {
			if visible == nil {
				visible = make(map[string]*Symbol, len(exports))
				for n, s := range exports {
					visible[n] = s
				}
				hidden = map[string]string{}
			}
			delete(visible, name)
			hidden[name] = filepath.Base(sym.ScopeDir)
		}
	}
	if visible == nil {
		return exports, nil
	}
	return visible, hidden
}

type ModuleType struct {
	Exports map[string]*Symbol
	Hidden  map[string]string
}

func (m *ModuleType) String() string {
	return "module"
}

func (m *ModuleType) Equals(other Type) bool {
	_, ok := other.(*ModuleType)
	return ok
}

func (a *Analyser) registerImports() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		imp, ok := d.(*ast.ImportDecl)
		if !ok {
			continue
		}
		line, col := imp.Position()
		mod, ok := a.modules[imp.Path]
		message, known := a.importErrors[imp.Path]
		if !ok || known {
			if !known {
				message = fmt.Sprintf("unresolved import %q", imp.Path)
			}
			message, help, _ := strings.Cut(message, "\n")
			d := a.error(imp, message)
			if help != "" {
				d.WithHelp("%s", help)
			}
			a.scope.Define(&Symbol{Name: imp.Alias, Kind: MODULE, Type: InvalidType{}, Line: line, Col: col, File: a.currentFile, Used: true})
			continue
		}
		if len(mod.Exports) == 0 {
			a.error(imp, fmt.Sprintf("module %q exports nothing", imp.Path)).WithHelp("a package entry with no exports only marks its folder as a package")
			a.scope.Define(&Symbol{Name: imp.Alias, Kind: MODULE, Type: InvalidType{}, Line: line, Col: col, File: a.currentFile, Used: true})
			continue
		}
		exports, hidden := visibleExports(mod.Exports, a.currentFile)
		if len(exports) == 0 {
			folder := ""
			for _, f := range hidden {
				folder = f
				break
			}
			a.error(imp, fmt.Sprintf("module %q is internal to %q", imp.Path, folder)).WithHelp("everything it exports is only visible under the folder %s", folder)
			a.scope.Define(&Symbol{Name: imp.Alias, Kind: MODULE, Type: InvalidType{}, Line: line, Col: col, File: a.currentFile, Used: true})
			continue
		}
		symbol := &Symbol{Name: imp.Alias, Kind: MODULE, Type: &ModuleType{Exports: exports, Hidden: hidden}, Line: line, Col: col, File: a.currentFile}
		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(imp, MODULE, imp.Alias)
		}
	}
}
