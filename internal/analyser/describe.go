package analyser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Gui97p/wisp/internal/ast"
)

func TypeText(t Type) string {
	switch v := t.(type) {
	case *StructType:
		return v.Name
	case PointerType:
		return "*" + TypeText(v.Element)
	case ArrayType:
		return fmt.Sprintf("%s[%d]", TypeText(v.Element), v.Size)
	case SpanType:
		return TypeText(v.Element) + "[]"
	case UntypedIntType:
		return "int"
	case UntypedFloatType:
		return "float"
	}
	return t.String()
}

func FuncSignature(name string, ft *FuncType, decl *ast.FuncDecl) string {
	receiver := ""
	if ft.Receiver != nil {
		receiver = "(" + TypeText(ft.Receiver)
		if decl != nil && decl.Receiver != nil && decl.Receiver.Name != "" {
			receiver += " " + decl.Receiver.Name
		}
		receiver += ") "
	}

	params := make([]string, 0, len(ft.Params)+1)
	for i, p := range ft.Params {
		label := TypeText(p)
		paramName := ""
		if decl != nil && i < len(decl.Params) {
			paramName = decl.Params[i].Name
		}
		if _, boxed := p.(BoxedType); boxed && ft.Variadic && i == len(ft.Params)-1 {
			label = "..." + paramName
			params = append(params, label)
			continue
		}
		if ft.Variadic && i == len(ft.Params)-1 {
			label += " ..."
		}
		if paramName != "" {
			label += " " + paramName
		}
		params = append(params, label)
	}
	if ft.CVariadic {
		params = append(params, "...")
	}

	return fmt.Sprintf("func %s%s(%s)%s", receiver, name, strings.Join(params, ", "), returnsText(ft))
}

func returnsText(ft *FuncType) string {
	var b strings.Builder
	if ft.Fallible {
		b.WriteString(" !")
	}
	switch len(ft.Returns) {
	case 0:
	case 1:
		if !ft.Fallible {
			b.WriteString(" ")
		}
		b.WriteString(TypeText(ft.Returns[0]))
	default:
		if !ft.Fallible {
			b.WriteString(" ")
		}
		parts := make([]string, len(ft.Returns))
		for i, r := range ft.Returns {
			parts[i] = TypeText(r)
		}
		b.WriteString("(")
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(")")
	}
	return b.String()
}

func EnumText(t NamedType) string {
	var b strings.Builder
	fmt.Fprintf(&b, "enum %s {\n", t.Name)
	for _, name := range t.MemberOrder {
		fmt.Fprintf(&b, "  %s = %d,\n", name, t.Members[name])
	}
	b.WriteString("}")
	return b.String()
}

func StructText(st *StructType) string {
	var b strings.Builder
	fmt.Fprintf(&b, "struct %s {\n", st.Name)
	for _, field := range st.Order {
		fmt.Fprintf(&b, "  %s %s\n", TypeText(st.Fields[field]), field)
	}
	b.WriteString("}")

	names := make([]string, 0, len(st.Methods))
	for name := range st.Methods {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		b.WriteString("\n")
		for _, name := range names {
			fmt.Fprintf(&b, "\n%s", FuncSignature(name, st.Methods[name], nil))
		}
	}

	return b.String()
}
