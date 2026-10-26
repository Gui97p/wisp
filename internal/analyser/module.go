package analyser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/folders"
)

type ModuleInfo struct {
	Exports map[string]*Symbol
	Pending bool
}

func (m *ModuleInfo) Replace(exports map[string]*Symbol) {
	if m.Exports == nil {
		m.Exports = make(map[string]*Symbol, len(exports))
	}
	clear(m.Exports)
	for name, sym := range exports {
		m.Exports[name] = sym
	}
}

func (a *Analyser) Exports() map[string]*Symbol {
	return a.buildExports(false)
}

func (a *Analyser) buildExports(quiet bool) map[string]*Symbol {
	exports := map[string]*Symbol{}

	reach := ast.ReachNone
	add := func(node ast.Node, name string, sym *Symbol) {
		if reach != ast.ReachNone {
			sym.ScopeDir = reachDir(reach, sym.File)
		}
		if _, exists := exports[name]; exists {
			if !quiet {
				a.errorf(node, "%s already declared in exported scope", name)
			}
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
			if quiet {
				continue
			}
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

func MethodAccessible(recv Type, ft *FuncType, module, file string) bool {
	_, _, ownerExported := methodOwner(recv)
	if ft.Module == module {
		return true
	}
	if !(ft.Exported && ownerExported) {
		return false
	}
	return ft.InternalDir == "" || insideDir(file, ft.InternalDir)
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
	return folders.Inside(file, dir)
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
		mt := &ModuleType{}
		symbol := &Symbol{Name: imp.Alias, Kind: MODULE, Type: mt, Line: line, Col: col, File: a.currentFile}
		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(imp, MODULE, imp.Alias)
		}
		link := &importLink{imp: imp, symbol: symbol, mt: mt, target: mod, file: a.currentFile}
		link.pendingAtStart = mod.Pending
		a.links = append(a.links, link)
		if !mod.Pending {
			a.applyLink(link, true)
		}
	}
}

type importLink struct {
	imp    *ast.ImportDecl
	symbol *Symbol
	mt     *ModuleType
	target *ModuleInfo
	file   string

	pendingAtStart bool
}

func (a *Analyser) applyLink(link *importLink, check bool) {
	exports, hidden := visibleExports(link.target.Exports, link.file)
	link.mt.Exports, link.mt.Hidden = exports, hidden
	if !check {
		return
	}
	a.currentFile = link.file
	switch {
	case len(link.target.Exports) == 0:
		a.error(link.imp, fmt.Sprintf("module %q exports nothing", link.imp.Path)).WithHelp("a package entry with no exports only marks its folder as a package")
		link.symbol.Type = InvalidType{}
		link.symbol.Used = true
	case len(exports) == 0:
		folder := ""
		for _, f := range hidden {
			folder = f
			break
		}
		a.error(link.imp, fmt.Sprintf("module %q is internal to %q", link.imp.Path, folder)).WithHelp("everything it exports is only visible under the folder %s", folder)
		link.symbol.Type = InvalidType{}
		link.symbol.Used = true
	}
}

func PendingExports(exports map[string]*ModuleInfo, paths []string) {
	if len(paths) < 2 {
		return
	}
	for _, path := range paths {
		exports[path] = &ModuleInfo{Exports: map[string]*Symbol{}, Pending: true}
	}
}

func SetExports(exports map[string]*ModuleInfo, path string, module map[string]*Symbol) {
	if existing, ok := exports[path]; ok {
		existing.Replace(module)
		existing.Pending = false
		return
	}
	exports[path] = &ModuleInfo{Exports: module}
}
