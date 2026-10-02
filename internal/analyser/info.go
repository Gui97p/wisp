package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

type Info struct {
	VarTypes    map[ast.Node][]Type
	VarSymbols  map[ast.Node][]*Symbol
	Types       map[ast.Expression]Type
	CallReturns map[*ast.CallExpr][]Type
	Idents      map[*ast.IdentLiteral]*Symbol
	Scopes      map[ast.Node]*Scope
	Reexports   map[*ast.ReexportDecl][]string
}

func NewInfo() *Info {
	return &Info{
		VarTypes:    make(map[ast.Node][]Type),
		VarSymbols:  make(map[ast.Node][]*Symbol),
		Types:       make(map[ast.Expression]Type),
		CallReturns: make(map[*ast.CallExpr][]Type),
		Idents:      make(map[*ast.IdentLiteral]*Symbol),
		Scopes:      make(map[ast.Node]*Scope),
		Reexports:   make(map[*ast.ReexportDecl][]string),
	}
}
