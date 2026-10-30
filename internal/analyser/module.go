package analyser

import (
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/folders"
)

type ModuleInfo struct {
	Exports map[string]*Symbol
	Pending bool

	own   map[string]*Symbol
	Parts []*ModuleInfo
	Entry *ModuleInfo
}

func (m *ModuleInfo) Replace(exports map[string]*Symbol) {
	m.own = make(map[string]*Symbol, len(exports))
	maps.Copy(m.own, exports)
	m.refresh()
	if m.Entry != nil {
		m.Entry.refresh()
	}
}

func (m *ModuleInfo) refresh() {
	if m.Exports == nil {
		m.Exports = make(map[string]*Symbol, len(m.own))
	}
	clear(m.Exports)
	for name, sym := range m.own {
		m.Exports[name] = sym
	}
	for _, part := range m.Parts {
		for name, sym := range part.own {
			if _, taken := m.Exports[name]; !taken {
				m.Exports[name] = sym
			}
		}
	}
}

func infoFor(exports map[string]*ModuleInfo, path string) *ModuleInfo {
	info, ok := exports[path]
	if !ok {
		info = &ModuleInfo{Exports: map[string]*Symbol{}}
		exports[path] = info
	}
	return info
}

func LinkPackage(exports map[string]*ModuleInfo, entry string, parts []string) {
	entryInfo := infoFor(exports, entry)
	entryInfo.Parts = entryInfo.Parts[:0]
	for _, part := range parts {
		partInfo := infoFor(exports, part)
		partInfo.Entry = entryInfo
		entryInfo.Parts = append(entryInfo.Parts, partInfo)
	}
}

func (a *Analyser) Exports() map[string]*Symbol {
	if a.exports != nil {
		return a.exports
	}
	return a.buildExports(false)
}

func (a *Analyser) checkPartConflicts() {
	if len(a.parts) == 0 {
		return
	}

	owners := map[string]string{}
	for name := range a.exports {
		owners[name] = a.module
	}
	for _, part := range a.parts {
		info, ok := a.modules[part]
		if !ok {
			continue
		}
		names := make([]string, 0, len(info.own))
		for name := range info.own {
			names = append(names, name)
		}
		sort.Strings(names)

		var visible []string
		for _, name := range names {
			if other, taken := owners[name]; taken {
				file, line, col := a.entryPosition(other, name)
				a.errors.Add(file, line, col, line, col+len(name), "%s is public in %s and in %s", name, part, other)
				continue
			}
			owners[name] = part
			visible = append(visible, name)
		}
		a.info.PartExports[part] = visible
	}
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
				a.errorf(node, "%s already declared in the public scope", name)
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
		if a.repeatsImport(imp) {
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

func (a *Analyser) repeatsImport(imp *ast.ImportDecl) bool {
	for _, l := range a.links {
		if l.imp.Alias == imp.Alias && l.imp.Path == imp.Path && l.file != a.currentFile {
			return true
		}
	}
	return false
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
		infoFor(exports, path).Pending = true
	}
}

func SetExports(exports map[string]*ModuleInfo, path string, module map[string]*Symbol) {
	info := infoFor(exports, path)
	info.Replace(module)
	info.Pending = false
}

func (a *Analyser) entryPosition(owner, name string) (string, int, int) {
	if sym, ok := a.exports[name]; ok && owner == a.module {
		return sym.File, sym.Line, sym.Col
	}
	file := ""
	for _, d := range a.program.Declarations {
		file = a.declFiles[d]
		break
	}
	return file, 1, 1
}
