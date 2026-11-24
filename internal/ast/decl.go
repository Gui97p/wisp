package ast

import (
	"fmt"
	"strings"
)

type ConstDecl struct {
	Tagged
	Doc        string
	Vars       []Param
	Values     []Expression
	Exported   bool
	Reach      int
	Selector   string
	SelectorAt Position

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
	Tagged
	Doc        string
	Vars       []Param
	Values     []Expression
	Exported   bool
	Reach      int
	Selector   string
	SelectorAt Position

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
	Tagged
	Doc         string
	Name        string
	Receiver    *Param
	TypeParams  []TypeParam
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
	Instance    string
	Inline      bool
	Native      bool
	Selector    string
	SelectorAt  Position
	NameAt      Position

	NodePos
}

func (f *FuncDecl) HasConstParams() bool {
	for _, p := range f.Params {
		if p.Const {
			return true
		}
	}
	return false
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
	Tagged
	Doc        string
	Name       string
	TypeParams []TypeParam
	Implements []TypeRef
	Members    []Param
	Exported   bool
	Reach      int
	Selector   string
	SelectorAt Position

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
	Name     string
	Value    int64
	Fields   []Param
	Payload  bool
	Numbered bool
	Doc      string
	Line     int
	Col      int
	EndLine  int
	EndCol   int
}

type TypeDecl struct {
	Tagged
	Doc        string
	Name       string
	TypeParams []TypeParam
	Implements []TypeRef
	Underlying TypeRef
	Exported   bool
	Reach      int
	IsEnum     bool
	Data       bool
	Members    []EnumMember
	Selector   string
	SelectorAt Position

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
	Path       string
	Alias      string
	Selector   string
	SelectorAt Position

	NodePos
}

func (*ImportDecl) decl() {}
func (i *ImportDecl) Tree(indent string) string {
	return fmt.Sprintf("%sImportDecl(%q as %s)\n", indent, i.Path, i.Alias)
}

const (
	ReachNone = iota
	ReachProject
	ReachFile
)

type StaticAssertDecl struct {
	Call *CallExpr

	NodePos
}

func (*StaticAssertDecl) decl() {}
func (d *StaticAssertDecl) Tree(indent string) string {
	return indent + "StaticAssertDecl\n" + d.Call.Tree(indent+"└─ ")
}

type InterfaceMethod struct {
	Name        string
	Doc         string
	Params      []Param
	ReturnTypes []TypeRef
	Fallible    bool
	At          Position
	NameAt      Position
}

type InterfaceDecl struct {
	Tagged
	Doc        string
	Name       string
	TypeParams []TypeParam
	Methods    []InterfaceMethod
	Constraint *Constraint
	Exported   bool
	Reach      int
	Selector   string
	SelectorAt Position

	NodePos
}

func (*InterfaceDecl) decl() {}
func (i *InterfaceDecl) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sInterfaceDecl(%s)\n", indent, i.Name)
	for _, m := range i.Methods {
		fmt.Fprintf(&b, "%s├─ Method(%s)\n", indent, m.Name)
	}
	if i.Constraint != nil {
		fmt.Fprintf(&b, "%s└─ Constraint(%s)\n", indent, i.Constraint.String())
	}

	return b.String()
}
