package analyser

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/diag"
)

type Info struct {
	Module           *Scope
	VarTypes         map[ast.Node][]Type
	VarSymbols       map[ast.Node][]*Symbol
	Types            map[ast.Expression]Type
	CallReturns      map[*ast.CallExpr][]Type
	FuncReturns      map[*ast.FuncDecl][]Type
	ErrorReturns     map[*ast.ReturnStmt]bool
	ErrorLiterals    map[*ast.CallExpr]*ErrorLiteral
	Idents           map[*ast.IdentLiteral]*Symbol
	Members          map[*ast.MemberExpr]*MemberInfo
	GenericFuncs     map[*ast.GenericExpr]*Symbol
	MethodCalls      map[*ast.CallExpr]*MethodCall
	MethodVisible    map[*ast.FuncDecl]bool
	UntilSeesBody    map[*ast.LoopStmt]bool
	EnumValues       map[ast.Expression]EnumValue
	Scopes           map[ast.Node]*Scope
	PartExports      map[string][]string
	SizeOf           map[*ast.CallExpr]Type
	ConstValues      map[ast.Expression]ConstValue
	Folds            map[ast.Expression]ConstValue
	Inlines          map[*ast.CallExpr]*InlineCall
	InlineReturns    map[*ast.ReturnStmt]*InlineCall
	InlinePropagates map[*ast.PropagateExpr]*InlineCall
	ConstRefs        map[*ast.ConstRef]ConstRefInfo
	Boxes            map[ast.Expression]Type
	TypeCases        map[*ast.BinaryExpr]*TypeCase
	Narrowed         map[*Symbol]*Symbol
	SwitchValues     map[*ast.SwitchExpr][]Type
	Variants         map[ast.Expression]*VariantInfo
	SumTests         map[*ast.BinaryExpr]*SumTest
	Makes            map[*ast.CallExpr]*MakeInfo
	Natives          map[*ast.NativeStmt]*NativeInfo
	LayoutValues     map[*ast.CallExpr]int
	Statics          map[*ast.BlockExpr]ConstValue
	StaticAsserts    map[*ast.CallExpr]bool
	Resizes          map[*ast.CallExpr]*ResizeInfo
}

type MakeInfo struct {
	Elem  Type
	Count ast.Expression
	Size  *ast.CallExpr
}

type ResizeInfo struct {
	Elem  Type
	Slice ast.Expression
	Count ast.Expression
}

type VariantInfo struct {
	Sum     *StructType
	Variant SumVariant
}

type SumTest struct {
	Sum     *StructType
	Variant SumVariant
	Subject ast.Expression
}

type TypeCase struct {
	Type  Type
	Group bool
}

func NewInfo() *Info {
	return &Info{
		VarTypes:         make(map[ast.Node][]Type),
		ConstValues:      make(map[ast.Expression]ConstValue),
		Folds:            make(map[ast.Expression]ConstValue),
		Inlines:          make(map[*ast.CallExpr]*InlineCall),
		InlineReturns:    make(map[*ast.ReturnStmt]*InlineCall),
		InlinePropagates: make(map[*ast.PropagateExpr]*InlineCall),
		ConstRefs:        make(map[*ast.ConstRef]ConstRefInfo),
		SizeOf:           make(map[*ast.CallExpr]Type),
		VarSymbols:       make(map[ast.Node][]*Symbol),
		Types:            make(map[ast.Expression]Type),
		CallReturns:      make(map[*ast.CallExpr][]Type),
		FuncReturns:      make(map[*ast.FuncDecl][]Type),
		ErrorReturns:     make(map[*ast.ReturnStmt]bool),
		ErrorLiterals:    make(map[*ast.CallExpr]*ErrorLiteral),
		Idents:           make(map[*ast.IdentLiteral]*Symbol),
		Members:          make(map[*ast.MemberExpr]*MemberInfo),
		MethodCalls:      make(map[*ast.CallExpr]*MethodCall),
		MethodVisible:    make(map[*ast.FuncDecl]bool),
		UntilSeesBody:    make(map[*ast.LoopStmt]bool),
		EnumValues:       make(map[ast.Expression]EnumValue),
		Scopes:           make(map[ast.Node]*Scope),
		PartExports:      make(map[string][]string),
		Boxes:            make(map[ast.Expression]Type),
		TypeCases:        make(map[*ast.BinaryExpr]*TypeCase),
		Narrowed:         make(map[*Symbol]*Symbol),
		SwitchValues:     make(map[*ast.SwitchExpr][]Type),
		GenericFuncs:     make(map[*ast.GenericExpr]*Symbol),
		Variants:         make(map[ast.Expression]*VariantInfo),
		Natives:          make(map[*ast.NativeStmt]*NativeInfo),
		LayoutValues:     make(map[*ast.CallExpr]int),
		Statics:          make(map[*ast.BlockExpr]ConstValue),
		StaticAsserts:    make(map[*ast.CallExpr]bool),
		SumTests:         make(map[*ast.BinaryExpr]*SumTest),
		Makes:            make(map[*ast.CallExpr]*MakeInfo),
		Resizes:          make(map[*ast.CallExpr]*ResizeInfo),
	}
}

type EnumValue struct {
	Type  NamedType
	Name  string
	Value int64
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
	MemberEnum
	MemberVariant
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
	Col     int
	EndLine int
	EndCol  int
	Block   string
}

func (i *Info) RenderErrorBlocks(buffers map[string][]byte) {
	for _, lit := range i.ErrorLiterals {
		source, ok := buffers[lit.File]
		if !ok {
			continue
		}
		header := fmt.Sprintf("error[%s]: %s", lit.Domain, lit.Message)
		lit.Block = diag.RuntimeBlock(header, lit.File, source, lit.Line, lit.Col, lit.EndLine, lit.EndCol, "raised here")
	}
}
