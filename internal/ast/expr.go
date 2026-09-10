package ast

import (
	"fmt"
	"strings"
)

type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (*BinaryExpr) expr() {}
func (b *BinaryExpr) Tree(indent string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%sBinaryExpr(%s)\n", indent, b.Operator)

	if b.Left != nil {
		sb.WriteString(indent)
		sb.WriteString("├─ Left\n")
		sb.WriteString(b.Left.Tree(indent + "│  "))
	}

	if b.Right != nil {
		sb.WriteString(indent)
		sb.WriteString("└─ Right\n")
		sb.WriteString(b.Right.Tree(indent + "   "))
	}

	return sb.String()
}

type UnaryExpr struct {
	Value    Expression
	Operator string
}

func (*UnaryExpr) expr() {}
func (u *UnaryExpr) Tree(indent string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%sUnaryExpr(%s)\n", indent, u.Operator)

	if u.Value != nil {
		sb.WriteString(u.Value.Tree(indent + "└─ "))
	}

	return sb.String()
}

type CallExpr struct {
	Name Expression
	Args []Expression
}

func (*CallExpr) expr() {}
func (c *CallExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sCallExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Name\n")
	b.WriteString(c.Name.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("├─ Args\n")

	for _, a := range c.Args {
		b.WriteString(a.Tree(indent + "│  "))
	}

	return b.String()
}

type MemberExpr struct {
	Object Expression
	Field  string
}

type IndexExpr struct {
	Array Expression
	Index Expression
}

func (*IndexExpr) expr() {}
func (i *IndexExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sIndexExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Array\n")
	b.WriteString(i.Array.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("├─ Index\n")
	b.WriteString(i.Index.Tree(indent + "│  "))

	return b.String()
}

func (*MemberExpr) expr() {}
func (m *MemberExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sMemberExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Object\n")
	b.WriteString(m.Object.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("└─ Field\n")
	b.WriteString(indent)
	b.WriteString("│  ")
	b.WriteString(m.Field)

	return b.String()
}
