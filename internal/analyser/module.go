package analyser

import "github.com/Gui97p/wisp/internal/ast"

type ModuleInfo struct {
	Exports map[string]*Symbol
}

func (a *Analyser) Exports() map[string]*Symbol {
	exports := map[string]*Symbol{}

	for _, d := range a.program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			if decl.Receiver != nil || !decl.Exported {
				continue
			}
			if sym, ok := a.scope.Resolve(decl.Name); ok {
				exports[decl.Name] = sym
			}
		case *ast.StructDecl:
			if !decl.Exported {
				continue
			}
			if sym, ok := a.scope.Resolve(decl.Name); ok {
				exports[decl.Name] = sym
			}
		case *ast.TypeDecl:
			if !decl.Exported {
				continue
			}
			if sym, ok := a.scope.Resolve(decl.Name); ok {
				exports[decl.Name] = sym
			}
		case *ast.ConstDecl:
			if !decl.Exported {
				continue
			}
			for _, v := range decl.Vars {
				if sym, ok := a.scope.Resolve(v.Name); ok {
					exports[v.Name] = sym
				}
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
		imp, ok := d.(*ast.ImportDecl)
		if !ok {
			continue
		}
		mod, ok := a.modules[imp.Path]
		if !ok {
			a.errorf(imp, "unresolved import %q", imp.Path)
			continue
		}
		symbol := &Symbol{Name: imp.Alias, Kind: MODULE, Type: &ModuleType{Exports: mod.Exports}}
		if !a.scope.Define(symbol) {
			a.errorAlreadyDeclared(imp, MODULE, imp.Alias)
		}
	}
}
