package ast

import (
	"fmt"
	"strings"
)

type IdentLiteral struct {
	Value string
}

func (*IdentLiteral) expr() {}
func (i *IdentLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("IdentLiteral(%s)\n", i.Value)
}

type ArrayLiteral struct {
	Elements []Expression
}

func (*ArrayLiteral) expr() {}
func (a *ArrayLiteral) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("ArrayLiteral\n")

	b.WriteString(indent)
	b.WriteString("├─ Args\n")

	for _, e := range a.Elements {
		b.WriteString(e.Tree(indent + "│  "))
	}

	return b.String()
}

type IntLiteral struct {
	Value int64
}

func (*IntLiteral) expr() {}
func (i *IntLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("IntLiteral(%d)\n", i.Value)
}

type FloatLiteral struct {
	Value float64
}

func (*FloatLiteral) expr() {}
func (f *FloatLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("FloatLiteral(%f)\n", f.Value)
}

type StringLiteral struct {
	Value string
}

func (*StringLiteral) expr() {}
func (s *StringLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("StringLiteral(%s)\n", s.Value)
}

type CharLiteral struct {
	Value byte
}

func (*CharLiteral) expr() {}
func (c *CharLiteral) Tree(indent string) string {
	return indent + fmt.Sprintf("CharLiteral(%c)\n", c.Value)
}

type BoolLiteral struct {
	Value bool
}

func (*BoolLiteral) expr() {}
func (b *BoolLiteral) Tree(indent string) string {
	if b.Value {
		return indent + "BoolLiteral(true)"
	}

	return indent + "BoolLiteral(false)"
}
