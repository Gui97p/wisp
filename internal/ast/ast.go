package ast

import (
	"fmt"
	"strings"
)

type Node interface {
	Tree(indent string) string
	Position() (int, int)
	EndPosition() (int, int)
	SetPos(line, col int)
	SetEndPos(line, col int)
}

type Declaration interface {
	Node
	decl()
}

type Statement interface {
	Node
	stmt()
}

type Expression interface {
	Node
	expr()
}

type NodePos struct {
	Line, Col       int
	EndLine, EndCol int
}

func (p NodePos) Position() (int, int) {
	return p.Line, p.Col
}

func (p NodePos) EndPosition() (int, int) {
	return p.EndLine, p.EndCol
}

func (p *NodePos) SetPos(line, col int) {
	p.Line, p.Col = line, col
}

func (p *NodePos) SetEndPos(line, col int) {
	p.EndLine, p.EndCol = line, col
}

type Program struct {
	Declarations []Declaration
	Target       string
	TargetAt     Position
	Standalone   bool
	Allow        []string
	Allowed      map[string]map[string]bool
}

func (p *Program) Tree(indent string) string {
	var b strings.Builder

	b.WriteString("Program\n")

	for i, decl := range p.Declarations {
		if i == len(p.Declarations)-1 {
			b.WriteString(decl.Tree("└─ "))
		} else {
			b.WriteString(decl.Tree("├─ "))
		}
	}

	return b.String()
}

type Position struct {
	Line, Col, EndLine, EndCol int
}

type TypeLayer struct {
	Pointer  bool
	Optional bool
	IsSpan   bool
	Size     int64
	Expr     Expression
	Text     string
}

type TypeParam struct {
	Name       string
	Const      bool
	Type       TypeRef
	Constraint *Constraint
	At         Position
}

type ConstraintKind int

const (
	ConstraintType ConstraintKind = iota
	ConstraintOr
	ConstraintAnd
	ConstraintNot
)

type Constraint struct {
	Kind  ConstraintKind
	Type  *TypeRef
	Left  *Constraint
	Right *Constraint
	At    Position
}

func (c *Constraint) String() string {
	switch c.Kind {
	case ConstraintOr:
		return c.Left.operand(ConstraintOr) + " | " + c.Right.operand(ConstraintOr)
	case ConstraintAnd:
		return c.Left.operand(ConstraintAnd) + " & " + c.Right.operand(ConstraintAnd)
	case ConstraintNot:
		return "!" + c.Left.operand(ConstraintNot)
	}
	return c.Type.String()
}

func (c *Constraint) operand(parent ConstraintKind) string {
	text := c.String()
	switch {
	case c.Kind == ConstraintType:
		return text
	case parent == ConstraintNot, c.Kind == ConstraintOr && parent == ConstraintAnd:
		return "(" + text + ")"
	}
	return text
}

type TypeArg struct {
	Type *TypeRef
	Expr Expression
}

func (a TypeArg) String() string {
	if a.Type != nil {
		return a.Type.String()
	}
	return ExprText(a.Expr)
}

func TypeArgsText(args []TypeArg) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.String()
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

type TypeRef struct {
	Module string
	Name   string
	Args   []TypeArg

	Line, Col, EndLine, EndCol int

	Layers []TypeLayer

	IsFunc       bool
	FuncParams   []TypeRef
	FuncReturns  []TypeRef
	FuncFallible bool
}

func (t *TypeRef) String() string {
	var b strings.Builder
	for _, layer := range t.Layers {
		switch {
		case layer.Pointer:
			b.WriteString("*")
		case layer.Optional:
			b.WriteString("?")
		case layer.IsSpan:
			b.WriteString("[]")
		case layer.Expr != nil:
			fmt.Fprintf(&b, "[%s]", layer.Text)
		default:
			fmt.Fprintf(&b, "[%d]", layer.Size)
		}
	}
	if t.Module != "" {
		b.WriteString(t.Module)
		b.WriteString(".")
	}
	b.WriteString(t.Name)
	if len(t.Args) > 0 {
		b.WriteString(TypeArgsText(t.Args))
	}
	return b.String()
}

func (t *TypeRef) Tree(indent string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sTypeRef\n", indent)

	if t.Module != "" {
		fmt.Fprintf(&b, "%s├─ Module\n%s│  %s\n", indent, indent, t.Module)
	}
	fmt.Fprintf(&b, "%s├─ Name\n%s│  %s\n", indent, indent, t.Name)

	for k, layer := range t.Layers {
		fmt.Fprintf(&b, "%s├─ Layer(%d)\n", indent, k)
		switch {
		case layer.Pointer:
			fmt.Fprintf(&b, "%s│  └─ Pointer\n", indent)
		case layer.Optional:
			fmt.Fprintf(&b, "%s│  └─ Optional\n", indent)
		case layer.IsSpan:
			fmt.Fprintf(&b, "%s│  └─ Span\n", indent)
		case layer.Expr != nil:
			fmt.Fprintf(&b, "%s│  └─ ArraySize\n%s│     %s\n", indent, indent, layer.Text)
		default:
			fmt.Fprintf(&b, "%s│  └─ ArraySize\n%s│     %d\n", indent, indent, layer.Size)
		}
	}

	return b.String()
}

type Param struct {
	Name     string
	Type     TypeRef
	Variadic bool
	Any      bool
	Const    bool
	Doc      string
	Reach    int

	Line, Col       int
	EndLine, EndCol int
}

func (p *Param) Tree(indent string) string {
	var b strings.Builder

	if p.Type.Name != "" {
		b.WriteString(p.Type.Tree(indent))
	}
	if p.Name != "" {
		fmt.Fprintf(&b, "%sName: %s\n", indent, p.Name)
	}

	return b.String()
}

type Tag struct {
	Name string
	Args []Expression
	At   Position
}

type Tagged struct {
	Tags []Tag
}

func (t *Tagged) SetTags(tags []Tag) {
	t.Tags = tags
}

func (t *Tagged) TagNamed(name string) *Tag {
	for i := range t.Tags {
		if t.Tags[i].Name == name {
			return &t.Tags[i]
		}
	}
	return nil
}

type Tags interface {
	SetTags([]Tag)
	TagNamed(name string) *Tag
}
