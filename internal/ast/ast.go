package ast

import (
	"fmt"
	"strings"
)

type Node interface {
	Tree(indent string) string
	Position() (int, int)
	EndPosition() (int, int)
	SetPos(line, col int)
	SetEndPos(line, col int)
}

type Declaration interface {
	Node
	decl()
}

type Statement interface {
	Node
	stmt()
}

type Expression interface {
	Node
	expr()
}

type NodePos struct {
	Line, Col       int
	EndLine, EndCol int
}

func (p NodePos) Position() (int, int) {
	return p.Line, p.Col
}

func (p NodePos) EndPosition() (int, int) {
	return p.EndLine, p.EndCol
}

func (p *NodePos) SetPos(line, col int) {
	p.Line, p.Col = line, col
}

func (p *NodePos) SetEndPos(line, col int) {
	p.EndLine, p.EndCol = line, col
}

type Program struct {
	Declarations []Declaration
}

func (p *Program) Tree(indent string) string {
	var b strings.Builder

	b.WriteString("Program\n")

	for i, decl := range p.Declarations {
		if i == len(p.Declarations)-1 {
			b.WriteString(decl.Tree("└─ "))
		} else {
			b.WriteString(decl.Tree("├─ "))
		}
	}

	return b.String()
}

type ArrayDim struct {
	IsSpan bool
	Size   int64
}

type TypeRef struct {
	Name         string
	PointerDepth int

	Dims []ArrayDim

	IsMap    bool
	MapKey   *TypeRef
	MapValue *TypeRef

	IsFunc      bool
	FuncParams  []TypeRef
	FuncReturns []TypeRef

	Fallible bool
}

func (t *TypeRef) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sTypeRef\n", indent)

	fmt.Fprintf(&b, "%s├─ Name\n%s│  %s\n", indent, indent, t.Name)
	fmt.Fprintf(&b, "%s├─ PointerDepth\n%s│  %d\n", indent, indent, t.PointerDepth)

	for k, dim := range t.Dims {
		fmt.Fprintf(&b, "%s├─ ArrayDim(%d)\n", indent, k)
		if dim.IsSpan {
			fmt.Fprintf(&b, "%s│  └─ IsSpan\n%s│     true\n", indent, indent)
		} else {
			fmt.Fprintf(&b, "%s│  └─ ArraySize\n%s│     %d\n", indent, indent, dim.Size)
		}
	}

	fmt.Fprintf(&b, "%s└─ IsMap\n", indent)
	if t.IsMap {
		fmt.Fprintf(&b, "%s   true\n", indent)

		b.WriteString(indent)
		b.WriteString("   ├─ MapKey\n")
		b.WriteString(t.MapKey.Tree(indent + "   │  "))

		b.WriteString(indent)
		b.WriteString("   └─ MapValue\n")
		b.WriteString(t.MapValue.Tree(indent + "      "))
	} else {
		fmt.Fprintf(&b, "%s   false\n", indent)
	}

	return b.String()
}

type Param struct {
	Name     string
	Type     TypeRef
	Variadic bool

	Line, Col       int
	EndLine, EndCol int
}

func (p *Param) Tree(indent string) string {
	var b strings.Builder

	if p.Type.Name != "" {
		b.WriteString(p.Type.Tree(indent))
	}
	if p.Name != "" {
		fmt.Fprintf(&b, "%sName: %s\n", indent, p.Name)
	}

	return b.String()
}
