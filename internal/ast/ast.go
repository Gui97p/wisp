package ast

import (
	"fmt"
	"strings"
)

type Node interface {
	Tree(indent string) string
	Position() (int, int)
	SetPos(line, col int)
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
	Line, Col int
}

func (p NodePos) Position() (int, int) {
	return p.Line, p.Col
}

func (p *NodePos) SetPos(line, col int) {
	p.Line, p.Col = line, col
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
}

func (t *TypeRef) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sTypeRef\n", indent)

	b.WriteString(indent)
	b.WriteString("└─ Name\n")
	b.WriteString(indent)
	b.WriteString("│  ")
	b.WriteString(t.Name)
	b.WriteRune('\n')

	b.WriteString(indent)
	b.WriteString("└─ PointerDepth\n")
	b.WriteString(indent)
	b.WriteString("│  ")
	fmt.Fprint(&b, t.PointerDepth)
	b.WriteRune('\n')

	for k, dim := range t.Dims {
		b.WriteString(indent)
		fmt.Fprintf(&b, "└─ ArrayDim(%d)\n", k)
		b.WriteString(indent)
		b.WriteString("│  ")
		if dim.IsSpan {
			b.WriteString(indent)
			b.WriteString("└─ IsSpan\n")
			b.WriteString(indent)
			b.WriteString("│  true")
		} else {
			b.WriteString(indent)
			b.WriteString("└─ ArraySize\n")
			b.WriteString(indent)
			b.WriteString("│  ")
			fmt.Fprint(&b, dim.Size)
		}
		b.WriteRune('\n')
	}

	b.WriteString(indent)
	b.WriteString("└─ IsMap\n")
	b.WriteString(indent)
	b.WriteString("│  ")
	if t.IsMap {
		b.WriteString("true\n")

		b.WriteString(indent)
		b.WriteString("└─ MapKey\n")
		b.WriteString(t.MapKey.Tree(indent + "│  "))
		b.WriteRune('\n')

		b.WriteString(indent)
		b.WriteString("└─ MapValue\n")
		b.WriteString(t.MapValue.Tree(indent + "│  "))
		b.WriteRune('\n')
	} else {
		b.WriteString("false\n")
	}

	return b.String()
}

type Param struct {
	Name     string
	Type     TypeRef
	Variadic bool
}

func (p *Param) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(p.Type.Tree(indent))
	b.WriteString(p.Name)

	return indent + b.String()
}
