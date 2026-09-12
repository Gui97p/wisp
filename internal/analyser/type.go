package analyser

import (
	"fmt"
	"strings"
)

type Type interface {
	String() string
	Equals(Type) bool
}

type InvalidType struct{}

func (InvalidType) String() string {
	return "<invalid>"
}

func (InvalidType) Equals(other Type) bool {
	return true
}

type PrimitiveType struct {
	Name string
}

func (p PrimitiveType) String() string {
	return p.Name
}

func (p PrimitiveType) Equals(other Type) bool {
	o, ok := other.(PrimitiveType)
	if !ok {
		return false
	}
	return p.Name == o.Name
}

type PointerType struct {
	Element Type
}

func (p PointerType) String() string {
	return "*" + p.Element.String()
}

func (p PointerType) Equals(other Type) bool {
	o, ok := other.(PointerType)
	if !ok {
		return false
	}
	return p.Element.Equals(o.Element)
}

type ArrayType struct {
	Element Type
	Size    int64
}

func (a ArrayType) String() string {
	return fmt.Sprintf("%s[%d]", a.Element.String(), a.Size)
}

func (a ArrayType) Equals(other Type) bool {
	o, ok := other.(ArrayType)
	if !ok {
		return false
	}
	return a.Size == o.Size && a.Element.Equals(o.Element)
}

type StructType struct {
	Name   string
	Fields map[string]Type
	Order  []string
}

func (s *StructType) String() string {
	var b strings.Builder
	b.WriteString("struct ")
	b.WriteString(s.Name)
	b.WriteString(" { ")
	for _, field := range s.Order {
		tp := s.Fields[field]
		b.WriteString(tp.String())
		b.WriteRune(' ')
		b.WriteString(field)
		b.WriteRune(' ')
	}
	b.WriteRune('}')

	return b.String()
}

func (s *StructType) Equals(other Type) bool {
	o, ok := other.(*StructType)
	if !ok {
		return false
	}
	return s.Name == o.Name
}

type FuncType struct {
	Name    string
	Params  []Type
	Returns []Type
}

func (f *FuncType) String() string {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(f.Name)
	b.WriteRune('(')
	for i, tp := range f.Params {
		b.WriteString(tp.String())
		if i != len(f.Params)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteRune(')')

	b.WriteString(" (")
	for i, tp := range f.Returns {
		b.WriteString(tp.String())
		if i != len(f.Returns)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteRune(')')

	return b.String()
}

func (f *FuncType) Equals(other Type) bool {
	o, ok := other.(*FuncType)
	if !ok {
		return false
	}
	if len(f.Params) != len(o.Params) || len(f.Returns) != len(o.Returns) {
		return false
	}
	for i := range f.Params {
		if !f.Params[i].Equals(o.Params[i]) {
			return false
		}
	}
	for i := range f.Returns {
		if !f.Returns[i].Equals(o.Returns[i]) {
			return false
		}
	}
	return true
}

var primitives = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"char": true, "string": true,
	"bool": true,
}

func isNumeric(t Type) bool {
	p, ok := t.(PrimitiveType)
	if !ok {
		return false
	}
	switch p.Name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}
