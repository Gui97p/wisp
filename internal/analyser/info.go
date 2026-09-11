package analyser

import "github.com/Gui97p/wisp/internal/ast"

type Info struct {
	Types  map[ast.Expression]Type
	Idents map[*ast.IdentLiteral]*Symbol
	Scopes map[ast.Node]*Scope
}

func NewInfo() *Info {
	return &Info{
		Types:  make(map[ast.Expression]Type),
		Idents: make(map[*ast.IdentLiteral]*Symbol),
		Scopes: make(map[ast.Node]*Scope),
	}
}
