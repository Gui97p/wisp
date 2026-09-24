package ast

import (
	"fmt"
	"strings"
)

type ExpressionStmt struct {
	Expr Expression

	NodePos
}

func (*ExpressionStmt) stmt() {}
func (e *ExpressionStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("ExpressionStmt\n")

	if e.Expr != nil {
		b.WriteString(e.Expr.Tree(indent + "└─ "))
	}

	return b.String()
}

type BlockStmt struct {
	Statements []Statement

	NodePos
}

func (*BlockStmt) stmt() {}
func (b *BlockStmt) Tree(indent string) string {
	var sb strings.Builder

	sb.WriteString(indent)
	sb.WriteString("BlockStmt\n")

	for _, stmt := range b.Statements {
		sb.WriteString(stmt.Tree(indent + "│  "))
	}

	return sb.String()
}

type GroupStmt struct {
	Statements []Statement

	NodePos
}

func (*GroupStmt) stmt() {}
func (g *GroupStmt) Tree(indent string) string {
	var sb strings.Builder

	sb.WriteString(indent)
	sb.WriteString("GroupStmt\n")

	for _, stmt := range g.Statements {
		sb.WriteString(stmt.Tree(indent + "│  "))
	}

	return sb.String()
}

type VarStmt struct {
	Vars   []Param
	Values []Expression

	NodePos
}

func (*VarStmt) stmt() {}
func (v *VarStmt) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sVarStmt\n", indent)

	total := len(v.Vars) + len(v.Values)
	n := 0
	branch := func() string {
		n++
		if n == total {
			return "└─ "
		}
		return "├─ "
	}

	for _, value := range v.Vars {
		b.WriteString(value.Tree(indent + branch()))
	}

	for _, value := range v.Values {
		b.WriteString(value.Tree(indent + branch()))
	}

	return b.String()
}

type ConstStmt struct {
	Vars   []Param
	Values []Expression

	NodePos
}

func (*ConstStmt) stmt() {}
func (c *ConstStmt) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sConstStmt\n", indent)

	total := len(c.Vars) + len(c.Values)
	n := 0
	branch := func() string {
		n++
		if n == total {
			return "└─ "
		}
		return "├─ "
	}

	for _, value := range c.Vars {
		b.WriteString(value.Tree(indent + branch()))
	}

	for _, value := range c.Values {
		b.WriteString(value.Tree(indent + branch()))
	}

	return b.String()
}

type ReturnStmt struct {
	Values []Expression

	NodePos
}

func (*ReturnStmt) stmt() {}
func (r *ReturnStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("ReturnStmt\n")

	for i, value := range r.Values {
		if i == len(r.Values)-1 {
			b.WriteString(value.Tree(indent + "└─ "))
		} else {
			b.WriteString(value.Tree(indent + "├─ "))
		}
	}

	return b.String()
}

type IfStmt struct {
	Condition Expression
	Then      *BlockStmt
	Else      Statement

	NodePos
}

func (*IfStmt) stmt() {}
func (i *IfStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("IfStmt\n")

	if i.Condition != nil {
		b.WriteString(indent)
		b.WriteString("├─ Condition\n")
		b.WriteString(i.Condition.Tree(indent + "│  "))
	}

	if i.Then != nil {
		b.WriteString(indent)
		if i.Else != nil {
			b.WriteString("├─ Then\n")
			b.WriteString(i.Then.Tree(indent + "│  "))
		} else {
			b.WriteString("└─ Then\n")
			b.WriteString(i.Then.Tree(indent + "   "))
		}
	}

	if i.Else != nil {
		b.WriteString(indent)
		b.WriteString("└─ Else\n")
		b.WriteString(i.Else.Tree(indent + "   "))
	}

	return b.String()
}

type ForStmt struct {
	Label string
	Var   string
	Var2  string
	Start Expression
	End   Expression
	Step  Expression
	Range Expression
	Body  *BlockStmt

	NodePos
}

