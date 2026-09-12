package ast

import (
	"fmt"
	"strings"
)

type Node interface {
	Tree(indent string) string
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

type TypeRef struct {
	Name         string
	PointerDepth int
	IsArray      bool
	ArraySize    int64
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

	b.WriteString(indent)
	b.WriteString("└─ IsArray\n")
	b.WriteString(indent)
	b.WriteString("│  ")
	if t.IsArray {
		b.WriteString("true\n")

		b.WriteString(indent)
		b.WriteString("└─ ArraySize\n")
		b.WriteString(indent)
		b.WriteString("│  ")
		fmt.Fprint(&b, t.ArraySize)
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
