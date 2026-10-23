package ast

import (
	"fmt"
	"strings"
)

type ConstDecl struct {
	Doc      string
	Vars     []Param
	Values   []Expression
	Exported bool
	Reach    int

	NodePos
}

func (*ConstDecl) decl() {}
func (c *ConstDecl) Tree(indent string) string {
	var b strings.Builder
	b.WriteString(indent)
	b.WriteString("ConstDecl\n")

	total := len(c.Vars) + len(c.Values)
	n := 0
	branch := func() string {
		n++
		if n == total {
			return "└─ "
		}
		return "├─ "
	}

	for _, v := range c.Vars {
		b.WriteString(v.Tree(indent + branch()))
	}
	for _, value := range c.Values {
		b.WriteString(value.Tree(indent + branch()))
	}

	return b.String()
}

type VarDecl struct {
	Doc      string
	Vars     []Param
	Values   []Expression
	Exported bool
	Reach    int

	NodePos
}

func (*VarDecl) decl() {}
func (c *VarDecl) Tree(indent string) string {
	var b strings.Builder
	b.WriteString(indent)
	b.WriteString("VarDecl\n")

	total := len(c.Vars) + len(c.Values)
	n := 0
	branch := func() string {
		n++
		if n == total {
			return "└─ "
		}
		return "├─ "
	}

	for _, v := range c.Vars {
		b.WriteString(v.Tree(indent + branch()))
	}
	for _, value := range c.Values {
		b.WriteString(value.Tree(indent + branch()))
	}

	return b.String()
}

type FuncDecl struct {
	Doc         string
	Name        string
	Receiver    *Param
	Params      []Param
	Body        *BlockStmt
	ReturnTypes []TypeRef
	Fallible    bool
	Exported    bool
	Reach       int
	Extern      bool
	ExternName  string
	CExport     string
	CVariadic   bool
	Generated   bool

	NodePos
}

func (f *FuncDecl) SignatureEnd() (int, int) {
	line, col := f.Position()
	if f.Body == nil {
		return f.EndPosition()
	}
	bodyLine, bodyCol := f.Body.Position()
	if bodyLine == line && bodyCol-2 >= col {
		return bodyLine, bodyCol - 2
	}
	return line, col + 3
}

func (*FuncDecl) decl() {}
func (f *FuncDecl) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sFuncDecl(%s)\n", indent, f.Name)

	if f.Receiver != nil {
		b.WriteString(indent)
		b.WriteString("├─ Receiver\n")
		b.WriteString(f.Receiver.Tree(indent + "│  "))
	}

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
	Doc      string
	Name     string
	Members  []Param
	Exported bool
	Reach    int

	NodePos
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

type EnumMember struct {
	Name    string
	Value   int64
	Doc     string
	Line    int
	Col     int
	EndLine int
	EndCol  int
}

type TypeDecl struct {
	Doc        string
	Name       string
	Underlying TypeRef
	Exported   bool
	Reach      int
	IsEnum     bool
	Members    []EnumMember

	NodePos
}

func (*TypeDecl) decl() {}
func (t *TypeDecl) Tree(indent string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%sTypeDecl(%s)\n", indent, t.Name)
	b.WriteString(indent)
	b.WriteString("└─ Underlying\n")
	b.WriteString(t.Underlying.Tree(indent + "   "))
	return b.String()
}

type ImportDecl struct {
	Path  string
	Alias string

	NodePos
}

func (*ImportDecl) decl() {}
func (i *ImportDecl) Tree(indent string) string {
	return fmt.Sprintf("%sImportDecl(%q as %s)\n", indent, i.Path, i.Alias)
}

const (
	ReachNone = iota
	ReachInternal
	ReachShared
)

type ReexportDecl struct {
	Name  string
	Alias string
	Reach int

	NodePos
}

func (*ReexportDecl) decl() {}
func (r *ReexportDecl) Tree(indent string) string {
	return fmt.Sprintf("%sReexportDecl(%s)\n", indent, r.Name)
}