func (*ForStmt) stmt() {}
func (f *ForStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("ForStmt\n")

	if f.Label != "" {
		fmt.Fprintf(&b, "%s├─ Label(%s)\n", indent, f.Label)
	}

	if f.Var != "" {
		fmt.Fprintf(&b, "%s├─ Var(%s)\n", indent, f.Var)
	}
	if f.Var2 != "" {
		fmt.Fprintf(&b, "%s├─ Var2(%s)\n", indent, f.Var)
	}

	if f.Start != nil {
		b.WriteString(indent)
		b.WriteString("├─ Start\n")
		b.WriteString(f.Start.Tree(indent + "│  "))
	}

	if f.End != nil {
		b.WriteString(indent)
		b.WriteString("├─ End\n")
		b.WriteString(f.End.Tree(indent + "│  "))
	}

	if f.Step != nil {
		b.WriteString(indent)
		b.WriteString("├─ Step\n")
		b.WriteString(f.Step.Tree(indent + "│  "))
	}

	if f.Range != nil {
		b.WriteString(indent)
		b.WriteString("├─ Range\n")
		b.WriteString(f.Range.Tree(indent + "│  "))
	}

	if f.Body != nil {
		b.WriteString(indent)
		b.WriteString("└─ Body\n")
		b.WriteString(f.Body.Tree(indent + "   "))
	}

	return b.String()
}

type LoopStmt struct {
	Label          string
	Condition      Expression
	Body           *BlockStmt
	UntilCondition Expression

	NodePos
}

func (*LoopStmt) stmt() {}
func (l *LoopStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("LoopStmt\n")

	if l.Label != "" {
		fmt.Fprintf(&b, "%s├─ Label(%s)\n", indent, l.Label)
	}

	if l.Condition != nil {
		b.WriteString(indent)
		b.WriteString("├─ Condition\n")
		b.WriteString(l.Condition.Tree(indent + "│  "))
	}

	if l.UntilCondition != nil {
		b.WriteString(indent)
		b.WriteString("├─ Until\n")
		b.WriteString(l.UntilCondition.Tree(indent + "│  "))
	}

	if l.Body != nil {
		b.WriteString(indent)
		b.WriteString("└─ Body\n")
		b.WriteString(l.Body.Tree(indent + "   "))
	}

	return b.String()
}

type BreakStmt struct {
	Label string

	NodePos
}

func (*BreakStmt) stmt() {}
func (b *BreakStmt) Tree(indent string) string {
	if b.Label != "" {
		return fmt.Sprintf("%sBreakStmt(%s)\n", indent, b.Label)
	}

	return indent + "BreakStmt\n"
}

type ContinueStmt struct {
	Label string

	NodePos
}

func (*ContinueStmt) stmt() {}
func (c *ContinueStmt) Tree(indent string) string {
	if c.Label != "" {
		return fmt.Sprintf("%sContinueStmt(%s)\n", indent, c.Label)
	}

	return indent + "ContinueStmt\n"
}

type NativeStmt struct {
	Code string

	NodePos
}

func (*NativeStmt) stmt() {}
func (n *NativeStmt) Tree(indent string) string {
	return fmt.Sprintf("%sNativeStmt(%s)\n", indent, n.Code)
}

type AssignStmt struct {
	Target Expression
	Op     string
	Value  Expression

	NodePos
}

func (*AssignStmt) stmt() {}
func (a *AssignStmt) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sAssignStmt(%s)\n", indent, a.Op)

	if a.Target != nil {
		b.WriteString(a.Target.Tree(indent + "├─ "))
	}

	if a.Value != nil {
		b.WriteString(a.Value.Tree(indent + "└─ "))
	}

	return b.String()
}

type IncDecStmt struct {
	Target Expression
	Op     string

	NodePos
}

func (*IncDecStmt) stmt() {}
func (i *IncDecStmt) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sIncDecStmt(%s)\n", indent, i.Op)

	if i.Target != nil {
		b.WriteString(i.Target.Tree(indent + "└─ "))
	}

	return b.String()
}
