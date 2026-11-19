package analyser

import "github.com/Gui97p/wisp/internal/ast"

func (a *Analyser) rejectInterfaces() {
	for _, d := range a.program.Declarations {
		a.currentFile = a.declFiles[d]
		switch decl := d.(type) {
		case *ast.InterfaceDecl:
			a.errorf(decl, "interfaces are parsed but not analysed yet")
			a.rejectConstraints(decl.TypeParams)
		case *ast.StructDecl:
			a.rejectImplements(decl.Implements)
			a.rejectConstraints(decl.TypeParams)
		case *ast.TypeDecl:
			a.rejectImplements(decl.Implements)
			a.rejectConstraints(decl.TypeParams)
		case *ast.FuncDecl:
			a.rejectConstraints(decl.TypeParams)
		}
	}
}

func (a *Analyser) rejectImplements(refs []ast.TypeRef) {
	for _, ref := range refs {
		a.errorRange(ast.Position{Line: ref.Line, Col: ref.Col, EndLine: ref.EndLine, EndCol: ref.EndCol}, "implements is parsed but not checked yet")
	}
}

func (a *Analyser) rejectConstraints(params []ast.TypeParam) {
	for _, param := range params {
		if param.Constraint != nil {
			a.errorRange(param.Constraint.At, "the constraint of a type parameter is parsed but not checked yet")
		}
	}
}
