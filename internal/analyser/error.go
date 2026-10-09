package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

func (a *Analyser) Errors() diag.List {
	return a.errors
}

func (a *Analyser) HasErrors() bool {
	return a.errors.HasErrors()
}

func (a *Analyser) HasDiagnostics() bool {
	return len(a.errors) > 0
}

func (a *Analyser) errorf(node ast.Node, format string, args ...any) *diag.Diagnostic {
	line, col := node.Position()
	endLine, endCol := node.EndPosition()
	return a.errors.Add(a.currentFile, line, col, endLine, endCol, format, args...)
}

func (a *Analyser) warnf(node ast.Node, format string, args ...any) *diag.Diagnostic {
	line, col := node.Position()
	endLine, endCol := node.EndPosition()
	return a.errors.Warn(a.currentFile, line, col, endLine, endCol, format, args...)
}

func (a *Analyser) noteAt(d *diag.Diagnostic, node ast.Node, format string, args ...any) {
	line, col := node.Position()
	endLine, endCol := node.EndPosition()
	d.Note(a.currentFile, line, col, endLine, endCol, format, args...)
}

func (a *Analyser) error(node ast.Node, msg string) *diag.Diagnostic {
	return a.errorf(node, "%s", msg)
}

func (a *Analyser) errorAlreadyDeclared(node ast.Node, kind SymbolKind, name string) {
	d := a.errorf(node, "%s %s already declared in this scope", kind, name)
	if prev, ok := a.scope.symbols[name]; ok {
		a.notePrevious(d, prev, "first declared here")
	}
}

func (a *Analyser) errorDeclaredAs(node ast.Node, name string, declared, got Type) {
	a.errorf(node, "variable %s declared as %s, got %s", name, declared.String(), got.String())
}

func (a *Analyser) errorConstAssign(node ast.Node, name string) {
	d := a.errorf(node, "cannot assign to constant %s", name)
	if prev, ok := a.scope.Resolve(name); ok {
		a.notePrevious(d, prev, "declared as a constant here")
	}
}
