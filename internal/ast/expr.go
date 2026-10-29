package ast

import (
	"fmt"
	"strings"
)

type BinaryExpr struct {
	Left      Expression
	Operator  string
	Right     Expression
	Synthetic bool

	NodePos
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
	Cast     *CastExpr

	NodePos
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

type DerefExpr struct {
	Value Expression

	NodePos
}

func (*DerefExpr) expr() {}
func (d *DerefExpr) Tree(indent string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%sDerefExpr\n", indent)
	if d.Value != nil {
		sb.WriteString(d.Value.Tree(indent + "└─ "))
	}

	return sb.String()
}

type TypeLabel struct {
	Type TypeRef

	NodePos
}

func (*TypeLabel) expr() {}
func (t *TypeLabel) Tree(indent string) string {
	return fmt.Sprintf("%sTypeLabel(%s%s)\n", indent, strings.Repeat("*", t.Type.PointerDepth), t.Type.Name)
}

type CallExpr struct {
	Name   Expression
	Args   []Expression
	Spread bool
	Cast   *CastExpr

	NodePos
}

func (*CallExpr) expr() {}
func (c *CallExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sCallExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Name\n")
	b.WriteString(c.Name.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("└─ Args\n")

	for _, a := range c.Args {
		b.WriteString(a.Tree(indent + "   "))
	}

	return b.String()
}

type MemberExpr struct {
	Object  Expression
	Field   string
	FieldAt Position

	NodePos
}

type IndexExpr struct {
	Array Expression
	Index Expression

	NodePos
}

func (*IndexExpr) expr() {}
func (i *IndexExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sIndexExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Array\n")
	b.WriteString(i.Array.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("└─ Index\n")
	b.WriteString(i.Index.Tree(indent + "   "))

	return b.String()
}

type SliceExpr struct {
	Array Expression
	Start Expression
	End   Expression

	NodePos
}

func (*SliceExpr) expr() {}
func (s *SliceExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sSliceExpr\n", indent)

	if s.Start == nil && s.End == nil {
		b.WriteString(indent)
		b.WriteString("└─ Array\n")
		b.WriteString(s.Array.Tree(indent + "   "))
	} else {
		b.WriteString(indent)
		b.WriteString("├─ Array\n")
		b.WriteString(s.Array.Tree(indent + "│  "))
	}

	if s.Start != nil {
		if s.End == nil {
			b.WriteString(indent)
			b.WriteString("└─ Start\n")
			b.WriteString(s.Start.Tree(indent + "   "))
		} else {
			b.WriteString(indent)
			b.WriteString("├─ Start\n")
			b.WriteString(s.Start.Tree(indent + "│  "))
		}
	}

	if s.End != nil {
		b.WriteString(indent)
		b.WriteString("└─ End\n")
		b.WriteString(s.End.Tree(indent + "   "))
	}

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
	b.WriteString("   ")
	b.WriteString(m.Field)
	b.WriteRune('\n')

	return b.String()
}

type TernaryExpr struct {
	Condition Expression
	Then      Expression
	Else      Expression
	Synthetic bool

	NodePos
}

func (*TernaryExpr) expr() {}
func (i *TernaryExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sTernaryExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Condition\n")
	b.WriteString(i.Condition.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("├─ Then\n")
	b.WriteString(i.Then.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("└─ Else\n")
	b.WriteString(i.Else.Tree(indent + "   "))

	return b.String()
}

type CoalesceExpr struct {
	Left  Expression
	Block Statement

	ErrorBind   string
	ErrorBindAt Position

	NodePos
}

func (*CoalesceExpr) expr() {}
func (c *CoalesceExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sCoalesceExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Left\n")
	b.WriteString(c.Left.Tree(indent + "│  "))

	if c.Block != nil {
		b.WriteString(indent)
		fmt.Fprintf(&b, "└─ Block(%s)\n", c.ErrorBind)
		b.WriteString(c.Block.Tree(indent + "   "))
	}

	return b.String()
}

type TupleExpr struct {
	Elements []Expression

	NodePos
}

func (*TupleExpr) expr() {}
func (t *TupleExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sTupleExpr\n", indent)
	for i, e := range t.Elements {
		branch, pad := "├─ ", "│  "
		if i == len(t.Elements)-1 {
			branch, pad = "└─ ", "   "
		}
		b.WriteString(indent)
		b.WriteString(branch)
		b.WriteString("\n")
		b.WriteString(e.Tree(indent + pad))
	}

	return b.String()
}

type ConvExpr struct {
	Type TypeRef
	Args []Expression
	Cast *CastExpr

	NodePos
}

func (*ConvExpr) expr() {}
func (c *ConvExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sConvExpr(%s%s)\n", indent, strings.Repeat("*", c.Type.PointerDepth), c.Type.Name)
	for _, a := range c.Args {
		b.WriteString(a.Tree(indent + "└─ "))
	}

	return b.String()
}

type CastExpr struct {
	Value    Expression
	Type     TypeRef
	Implicit bool

	NodePos
}

func (*CastExpr) expr() {}
func (c *CastExpr) Tree(indent string) string {
	var b strings.Builder

	if c.Implicit {
		fmt.Fprintf(&b, "%sCastExpr(implicit)\n", indent)
	} else {
		fmt.Fprintf(&b, "%sCastExpr(%s)\n", indent, c.Type.Name)
	}
	b.WriteString(indent)
	b.WriteString("└─ Value\n")
	b.WriteString(c.Value.Tree(indent + "   "))

	return b.String()
}

type InExpr struct {
	Left  Expression
	Right Expression

	NodePos
}

func (*InExpr) expr() {}
func (i *InExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sInExpr\n", indent)

	b.WriteString(indent)
	b.WriteString("├─ Left\n")
	b.WriteString(i.Left.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("└─ Right\n")
	b.WriteString(i.Right.Tree(indent + "|  "))

	return b.String()
}

type PropagateExpr struct {
	Value  Expression
	Assert bool

	NodePos
}

func (*PropagateExpr) expr() {}
func (p *PropagateExpr) Tree(indent string) string {
	var b strings.Builder

	name := "PropagateExpr"
	if p.Assert {
		name = "AssertExpr"
	}
	fmt.Fprintf(&b, "%s%s\n", indent, name)
	b.WriteString(indent)
	b.WriteString("└─ Value\n")
	b.WriteString(p.Value.Tree(indent + "   "))

	return b.String()
}

type SwitchArm struct {
	Condition Expression
	Values    []Expression
	Exit      Statement

	NodePos
}

type SwitchExpr struct {
	Prelude []Statement
	Arms    []SwitchArm
	Default *SwitchArm

	NodePos
}

func (*SwitchExpr) expr() {}
func (s *SwitchExpr) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sSwitchExpr\n", indent)

	for i := range s.Arms {
		b.WriteString(s.Arms[i].tree(indent+"├─ ", indent+"│  "))
	}
	if s.Default != nil {
		b.WriteString(s.Default.tree(indent+"└─ ", indent+"   "))
	}

	return b.String()
}

func (a *SwitchArm) Tree(indent string) string {
	return a.tree(indent, indent)
}

func (a *SwitchArm) tree(head, pad string) string {
	var b strings.Builder

	if a.Condition != nil {
		b.WriteString(head + "Case\n")
		b.WriteString(a.Condition.Tree(pad + "├─ "))
	} else {
		b.WriteString(head + "Default\n")
	}
	for _, v := range a.Values {
		b.WriteString(v.Tree(pad + "└─ "))
	}
	if a.Exit != nil {
		b.WriteString(a.Exit.Tree(pad + "└─ "))
	}

	return b.String()
}
