package analyser

import "github.com/Gui97p/wisp/internal/ast"

type ModuleInfo struct {
	Exports map[string]*Symbol
}

func (a *Analyser) Exports() map[string]*Symbol {
	exports := map[string]*Symbol{}

	add := func(node ast.Node, name string, sym *Symbol) {
		if _, exists := exports[name]; exists {
			a.errorf(node, "%s already declared in exported scope", name)
			return
		}
		exports[name] = sym
	}

	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
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
		case *ast.ReexportDecl:
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

type ModuleType struct {
	Exports map[string]*Symbol
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
		mod, ok := a.modules[imp.Path]
		if !ok {
			a.errorf(imp, "unresolved import %q", imp.Path)
			continue
		}
		if imp.Implicit {
			for _, sym := range mod.Exports {
				shared := *sym
				shared.Via = imp.Alias
				a.prelude.Define(&shared)
			}
			continue
		}
		line, col := imp.Position()
		symbol := &Symbol{Name: imp.Alias, Kind: MODULE, Type: &ModuleType{Exports: mod.Exports}, Line: line, Col: col, File: a.currentFile}
		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(imp, MODULE, imp.Alias)
		}
	}
}
