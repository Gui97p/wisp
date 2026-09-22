package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

func (a *Analyser) Errors() diag.List {
	return a.errors
}

func (a *Analyser) HasErrors() bool {
	return len(a.errors) > 0
}

func (a *Analyser) errorf(node ast.Node, format string, args ...any) {
	line, col := node.Position()
	a.errors.Add(a.currentFile, line, col, format, args...)
}

func (a *Analyser) error(node ast.Node, msg string) {
	a.errorf(node, "%s", msg)
}

func (a *Analyser) errorAlreadyDeclared(node ast.Node, kind SymbolKind, name string) {
	a.errorf(node, "%s %s already declared in this scope", kind, name)
}

func (a *Analyser) errorDeclaredAs(node ast.Node, name string, declared, got Type) {
	a.errorf(node, "variable %s declared as %s, got %s", name, declared.String(), got.String())
}

func (a *Analyser) errorConstAssign(node ast.Node, name string) {
	a.errorf(node, "cannot assign to constant %s", name)
}
