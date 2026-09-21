package ast

import (
	"fmt"
	"strings"
)

type IdentLiteral struct {
	Value string

	NodePos
}

func (*IdentLiteral) expr() {}
func (i *IdentLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("IdentLiteral(%s)\n", i.Value)
}

type ArrayLiteral struct {
	Elements []Expression

	NodePos
}

func (*ArrayLiteral) expr() {}
func (a *ArrayLiteral) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("ArrayLiteral\n")

	b.WriteString(indent)
	b.WriteString("└─ Args\n")

	for _, e := range a.Elements {
		b.WriteString(e.Tree(indent + "   "))
	}

	return b.String()
}

type MapLiteral struct {
	Keys   []Expression
	Values []Expression

	NodePos
}

func (*MapLiteral) expr() {}
func (m *MapLiteral) Tree(indent string) string {
	var b strings.Builder
	b.WriteString(indent)
	b.WriteString("MapLiteral\n")

	for i := range m.Keys {
		branch, childIndent := "├─ ", "│  "
		if i == len(m.Keys)-1 {
			branch, childIndent = "└─ ", "   "
		}

		fmt.Fprintf(&b, "%s%sEntry(%d)\n", indent, branch, i)
		b.WriteString(m.Keys[i].Tree(indent + childIndent + "├─ "))
		b.WriteString(m.Values[i].Tree(indent + childIndent + "└─ "))
	}

	return b.String()
}

type IntLiteral struct {
	Value int64

	NodePos
}

func (*IntLiteral) expr() {}
func (i *IntLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("IntLiteral(%d)\n", i.Value)
}

type FloatLiteral struct {
	Value float64

	NodePos
}

func (*FloatLiteral) expr() {}
func (f *FloatLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("FloatLiteral(%f)\n", f.Value)
}

type StringLiteral struct {
	Value string

	NodePos
}

func (*StringLiteral) expr() {}
func (s *StringLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("StringLiteral(%s)\n", s.Value)
}

type CharLiteral struct {
	Value byte

	NodePos
}

func (*CharLiteral) expr() {}
func (c *CharLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("CharLiteral(%c)\n", c.Value)
}

type BoolLiteral struct {
	Value bool

	NodePos
}

func (*BoolLiteral) expr() {}
func (b *BoolLiteral) Tree(indent string) string {
	if b.Value {
		return indent + "BoolLiteral(true)\n"
	}

	return indent + "BoolLiteral(false)\n"
}

type NullLiteral struct {
	NodePos
}

func (*NullLiteral) expr() {}
func (*NullLiteral) Tree(indent string) string {
	return indent + "NullLiteral(null)\n"
}

type StructLiteral struct {
	Name   string
	Keys   []string
	Values []Expression

	NodePos
}

func (*StructLiteral) expr() {}
func (s *StructLiteral) Tree(indent string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%sStructLiteral(%s)\n", indent, s.Name)

	for i := range s.Keys {
		branch, childIndent := "├─ ", "│  "
		if i == len(s.Keys)-1 {
			branch, childIndent = "└─ ", "   "
		}
		fmt.Fprintf(&b, "%s%sField(%s)\n", indent, branch, s.Keys[i])
		b.WriteString(s.Values[i].Tree(indent + childIndent))
	}

	return b.String()
}

type FuncLiteral struct {
	Params      []Param
	ReturnTypes []TypeRef
	Block       *BlockStmt

	NodePos
}

func (*FuncLiteral) expr() {}
func (f *FuncLiteral) Tree(indent string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%sFuncLiteral\n", indent)

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

	if f.Block != nil {
		b.WriteString(indent)
		b.WriteString("└─ Block\n")
		b.WriteString(f.Block.Tree(indent + "   "))
	}

	return b.String()
}
