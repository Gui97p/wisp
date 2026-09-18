package analyser

import (
	"fmt"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
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
	if _, ok := other.(UntypedIntType); ok {
		return isNumeric(p)
	}
	if _, ok := other.(UntypedFloatType); ok {
		return p.Name == "float32" || p.Name == "float64"
	}
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
	if _, ok := other.(NullType); ok {
		return true
	}
	o, ok := other.(PointerType)
	if !ok {
		return false
	}
	if _, isVoid := p.Element.(VoidType); isVoid {
		return true
	}
	if _, isVoid := o.Element.(VoidType); isVoid {
		return true
	}
	return p.Element.Equals(o.Element)
}

type AnyType struct{}

func (AnyType) String() string {
	return "any"
}

func (AnyType) Equals(other Type) bool {
	return true
}

type UntypedIntType struct{}

func (UntypedIntType) String() string {
	return "untyped int"
}
func (UntypedIntType) Equals(other Type) bool {
	if nt, ok := other.(NamedType); ok {
		return isNumeric(nt.Underlying)
	}
	return isNumeric(other)
}

type UntypedFloatType struct{}

func (UntypedFloatType) String() string {
	return "untyped float"
}
func (UntypedFloatType) Equals(other Type) bool {
	if nt, ok := other.(NamedType); ok {
		return nt.Underlying.Equals(PrimitiveType{Name: "float32"}) || nt.Underlying.Equals(PrimitiveType{Name: "float64"})
	}
	p, ok := other.(PrimitiveType)
	if !ok {
		return false
	}
	return p.Name == "float32" || p.Name == "float64"
}

type NullType struct{}

func (NullType) String() string {
	return "null"
}

func (NullType) Equals(other Type) bool {
	_, ok := other.(PointerType)
	return ok
}

type VoidType struct{}

func (VoidType) String() string {
	return "void"
}
func (VoidType) Equals(other Type) bool {
	_, ok := other.(VoidType)
	return ok
}

type ArrayType struct {
	Element Type
	Size    int64
}

func (a ArrayType) String() string {
	return fmt.Sprintf("%s[%d]", a.Element, a.Size)
}

func (a ArrayType) Equals(other Type) bool {
	o, ok := other.(ArrayType)
	if !ok {
		return false
	}
	return a.Size == o.Size && a.Element.Equals(o.Element)
}

type SpanType struct {
	Element Type
}

func (s SpanType) String() string {
	return s.Element.String() + "[]"
}

func (s SpanType) Equals(other Type) bool {
	o, ok := other.(SpanType)
	if !ok {
		return false
	}
	return s.Element.Equals(o.Element)
}

type MapType struct {
	Key   Type
	Value Type
}

func (m *MapType) String() string {
	return fmt.Sprintf("map[%s]%s", m.Key, m.Value)
}

func (m *MapType) Equals(other Type) bool {
	o, ok := other.(*MapType)
	if !ok {
		return false
	}
	return m.Key.Equals(o.Key) && m.Value.Equals(o.Value)
}

type NamedType struct {
	Name       string
	Underlying Type
	Methods    map[string]*FuncType
}

func (n NamedType) String() string {
	return n.Name
}

func (n NamedType) Equals(other Type) bool {
	if _, ok := other.(UntypedIntType); ok {
		return isNumeric(n.Underlying)
	}
	if _, ok := other.(UntypedFloatType); ok {
		return n.Underlying.Equals(PrimitiveType{Name: "float32"}) || n.Underlying.Equals(PrimitiveType{Name: "float64"})
	}

	o, ok := other.(NamedType)
	return ok && n.Name == o.Name
}

type StructType struct {
	Name    string
	Fields  map[string]Type
	Order   []string
	Methods map[string]*FuncType
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
	Name     string
	Params   []Type
	Returns  []Type
	Variadic bool
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

func MethodsOf(t Type) map[string]*FuncType {
	if pt, ok := t.(PointerType); ok {
		t = pt.Element
	}
	switch tt := t.(type) {
	case *StructType:
		return tt.Methods
	case NamedType:
		return tt.Methods
	default:
		return nil
	}
}

func isNumeric(t Type) bool {
	switch t.(type) {
	case UntypedIntType, UntypedFloatType:
		return true
	}
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

func isAddressable(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.IdentLiteral, *ast.MemberExpr, *ast.IndexExpr:
		return true
	default:
		return false
	}
}
