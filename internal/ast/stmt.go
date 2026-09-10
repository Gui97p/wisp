package ast

import (
	"fmt"
	"strings"
)

type ExpressionStmt struct {
	Expr Expression
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

type VarStmt struct {
	Name  string
	Type  TypeRef
	Value Expression
}

func (*VarStmt) stmt() {}
func (v *VarStmt) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sVarStmt(%s)\n", indent, v.Name)

	b.WriteString(v.Type.Tree(indent + "├─ "))

	if v.Value != nil {
		b.WriteString(v.Value.Tree(indent + "└─ "))
	}

	return b.String()
}

type ReturnStmt struct {
	Values []Expression
}

func (*ReturnStmt) stmt() {}
func (r *ReturnStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("ReturnStmt\n")

	for _, value := range r.Values {
		b.WriteString(value.Tree(indent + "├─ "))
	}

	return b.String()
}

type IfStmt struct {
	Condition Expression
	Then      *BlockStmt
	Else      Statement
}

func (*IfStmt) stmt() {}
func (i *IfStmt) Tree(indent string) string {
	var b strings.Builder

	b.WriteString(indent)
	b.WriteString("IfStmt\n")

	b.WriteString(indent)
	b.WriteString("├─ Condition\n")
	b.WriteString(i.Condition.Tree(indent + "│  "))

	b.WriteString(indent)
	b.WriteString("├─ Then\n")
	b.WriteString(i.Then.Tree(indent + "│  "))

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
	Start Expression
	End   Expression
	Step  Expression
	Body  *BlockStmt
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
}

func (*ContinueStmt) stmt() {}
func (c *ContinueStmt) Tree(indent string) string {
	if c.Label != "" {
		return fmt.Sprintf("%sContinueStmt(%s)\n", indent, c.Label)
	}

	return indent + "ContinueStmt\n"
}

type AssignStmt struct {
	Target Expression
	Op     string
	Value  Expression
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
}

func (*IncDecStmt) stmt() {}
func (i *IncDecStmt) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sIncDecStmt(%s)\n", indent, i.Op)

	if i.Target != nil {
		b.WriteString(i.Target.Tree(indent + "├─ "))
	}

	return b.String()
}
