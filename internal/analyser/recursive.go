package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

func (a *Analyser) checkInfiniteTypes() {
	reported := map[*StructType]bool{}
	for _, d := range a.program.Declarations {
		var name string
		switch decl := d.(type) {
		case *ast.StructDecl:
			if len(decl.TypeParams) > 0 {
				continue
			}
			name = decl.Name
		case *ast.TypeDecl:
			if !decl.Data || len(decl.TypeParams) > 0 {
				continue
			}
			name = decl.Name
		default:
			continue
		}
		symbol, ok := a.scope.Resolve(name)
		if !ok {
			continue
		}
		root, ok := symbol.Type.(*StructType)
		if !ok || reported[root] {
			continue
		}
		path := containsItself(root)
		if path == nil {
			continue
		}
		for _, st := range path {
			reported[st] = true
		}
		a.currentFile = a.declFiles[d]
		a.errorf(d, "type %s has infinite size: it contains itself by value; use a pointer (*%s) for the recursive field", name, name)
	}
}

func containsItself(root *StructType) []*StructType {
	var path []*StructType
	visited := map[*StructType]bool{}
	var walkType func(t Type) bool
	var walkStruct func(st *StructType) bool
	walkStruct = func(st *StructType) bool {
		if st == root && len(path) > 0 {
			return true
		}
		if visited[st] {
			return false
		}
		visited[st] = true
		path = append(path, st)
		for _, name := range st.Order {
			if walkType(st.Fields[name]) {
				return true
			}
		}
		path = path[:len(path)-1]
		return false
	}
	walkType = func(t Type) bool {
		switch tp := t.(type) {
		case *StructType:
			return walkStruct(tp)
		case ArrayType:
			return walkType(tp.Element)
		case NamedType:
			return walkType(tp.Underlying)
		}
		return false
	}
	if walkStruct(root) {
		return path
	}
	return nil
}
