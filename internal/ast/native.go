package ast

import (
	"strings"
)

type NativeOperandKind int

const (
	NativeWord NativeOperandKind = iota
	NativeNumber
	NativeAddr
	NativeMem
)

type NativeTerm struct {
	Negative bool
	Scale    string
	Number   string
	Path     []string
	At       Position
}

func (t NativeTerm) String() string {
	var b strings.Builder
	if t.Scale != "" {
		b.WriteString(t.Scale)
		b.WriteString("*")
	}
	if t.Number != "" {
		b.WriteString(t.Number)
	} else {
		b.WriteString(strings.Join(t.Path, "."))
	}
	return b.String()
}

type NativeOperand struct {
	Kind   NativeOperandKind
	Path   []string
	Number string
	Size   string
	Rel    bool
	Terms  []NativeTerm
	At     Position
}

func (o NativeOperand) String() string {
	switch o.Kind {
	case NativeNumber:
		return o.Number
	case NativeAddr:
		return "&" + strings.Join(o.Path, ".")
	case NativeMem:
		var b strings.Builder
		if o.Size != "" {
			b.WriteString(o.Size)
			b.WriteString(" ")
		}
		b.WriteString("[")
		if o.Rel {
			b.WriteString("rel ")
		}
		for i, t := range o.Terms {
			switch {
			case t.Negative:
				b.WriteString(" - ")
			case i > 0:
				b.WriteString(" + ")
			}
			b.WriteString(t.String())
		}
		b.WriteString("]")
		return b.String()
	}
	return strings.Join(o.Path, ".")
}

type NativeInstr struct {
	Label    string
	Prefixes []string
	Mnemonic string
	Operands []NativeOperand
	At       Position
}

func (i NativeInstr) String() string {
	var b strings.Builder
	if i.Label != "" {
		b.WriteString(i.Label)
		b.WriteString(":")
		if i.Mnemonic != "" {
			b.WriteString(" ")
		}
	}
	for _, p := range i.Prefixes {
		b.WriteString(p)
		b.WriteString(" ")
	}
	b.WriteString(i.Mnemonic)
	for n, o := range i.Operands {
		if n == 0 {
			b.WriteString(" ")
		} else {
			b.WriteString(", ")
		}
		b.WriteString(o.String())
	}
	return b.String()
}
