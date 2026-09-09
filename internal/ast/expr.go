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
