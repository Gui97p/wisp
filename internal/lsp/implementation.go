package lsp

import (
	"strings"

	"github.com/Gui97p/wisp/internal/analyser"
	"github.com/Gui97p/wisp/internal/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func implementation(uri protocol.DocumentUri, pos protocol.Position) []protocol.Location {
	path := uriToPath(uri)
	line, col := int(pos.Line)+1, int(pos.Character)+1

	pa, err := analyseProject(path)
	if err != nil || pa == nil {
		return nil
	}
	m := pa.moduleOf(path)
	if m == nil {
		return nil
	}
	own := declarationsOf(m.merged, m.declFiles, path)

	it, method := interfaceAtCursor(m, own, path, line, col)
	if it == nil {
		return nil
	}

	var locations []protocol.Location
	for _, other := range pa.modules {
		for _, d := range other.merged.Declarations {
			var name string
			var implements []ast.TypeRef
			var generic bool
			switch decl := d.(type) {
			case *ast.StructDecl:
				name, implements, generic = decl.Name, decl.Implements, len(decl.TypeParams) > 0
			case *ast.TypeDecl:
				name, implements, generic = decl.Name, decl.Implements, len(decl.TypeParams) > 0
			default:
				continue
			}
			if len(implements) == 0 || !implementsInterface(other, name, implements, generic, it) {
				continue
			}
			file := other.declFiles[d]
			if method == "" {
				line, col := d.Position()
				locations = append(locations, pointLocation(file, line, col))
				continue
			}
			for _, candidate := range other.merged.Declarations {
				fd, ok := candidate.(*ast.FuncDecl)
				if !ok || fd.Receiver == nil || fd.Name != method || fd.Receiver.Type.Name != name {
					continue
				}
				line, col := fd.Position()
				locations = append(locations, pointLocation(other.declFiles[candidate], line, col))
			}
		}
	}
	return locations
}

func implementsInterface(m moduleAnalysis, name string, implements []ast.TypeRef, generic bool, it *analyser.InterfaceType) bool {
	if generic {
		base, _, _ := strings.Cut(it.Name, "<")
		for _, ref := range implements {
			if ref.Name == base {
				return true
			}
		}
		return false
	}
	symbol, ok := m.info.Module.Resolve(name)
	return ok && analyser.Implements(symbol.Type, it)
}

func interfaceAtCursor(m *moduleAnalysis, own *ast.Program, path string, line, col int) (*analyser.InterfaceType, string) {
	if expr := exprInProgram(own, line, col); expr != nil {
		if member, ok := expr.(*ast.MemberExpr); ok {
			for call := range m.info.MethodCalls {
				if name, found := methodName(m.info.MethodCalls[call], m.info, member); found && m.info.MethodCalls[call].Iface != nil {
					return m.info.MethodCalls[call].Iface, name
				}
			}
		}
	}

	for _, decl := range own.Declarations {
		d, ok := decl.(*ast.InterfaceDecl)
		if !ok {
			continue
		}
		symbol, found := m.info.Module.Resolve(d.Name)
		if !found {
			continue
		}
		it, isInterface := symbol.Type.(*analyser.InterfaceType)
		if !isInterface {
			continue
		}
		for _, method := range d.Methods {
			if positionContains(method.NameAt, line, col) {
				return it, method.Name
			}
		}
		if declLine, _ := d.Position(); declLine == line && wordUnderCursor(fileBuffer[path], line, col) == d.Name {
			return it, ""
		}
	}

	if ref := typeRefAt(own, line, col); ref != nil {
		if symbol := typeRefSymbol(ref, m.info, line, col); symbol != nil && symbol.Kind == analyser.INTERFACE {
			if it, ok := symbol.Type.(*analyser.InterfaceType); ok {
				return it, ""
			}
		}
	}
	return nil, ""
}
