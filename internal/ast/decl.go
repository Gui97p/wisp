package ast

import (
	"fmt"
	"strings"
)

type Param struct {
	Name      string
	Type      string
	IsPointer bool
}

func (p *Param) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(p.Type)
	b.WriteString(" ")
	if p.IsPointer {
		b.WriteRune('*')
	}
	b.WriteString(p.Name)

	return indent + b.String()
}

type ReturnType struct {
	Type      string
	IsPointer bool
}

func (r *ReturnType) Tree(indent string) string {
	var b strings.Builder

	if r.IsPointer {
		b.WriteRune('*')
	}
	b.WriteString(r.Type)

	return indent + b.String()
}

type FuncDecl struct {
	Name        string
	Params      []Param
	Body        *BlockStmt
	ReturnTypes []ReturnType
}

func (*FuncDecl) decl() {}
func (f *FuncDecl) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sFuncDecl(%s)\n", indent, f.Name)

	b.WriteString(indent)
	b.WriteString("├─ Params\n")

	for _, p := range f.Params {
		b.WriteString(p.Tree(indent + "│  "))
		b.WriteRune('\n')
	}

	b.WriteString(indent)
	b.WriteString("├─ ReturnTypes\n")

	for _, r := range f.ReturnTypes {
		b.WriteString(r.Tree(indent + "│  "))
		b.WriteRune('\n')
	}

	if f.Body != nil {
		b.WriteString(indent)
		b.WriteString("└─ Body\n")
		b.WriteString(f.Body.Tree(indent + "   "))
	}

	return b.String()
}

type StructDecl struct {
	Name    string
	Members []Param
}

func (*StructDecl) decl() {}
func (s *StructDecl) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sStructDecl(%s)\n", indent, s.Name)

	b.WriteString(indent)
	b.WriteString("└─ Members\n")

	for _, m := range s.Members {
		b.WriteString(m.Tree(indent + "   "))
		b.WriteRune('\n')
	}

	return b.String()
}
