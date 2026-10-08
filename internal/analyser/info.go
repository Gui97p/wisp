package analyser

import (
	"github.com/Gui97p/wisp/internal/ast"
)

type Info struct {
	VarTypes      map[ast.Node][]Type
	VarSymbols    map[ast.Node][]*Symbol
	Types         map[ast.Expression]Type
	CallReturns   map[*ast.CallExpr][]Type
	FuncReturns   map[*ast.FuncDecl][]Type
	ErrorReturns  map[*ast.ReturnStmt]bool
	ErrorLiterals map[*ast.CallExpr]*ErrorLiteral
	Idents        map[*ast.IdentLiteral]*Symbol
	Members       map[*ast.MemberExpr]*MemberInfo
	MethodCalls   map[*ast.CallExpr]*MethodCall
	MethodVisible map[*ast.FuncDecl]bool
	Scopes        map[ast.Node]*Scope
	Reexports     map[*ast.ReexportDecl][]string
}

func NewInfo() *Info {
	return &Info{
		VarTypes:      make(map[ast.Node][]Type),
		VarSymbols:    make(map[ast.Node][]*Symbol),
		Types:         make(map[ast.Expression]Type),
		CallReturns:   make(map[*ast.CallExpr][]Type),
		FuncReturns:   make(map[*ast.FuncDecl][]Type),
		ErrorReturns:  make(map[*ast.ReturnStmt]bool),
		ErrorLiterals: make(map[*ast.CallExpr]*ErrorLiteral),
		Idents:        make(map[*ast.IdentLiteral]*Symbol),
		Members:       make(map[*ast.MemberExpr]*MemberInfo),
		MethodCalls:   make(map[*ast.CallExpr]*MethodCall),
		MethodVisible: make(map[*ast.FuncDecl]bool),
		Scopes:        make(map[ast.Node]*Scope),
		Reexports:     make(map[*ast.ReexportDecl][]string),
	}
}

type MethodCall struct {
	Owner           string
	Module          string
	Func            *FuncType
	PointerReceiver bool
	ObjectIsPointer bool
	Receiver        ast.Expression
}

type MemberKind int

const (
	MemberField MemberKind = iota
	MemberModule
	MemberErrorField
)

type MemberInfo struct {
	Kind      MemberKind
	Struct    *StructType
	ByPointer bool
}

type ErrorLiteral struct {
	Domain  string
	Code    int64
	Message string
	File    string
	Line    int
}
