package ast

import "strings"

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
