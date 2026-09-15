package ast

import (
	"fmt"
	"strings"
)

type FuncDecl struct {
	Name        string
	Params      []Param
	Body        *BlockStmt
	ReturnTypes []TypeRef
	Exported    bool
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
	Name     string
	Members  []Param
	Exported bool
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

type ImportDecl struct {
	Path  string
	Alias string
}

func (*ImportDecl) decl() {}
func (i *ImportDecl) Tree(indent string) string {
	return fmt.Sprintf("%sImportDecl(%q as %s)\n", indent, i.Path, i.Alias)
}
