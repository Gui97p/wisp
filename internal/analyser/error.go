package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

func (a *Analyser) Errors() diag.List {
	if len(a.mangled) == 0 {
		return a.errors
	}
	shown := make(diag.List, len(a.errors))
	copy(shown, a.errors)
	for i := range shown {
		shown[i].Message = a.demangle(shown[i].Message)
		shown[i].Help = a.demangle(shown[i].Help)
		notes := make([]diag.Note, len(shown[i].Notes))
		for j, n := range shown[i].Notes {
			n.Message = a.demangle(n.Message)
			notes[j] = n
		}
		shown[i].Notes = notes
	}
	return shown
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

func (a *Analyser) errorAtName(fd *ast.FuncDecl, format string, args ...any) *diag.Diagnostic {
	if fd.NameAt.Line == 0 {
		return a.errorf(fd, format, args...)
	}
	return a.errorRange(fd.NameAt, format, args...)
}

func (a *Analyser) errorRange(at ast.Position, format string, args ...any) *diag.Diagnostic {
	return a.errors.Add(a.currentFile, at.Line, at.Col, at.EndLine, at.EndCol, format, args...)
}

func (a *Analyser) errorAtRef(node ast.Node, ref ast.TypeRef, format string, args ...any) *diag.Diagnostic {
	if ref.Line == 0 {
		return a.errorf(node, format, args...)
	}
	return a.errors.Add(a.currentFile, ref.Line, ref.Col, ref.EndLine, ref.EndCol, format, args...)
}

func (a *Analyser) errorAtEnd(node ast.Node, format string, args ...any) *diag.Diagnostic {
	endLine, endCol := node.EndPosition()
	return a.errors.Add(a.currentFile, endLine, endCol, endLine, endCol, format, args...)
}

func (a *Analyser) dropAllowedWarnings() {
	allowed := a.program.Allowed
	if len(allowed) == 0 {
		return
	}
	kept := a.errors[:0]
	for _, d := range a.errors {
		if d.Severity == diag.SeverityWarning {
			if set := allowed[d.File]; set["warnings"] || (d.Lint != "" && set[d.Lint]) {
				continue
			}
		}
		kept = append(kept, d)
	}
	a.errors = kept
}

func (a *Analyser) warnf(node ast.Node, format string, args ...any) *diag.Diagnostic {
	line, col := node.Position()
	endLine, endCol := node.EndPosition()
	return a.errors.Warn(a.currentFile, line, col, endLine, endCol, format, args...)
}

func (a *Analyser) error(node ast.Node, msg string) *diag.Diagnostic {
	return a.errorf(node, "%s", msg)
}

func (a *Analyser) errorAlreadyDeclared(node ast.Node, kind SymbolKind, name string) {
	var d *diag.Diagnostic
	if fd, ok := node.(*ast.FuncDecl); ok {
		d = a.errorAtName(fd, "%s %s already declared in this scope", kind, name)
	} else {
		d = a.errorf(node, "%s %s already declared in this scope", kind, name)
	}
	if prev, ok := a.scope.symbols[name]; ok {
		a.notePrevious(d, prev, "first declared here")
	}
}

func (a *Analyser) errorDeclaredAs(node ast.Node, name string, declared, got Type) {
	a.helpMismatch(a.errorf(node, "variable %s declared as %s, got %s", name, declared.String(), got.String()), got, declared)
}

func (a *Analyser) errorConstAssign(node ast.Node, name string) {
	d := a.errorf(node, "cannot assign to constant %s", name)
	if prev, ok := a.scope.Resolve(name); ok {
		a.notePrevious(d, prev, "declared as a constant here")
	}
}
