package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

type Info struct {
	VarTypes map[ast.Node][]Type
	Types    map[ast.Expression]Type
	Idents   map[*ast.IdentLiteral]*Symbol
	Scopes   map[ast.Node]*Scope
}

func NewInfo() *Info {
	return &Info{
		VarTypes: make(map[ast.Node][]Type),
		Types:    make(map[ast.Expression]Type),
		Idents:   make(map[*ast.IdentLiteral]*Symbol),
		Scopes:   make(map[ast.Node]*Scope),
	}
}
